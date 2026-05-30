package agent

import (
	"context"
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
		Direction:     domain.DirectionLong,
		EntryZone:     &domain.EntryZone{Low: 10, High: 11},
		StopLoss:      ptr(9.5),
		TakeProfit:    []float64{12},
		RiskReward:    &riskReward,
		Confidence:    0.7,
		Summary:       "Held support.",
		PriceAction:   []string{"Higher low"},
		InvalidatedIf: "Close below 9.5",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC),
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

	prompt, err := BuildCodexPrompt(input, generatedAt)
	if err != nil {
		t.Fatalf("BuildCodexPrompt returned error: %v", err)
	}

	assertContains(t, prompt, "You are analyzing supplied market data, not editing code.")
	assertContains(t, prompt, "Do not run tools, do not inspect accounts, do not place or prepare orders.")
	assertContains(t, prompt, "Return exactly one JSON object matching the schema.")
	assertContains(t, prompt, "Use generated_at exactly as: 2026-05-30T10:05:00Z")
	assertContains(t, prompt, `"symbol": "NVDA"`)
	assertContains(t, prompt, `"timeframe": "5m"`)
}

func TestCodexClientInvokesCodexExecAndParsesValidOutput(t *testing.T) {
	fake := createFakeCodex(t, `printf '%s\n' "$@" > "$RECORD_FILE"
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
{"direction":"neutral","take_profit":[],"confidence":0.42,"summary":"Range-bound action.","price_action":["Holding inside prior range"],"invalidated_if":"Breakout from the range.","generated_at":"2026-05-30T10:05:00Z"}
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
	assertContains(t, args, "--skip-git-repo-check")
	assertContains(t, args, "--output-schema")
	assertContains(t, args, "--output-last-message")
	assertContains(t, args, "-")

	prompt := readRecordedFile(t, "PROMPT_FILE")
	assertContains(t, prompt, `"symbol": "NVDA"`)
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
{"direction":"neutral","take_profit":[],"confidence":0.42,"summary":"Range-bound action.","price_action":["Holding inside prior range"],"invalidated_if":"Breakout from the range.","generated_at":"2026-05-30T10:05:00Z"}
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
	t.Setenv("RECORD_FILE", recordFile)
	t.Setenv("PROMPT_FILE", promptFile)
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

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
