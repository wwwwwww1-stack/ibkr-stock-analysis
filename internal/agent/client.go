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
	return strings.Join([]string{
		"You are analyzing supplied market data, not editing code.",
		"This is not financial advice; do not claim that it is financial advice or a recommendation.",
		"Do not claim to access accounts, portfolio data, positions, balances, or live brokerage state.",
		"Never request, store, or mention needing IBKR credentials.",
		"Do not run tools, do not inspect accounts, do not place or prepare orders.",
		"Use only supplied market data and supplied PriceAction knowledge excerpts.",
		"Do not browse the repository or read priceaction files yourself.",
		"For neutral direction, omit or null directional fields such as entry_zone, stop_loss, and risk_reward.",
		"For long or short direction, include entry_zone, stop_loss, take_profit, and positive risk_reward.",
		"Return exactly one JSON object matching the schema.",
		fmt.Sprintf("Use generated_at exactly as: %s", generatedAt.UTC().Format(time.RFC3339)),
		"PriceAction knowledge excerpts:",
		knowledgePayload,
		"Input market data:",
		string(payload),
	}, "\n\n"), nil
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
  "required": ["direction", "entry_zone", "stop_loss", "take_profit", "risk_reward", "confidence", "summary", "price_action", "invalidated_if", "generated_at"],
  "properties": {
    "direction": {
      "type": "string",
      "enum": ["long", "short", "neutral"]
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
    }
  }
}`
