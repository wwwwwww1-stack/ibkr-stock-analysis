package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

type Client interface {
	Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error)
}

type MockClient struct {
	mu      sync.Mutex
	outputs map[string]domain.AgentOutput
	errors  map[string]error
}

func NewMockClient() *MockClient {
	return &MockClient{
		outputs: make(map[string]domain.AgentOutput),
		errors:  make(map[string]error),
	}
}

func (m *MockClient) SetOutput(symbol string, output domain.AgentOutput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[strings.ToUpper(strings.TrimSpace(symbol))] = output
}

func (m *MockClient) SetError(symbol string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[strings.ToUpper(strings.TrimSpace(symbol))] = err
}

func (m *MockClient) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.errors[symbol]; ok {
		return domain.AgentOutput{}, err
	}
	output, ok := m.outputs[symbol]
	if !ok {
		return domain.AgentOutput{}, fmt.Errorf("no mock output for %s", symbol)
	}
	return output, nil
}

type CodexClient struct {
	Command   string
	Model     string
	Knowledge KnowledgeProvider
	Now       func() time.Time
}

func NewCodexClient() *CodexClient {
	return &CodexClient{
		Command:   "codex",
		Model:     strings.TrimSpace(os.Getenv("CODEX_MODEL")),
		Knowledge: NewDefaultKnowledgeProvider(),
	}
}

func (c *CodexClient) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	knowledge := c.Knowledge
	if knowledge == nil {
		knowledge = NewDefaultKnowledgeProvider()
	}
	snippets, err := knowledge.Snippets(input)
	if err != nil {
		return domain.AgentOutput{}, err
	}
	prompt, err := BuildCodexPrompt(input, now, snippets)
	if err != nil {
		return domain.AgentOutput{}, err
	}
	tmpDir, err := os.MkdirTemp("", "ibkr-codex-analysis-*")
	if err != nil {
		return domain.AgentOutput{}, err
	}
	defer os.RemoveAll(tmpDir)

	schemaPath := filepath.Join(tmpDir, "agent-output.schema.json")
	outputPath := filepath.Join(tmpDir, "agent-output.json")
	if err := os.WriteFile(schemaPath, []byte(agentOutputJSONSchema), 0o600); err != nil {
		return domain.AgentOutput{}, err
	}

	command := strings.TrimSpace(c.Command)
	if command == "" {
		command = "codex"
	}
	args := []string{
		"exec",
		"--sandbox", "read-only",
		"--ephemeral",
		"--ignore-rules",
		"--ignore-user-config",
		"--color", "never",
		"--skip-git-repo-check",
		"--output-schema", schemaPath,
		"--output-last-message", outputPath,
	}
	if model := strings.TrimSpace(c.Model); model != "" {
		args = append(args, "--model", model)
	}
	if input.ChartImage != nil && strings.TrimSpace(input.ChartImage.Path) != "" {
		args = append(args, "--image", strings.TrimSpace(input.ChartImage.Path))
	}
	args = append(args, "-")

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(prompt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) || strings.Contains(err.Error(), "no such file") {
			return domain.AgentOutput{}, fmt.Errorf("Codex CLI not found: install and authenticate codex before running real analysis")
		}
		details := strings.TrimSpace(stderr.String())
		if details == "" {
			details = strings.TrimSpace(stdout.String())
		}
		if details != "" {
			return domain.AgentOutput{}, fmt.Errorf("Codex CLI analysis failed: %s", details)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.AgentOutput{}, ctxErr
		}
		return domain.AgentOutput{}, fmt.Errorf("Codex CLI analysis failed: %w", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return domain.AgentOutput{}, fmt.Errorf("Codex CLI returned no analysis output: %w", err)
	}
	return decodeCodexOutput(data)
}

func BuildCodexPrompt(input domain.AgentInput, generatedAt time.Time, snippets []KnowledgeSnippet) (string, error) {
	payload, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return "", err
	}
	knowledgePayload, err := knowledgeSnippetsJSON(snippets)
	if err != nil {
		return "", err
	}
	instructions := []string{
		"You are analyzing supplied market data, not editing code.",
		"This is not financial advice; do not claim that it is financial advice or a recommendation.",
		"Never request, store, or mention needing IBKR login secrets.",
		"Do not run tools, do not inspect accounts, do not place or prepare orders.",
		"Do not run tools, inspect orders, browse files, place orders, cancel orders, prepare orders, output order payloads, mention order IDs, or claim execution.",
		"Use only supplied market data and supplied PriceAction knowledge excerpts.",
		"Do not browse the repository or read priceaction files yourself.",
		"Write all natural-language output fields in Simplified Chinese: summary, price_action, invalidated_if, market_regime, trade_thesis, counterargument, no_trade_reason, rejection_reasons, and position_management.management_notes.",
		"Use the top-level timeframe and bars as the primary analysis timeframe.",
		"Use multi_timeframe_context for 15m and 1h resonance, conflicts, and support/resistance context.",
		"Do not invent unavailable higher-timeframe data.",
		"Default to neutral/no-trade unless the setup is clearly A+.",
		"Think like a cautious trader: first classify market_regime as trend, range, choppy, reversal, multi-timeframe conflict, or unclear in Chinese.",
		"Assess whether price is at a good location rather than the middle of a range.",
		"Require a clear trigger, clear invalidation, and enough space to the first target.",
		"State the strongest counterargument against the trade in counterargument.",
		"Only output setup_quality \"a_plus\" when the counterargument is visibly negated by supplied price action.",
		"When uncertainty, range-middle location, weak signal bars, mixed wicks, multi-timeframe conflict, overly wide stop, or insufficient target space exists, output direction neutral and setup_quality none.",
	}
	if input.AccountContext == nil {
		instructions = append(instructions,
			"Do not claim to access accounts, portfolio data, positions, balances, or live brokerage state.",
			"Set position_management to null because no sanitized account context was supplied.",
		)
	} else {
		instructions = append(instructions,
			"Use only the supplied sanitized account snapshot, configured maximum stock trade amount, and sizing envelope.",
			"Do not infer, request, or claim access to any account data beyond the supplied account_context.",
			"Treat all position-management output as advisory and manually reviewed.",
			"Require manual_review_required to be true whenever position_management is not null.",
			"When the market setup is not A+, set position_management.sizing_status to no_trade or not_a_plus and do not suggest new exposure even if account capacity exists.",
			"When the market setup is A+ but advisory_max_shares is zero or advisory_notional_cap_usd is zero, explain the account-blocked condition for manual review.",
			"Use manual-review wording for possible add, trim, hold, or risk-review ideas; never phrase them as app-executed actions.",
		)
	}
	if input.ChartImage != nil && input.ChartImage.Provided {
		instructions = append(instructions,
			"Use the attached chart screenshot only as visual K-line context.",
			"Use supplied IBKR OHLCV data as the source of truth for exact prices and time.",
			"If the screenshot is unclear or does not match the supplied symbol/timeframe, say so in price_action and rely on supplied market data.",
		)
	}
	instructions = append(instructions,
		"For neutral direction, omit or null directional fields such as entry_zone, stop_loss, and risk_reward.",
		"For neutral direction, set setup_quality to none, explain no_trade_reason, and include concise rejection_reasons.",
		"For long or short direction, include entry_zone, stop_loss, take_profit, positive risk_reward, market_regime, trade_thesis, counterargument, and setup_quality.",
		"For long or short direction with setup_quality lower than a_plus, keep the directional levels but explain no_trade_reason and rejection_reasons; the backtest will skip it.",
		"For long direction, stop_loss must be below both current_price and entry_zone.low, and every take_profit must be above both current_price and entry_zone.high.",
		"For short direction, stop_loss must be above both current_price and entry_zone.high, and every take_profit must be below both current_price and entry_zone.low.",
		"If directional levels cannot satisfy those price-side rules, return neutral.",
		"Return exactly one JSON object matching the schema.",
		fmt.Sprintf("Use generated_at exactly as: %s", generatedAt.UTC().Format(time.RFC3339)),
		"PriceAction knowledge excerpts:",
		knowledgePayload,
		"Input market data:",
		string(payload),
	)
	return strings.Join(instructions, "\n\n"), nil
}

func decodeCodexOutput(data []byte) (domain.AgentOutput, error) {
	var output domain.AgentOutput
	if err := json.Unmarshal(bytes.TrimSpace(data), &output); err != nil {
		return domain.AgentOutput{}, err
	}
	if err := output.Validate(); err != nil {
		return domain.AgentOutput{}, err
	}
	return output, nil
}

const agentOutputJSONSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["direction", "setup_quality", "entry_zone", "stop_loss", "take_profit", "risk_reward", "confidence", "market_regime", "trade_thesis", "counterargument", "no_trade_reason", "rejection_reasons", "summary", "price_action", "invalidated_if", "generated_at", "position_management"],
  "properties": {
    "direction": {
      "type": "string",
      "enum": ["long", "short", "neutral"]
    },
    "setup_quality": {
      "type": "string",
      "enum": ["a_plus", "a", "b", "c", "none"]
    },
    "entry_zone": {
      "anyOf": [
        {
          "type": "object",
          "additionalProperties": false,
          "required": ["low", "high"],
          "properties": {
            "low": {"type": "number"},
            "high": {"type": "number"}
          }
        },
        {"type": "null"}
      ]
    },
    "stop_loss": {
      "anyOf": [{"type": "number"}, {"type": "null"}]
    },
    "take_profit": {
      "type": "array",
      "items": {"type": "number"}
    },
    "risk_reward": {
      "anyOf": [{"type": "number"}, {"type": "null"}]
    },
    "confidence": {
      "type": "number",
      "minimum": 0,
      "maximum": 1
    },
    "market_regime": {
      "type": "string",
      "minLength": 1
    },
    "trade_thesis": {
      "type": "string"
    },
    "counterargument": {
      "type": "string",
      "minLength": 1
    },
    "no_trade_reason": {
      "type": "string"
    },
    "rejection_reasons": {
      "type": "array",
      "items": {"type": "string"}
    },
    "summary": {
      "type": "string",
      "minLength": 1
    },
    "price_action": {
      "type": "array",
      "items": {"type": "string"}
    },
    "invalidated_if": {
      "type": "string",
      "minLength": 1
    },
    "generated_at": {
      "type": "string",
      "format": "date-time"
    },
    "position_management": {
      "anyOf": [
        {
          "type": "object",
          "additionalProperties": false,
          "required": ["account_aware", "sizing_status", "advisory_action", "advisory_max_shares", "advisory_notional_cap_usd", "estimated_risk_usd", "existing_exposure_usd", "management_notes", "manual_review_required"],
          "properties": {
            "account_aware": {"type": "boolean"},
            "sizing_status": {
              "type": "string",
              "enum": ["available", "blocked_by_cash", "blocked_by_cap", "existing_position_over_cap", "missing_account_snapshot", "missing_directional_levels", "not_a_plus", "no_trade"]
            },
            "advisory_action": {
              "type": "string",
              "enum": ["no_trade", "watch", "consider_setup", "manage_existing", "review_risk"]
            },
            "advisory_max_shares": {
              "anyOf": [{"type": "integer", "minimum": 0}, {"type": "null"}]
            },
            "advisory_notional_cap_usd": {
              "anyOf": [{"type": "number", "minimum": 0}, {"type": "null"}]
            },
            "estimated_risk_usd": {
              "anyOf": [{"type": "number", "minimum": 0}, {"type": "null"}]
            },
            "existing_exposure_usd": {
              "anyOf": [{"type": "number", "minimum": 0}, {"type": "null"}]
            },
            "management_notes": {
              "type": "array",
              "items": {"type": "string"}
            },
            "manual_review_required": {
              "type": "boolean",
              "const": true
            }
          }
        },
        {"type": "null"}
      ]
    }
  }
}`
