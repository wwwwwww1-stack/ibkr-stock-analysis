package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestMockClientReturnsConfiguredOutput(t *testing.T) {
	client := NewMockClient()
	riskReward := 2.0
	output := domain.AgentOutput{
		Direction:        domain.DirectionLong,
		SetupQuality:     domain.SetupQualityAPlus,
		EntryZone:        &domain.EntryZone{Low: 10, High: 11},
		StopLoss:         ptr(9.5),
		TakeProfit:       []float64{12},
		RiskReward:       &riskReward,
		Confidence:       0.7,
		MarketRegime:     "趋势回踩",
		TradeThesis:      "回踩后买盘重新接住。",
		Counterargument:  "如果跌回前低，突破可能失败。",
		NoTradeReason:    "",
		RejectionReasons: []string{},
		Summary:          "Held support.",
		PriceAction:      []string{"Higher low"},
		InvalidatedIf:    "Close below 9.5",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC),
	}
	client.SetOutput("NVDA", output)

	got, err := client.Analyze(context.Background(), domain.AgentInput{Symbol: "nvda"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got.Direction != domain.DirectionLong {
		t.Fatalf("direction = %q", got.Direction)
	}
}

func TestMockClientReturnsConfiguredError(t *testing.T) {
	client := NewMockClient()
	client.SetError("NVDA", errors.New("rate limited"))

	_, err := client.Analyze(context.Background(), domain.AgentInput{Symbol: "NVDA"})
	if err == nil || err.Error() != "rate limited" {
		t.Fatalf("err = %v, want rate limited", err)
	}
}

func TestBuildCodexPromptIncludesMarketDataAndSafetyInstructions(t *testing.T) {
	input := validAgentInput()
	generatedAt := time.Date(2026, 5, 30, 10, 5, 0, 0, time.UTC)
	snippets := []KnowledgeSnippet{
		{
			Source: "priceaction/趋势 1.md",
			Title:  "核心观点",
			Text:   "牛市趋势的关键是更高的低点，熊市趋势的关键是更低的高点。",
		},
	}

	prompt, err := BuildCodexPrompt(input, generatedAt, snippets)
	if err != nil {
		t.Fatalf("BuildCodexPrompt returned error: %v", err)
	}

	assertContains(t, prompt, "You are analyzing supplied market data, not editing code.")
	assertContains(t, prompt, "Do not run tools, do not inspect accounts, do not place or prepare orders.")
	assertContains(t, prompt, "Use only supplied market data and supplied PriceAction knowledge excerpts.")
	assertContains(t, prompt, "Do not browse the repository or read priceaction files yourself.")
	assertContains(t, prompt, "Write all natural-language output fields in Simplified Chinese")
	assertContains(t, prompt, "Default to neutral/no-trade unless the setup is clearly A+.")
	assertContains(t, prompt, `Only output setup_quality "a_plus"`)
	assertContains(t, prompt, "market_regime")
	assertContains(t, prompt, "counterargument")
	assertContains(t, prompt, "no_trade_reason")
	assertContains(t, prompt, "Return exactly one JSON object matching the schema.")
	assertContains(t, prompt, "For long direction, stop_loss must be below both current_price and entry_zone.low, and every take_profit must be above both current_price and entry_zone.high.")
	assertContains(t, prompt, "For short direction, stop_loss must be above both current_price and entry_zone.high, and every take_profit must be below both current_price and entry_zone.low.")
	assertContains(t, prompt, "If directional levels cannot satisfy those price-side rules, return neutral.")
	assertContains(t, prompt, "Use generated_at exactly as: 2026-05-30T10:05:00Z")
	assertContains(t, prompt, "PriceAction knowledge excerpts:")
	assertContains(t, prompt, `"source": "priceaction/趋势 1.md"`)
	assertContains(t, prompt, "更高的低点")
	assertContains(t, prompt, `"symbol": "NVDA"`)
	assertContains(t, prompt, `"timeframe": "5m"`)
}

func TestBuildCodexPromptIncludesMultiTimeframeContextInstructions(t *testing.T) {
	input := validAgentInput()
	currentPrice := 128.0
	input.MultiTimeframeContext = []domain.TimeframeContext{
		{
			Timeframe:    domain.Timeframe15m,
			Available:    true,
			CurrentPrice: &currentPrice,
			Bars: []domain.Bar{
				{
					Time:   time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC),
					Open:   126,
					High:   129,
					Low:    125,
					Close:  128,
					Volume: 360000,
				},
			},
			Derived: &domain.DerivedFeatures{
				SessionHigh:              129,
				SessionLow:               121,
				RecentSwingHighs:         []float64{129},
				RecentSwingLows:          []float64{121},
				ATR:                      2.4,
				VolumeContext:            "above_average",
				LastCloseRelativeToRange: "near_high",
			},
		},
		{
			Timeframe: domain.Timeframe1h,
			Available: false,
			Error:     "no data for NVDA 1h",
		},
	}
	generatedAt := time.Date(2026, 5, 30, 10, 5, 0, 0, time.UTC)

	prompt, err := BuildCodexPrompt(input, generatedAt, nil)
	if err != nil {
		t.Fatalf("BuildCodexPrompt returned error: %v", err)
	}

	assertContains(t, prompt, "Use the top-level timeframe and bars as the primary analysis timeframe.")
	assertContains(t, prompt, "Use multi_timeframe_context for 15m and 1h resonance, conflicts, and support/resistance context.")
	assertContains(t, prompt, "Do not invent unavailable higher-timeframe data.")
	assertContains(t, prompt, `"multi_timeframe_context"`)
	assertContains(t, prompt, `"timeframe": "15m"`)
	assertContains(t, prompt, `"timeframe": "1h"`)
	assertContains(t, prompt, `"available": false`)
	assertContains(t, prompt, "no data for NVDA 1h")
}

func TestBuildCodexPromptIncludesChartImageInstructionsAndMetadata(t *testing.T) {
	input := validAgentInput()
	capturedAt := time.Date(2026, 5, 30, 10, 4, 0, 0, time.UTC)
	input.ChartImage = &domain.ChartImageInput{
		Provided:   true,
		Source:     "desktop_region",
		MimeType:   "image/png",
		CapturedAt: &capturedAt,
		Path:       "/tmp/private-chart.png",
	}
	generatedAt := time.Date(2026, 5, 30, 10, 5, 0, 0, time.UTC)

	prompt, err := BuildCodexPrompt(input, generatedAt, nil)
	if err != nil {
		t.Fatalf("BuildCodexPrompt returned error: %v", err)
	}

	assertContains(t, prompt, "Use the attached chart screenshot only as visual K-line context.")
	assertContains(t, prompt, "Use supplied IBKR OHLCV data as the source of truth for exact prices and time.")
	assertContains(t, prompt, "If the screenshot is unclear or does not match the supplied symbol/timeframe, say so in price_action and rely on supplied market data.")
	assertContains(t, prompt, `"chart_image"`)
	assertContains(t, prompt, `"source": "desktop_region"`)
	assertContains(t, prompt, `"mime_type": "image/png"`)
	if strings.Contains(prompt, "/tmp/private-chart.png") {
		t.Fatalf("prompt leaked local image path: %s", prompt)
	}
}

func TestCodexClientInvokesCodexExecAndParsesValidOutput(t *testing.T) {
	fake := createFakeCodex(t, `printf '%s\n' "$@" > "$RECORD_FILE"
pwd > "$CWD_FILE"
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
cat > "$PROMPT_FILE"
cat > "$out" <<'JSON'
{"direction":"neutral","setup_quality":"none","entry_zone":null,"stop_loss":null,"take_profit":[],"risk_reward":null,"confidence":0.42,"market_regime":"震荡","trade_thesis":"","counterargument":"区间两端都没有被否定。","no_trade_reason":"价格在区间中部，没有 A+ 触发。","rejection_reasons":["区间中部"],"summary":"Range-bound action.","price_action":["Holding inside prior range"],"invalidated_if":"Breakout from the range.","generated_at":"2026-05-30T10:05:00Z"}
JSON
`)
	client := &CodexClient{
		Command: fake,
		Now:     func() time.Time { return time.Date(2026, 5, 30, 10, 5, 0, 0, time.UTC) },
	}

	got, err := client.Analyze(context.Background(), validAgentInput())
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got.Direction != domain.DirectionNeutral {
		t.Fatalf("direction = %q, want neutral", got.Direction)
	}
	args := readRecordedFile(t, "RECORD_FILE")
	assertContains(t, args, "exec")
	assertContains(t, args, "--sandbox\nread-only")
	assertContains(t, args, "--ephemeral")
	assertContains(t, args, "--ignore-rules")
	assertContains(t, args, "--ignore-user-config")
	assertContains(t, args, "--skip-git-repo-check")
	assertContains(t, args, "--output-schema")
	assertContains(t, args, "--output-last-message")
	assertContains(t, args, "-")

	prompt := readRecordedFile(t, "PROMPT_FILE")
	assertContains(t, prompt, `"symbol": "NVDA"`)
	assertContains(t, prompt, "PriceAction knowledge excerpts:")
	assertContains(t, prompt, "交易区间")
	cwd := readRecordedFile(t, "CWD_FILE")
	if strings.TrimSpace(cwd) == "" || strings.TrimSpace(cwd) == currentWorkingDir(t) {
		t.Fatalf("codex cwd = %q, want isolated temp dir", cwd)
	}
}

func TestCodexClientAttachesChartImage(t *testing.T) {
	fake := createFakeCodex(t, `printf '%s\n' "$@" > "$RECORD_FILE"
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
cat > "$out" <<'JSON'
{"direction":"neutral","setup_quality":"none","entry_zone":null,"stop_loss":null,"take_profit":[],"risk_reward":null,"confidence":0.42,"market_regime":"震荡","trade_thesis":"","counterargument":"区间两端都没有被否定。","no_trade_reason":"价格在区间中部，没有 A+ 触发。","rejection_reasons":["区间中部"],"summary":"Range-bound action.","price_action":["Holding inside prior range"],"invalidated_if":"Breakout from the range.","generated_at":"2026-05-30T10:05:00Z"}
JSON
`)
	input := validAgentInput()
	input.ChartImage = &domain.ChartImageInput{Provided: true, Source: "desktop_region", MimeType: "image/png", Path: "/tmp/chart.png"}
	client := &CodexClient{
		Command: fake,
		Now:     func() time.Time { return time.Date(2026, 5, 30, 10, 5, 0, 0, time.UTC) },
	}

	if _, err := client.Analyze(context.Background(), input); err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	args := readRecordedFile(t, "RECORD_FILE")
	assertContains(t, args, "--image\n/tmp/chart.png")
}

func TestCodexClientUsesConfiguredModel(t *testing.T) {
	fake := createFakeCodex(t, `printf '%s\n' "$@" > "$RECORD_FILE"
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
cat > "$out" <<'JSON'
{"direction":"neutral","setup_quality":"none","entry_zone":null,"stop_loss":null,"take_profit":[],"risk_reward":null,"confidence":0.42,"market_regime":"震荡","trade_thesis":"","counterargument":"区间两端都没有被否定。","no_trade_reason":"价格在区间中部，没有 A+ 触发。","rejection_reasons":["区间中部"],"summary":"Range-bound action.","price_action":["Holding inside prior range"],"invalidated_if":"Breakout from the range.","generated_at":"2026-05-30T10:05:00Z"}
JSON
`)
	client := &CodexClient{Command: fake, Model: "gpt-test"}

	if _, err := client.Analyze(context.Background(), validAgentInput()); err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	args := readRecordedFile(t, "RECORD_FILE")
	assertContains(t, args, "--model\ngpt-test")
}

func TestCodexClientRejectsInvalidOutput(t *testing.T) {
	fake := createFakeCodex(t, `out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
cat > "$out" <<'JSON'
{"direction":"long","confidence":2}
JSON
`)
	client := &CodexClient{Command: fake}

	_, err := client.Analyze(context.Background(), validAgentInput())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCodexClientReportsMissingCodexCLI(t *testing.T) {
	client := &CodexClient{Command: filepath.Join(t.TempDir(), "missing-codex")}

	_, err := client.Analyze(context.Background(), validAgentInput())
	if err == nil || !strings.Contains(err.Error(), "Codex CLI not found") {
		t.Fatalf("err = %v, want Codex CLI not found", err)
	}
}

func TestCodexClientReportsKnowledgeBaseError(t *testing.T) {
	client := &CodexClient{
		Command:   filepath.Join(t.TempDir(), "missing-codex"),
		Knowledge: NewMarkdownKnowledgeProvider(os.DirFS(filepath.Join(t.TempDir(), "missing-kb"))),
	}

	_, err := client.Analyze(context.Background(), validAgentInput())
	if err == nil || !strings.Contains(err.Error(), "PriceAction knowledge base unavailable") {
		t.Fatalf("err = %v, want PriceAction knowledge base unavailable", err)
	}
}

func TestAgentOutputSchemaRequiresEveryPropertyForCodexStrictSchema(t *testing.T) {
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(agentOutputJSONSchema), &schema); err != nil {
		t.Fatalf("agentOutputJSONSchema is invalid JSON: %v", err)
	}

	required := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = true
	}
	for name := range schema.Properties {
		if !required[name] {
			t.Fatalf("schema property %q is not listed in required; Codex strict schema requires every property to be required and nullable when optional", name)
		}
	}
}

func TestAgentOutputSchemaIncludesAPlusTraderReviewFields(t *testing.T) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(agentOutputJSONSchema), &schema); err != nil {
		t.Fatalf("agentOutputJSONSchema is invalid JSON: %v", err)
	}

	for _, name := range []string{"setup_quality", "market_regime", "trade_thesis", "counterargument", "no_trade_reason", "rejection_reasons"} {
		if _, ok := schema.Properties[name]; !ok {
			t.Fatalf("schema is missing %q", name)
		}
	}
	assertContains(t, string(schema.Properties["setup_quality"]), `"a_plus"`)
	assertContains(t, string(schema.Properties["setup_quality"]), `"none"`)
}

func ptr(v float64) *float64 { return &v }

func validAgentInput() domain.AgentInput {
	return domain.AgentInput{
		Symbol:       "NVDA",
		Timeframe:    domain.Timeframe5m,
		CurrentPrice: 125.5,
		Bars: []domain.Bar{
			{
				Time:   time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC),
				Open:   124,
				High:   126,
				Low:    123.5,
				Close:  125.5,
				Volume: 120000,
			},
		},
		Derived: domain.DerivedFeatures{
			SessionHigh:              126,
			SessionLow:               123.5,
			RecentSwingHighs:         []float64{126},
			RecentSwingLows:          []float64{123.5},
			ATR:                      1.2,
			VolumeContext:            "normal",
			LastCloseRelativeToRange: "upper",
		},
	}
}

func createFakeCodex(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake codex is unix-only")
	}
	dir := t.TempDir()
	recordFile := filepath.Join(dir, "args.txt")
	promptFile := filepath.Join(dir, "prompt.txt")
	cwdFile := filepath.Join(dir, "cwd.txt")
	t.Setenv("RECORD_FILE", recordFile)
	t.Setenv("PROMPT_FILE", promptFile)
	t.Setenv("CWD_FILE", cwdFile)
	path := filepath.Join(dir, "codex")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func readRecordedFile(t *testing.T, envName string) string {
	t.Helper()
	data, err := os.ReadFile(os.Getenv(envName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func currentWorkingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
