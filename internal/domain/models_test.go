package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParseTimeframeAllowsSupportedValues(t *testing.T) {
	tests := map[string]time.Duration{
		"1m":  time.Minute,
		"5m":  5 * time.Minute,
		"15m": 15 * time.Minute,
		"1h":  time.Hour,
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			timeframe, err := ParseTimeframe(input)
			if err != nil {
				t.Fatalf("ParseTimeframe(%q) returned error: %v", input, err)
			}
			if timeframe.Duration() != expected {
				t.Fatalf("duration = %s, want %s", timeframe.Duration(), expected)
			}
		})
	}
}

func TestParseTimeframeRejectsUnsupportedValues(t *testing.T) {
	_, err := ParseTimeframe("30m")
	if err == nil {
		t.Fatal("expected unsupported timeframe error")
	}
	if !strings.Contains(err.Error(), "unsupported timeframe") {
		t.Fatalf("error = %q, want unsupported timeframe", err.Error())
	}
}

func TestParseWatchlistNormalizesSymbols(t *testing.T) {
	symbols := ParseWatchlist(" nvda, TSLA\n aapl , nvda ,, msft ")

	want := []string{"NVDA", "TSLA", "AAPL", "MSFT"}
	if len(symbols) != len(want) {
		t.Fatalf("symbols = %#v, want %#v", symbols, want)
	}
	for i := range want {
		if symbols[i] != want[i] {
			t.Fatalf("symbols[%d] = %q, want %q", i, symbols[i], want[i])
		}
	}
}

func TestValidateAgentOutputAllowsLongSetup(t *testing.T) {
	entryLow := 125.1
	entryHigh := 125.5
	stop := 124.2
	riskReward := 2.1
	output := AgentOutput{
		Direction:     DirectionLong,
		EntryZone:     &EntryZone{Low: entryLow, High: entryHigh},
		StopLoss:      &stop,
		TakeProfit:    []float64{126.4, 127.2},
		RiskReward:    &riskReward,
		Confidence:    0.68,
		Summary:       "Price reclaimed the prior high and held above support.",
		PriceAction:   []string{"Higher low formed", "Breakout retest held"},
		InvalidatedIf: "A 5m candle closes below 124.2",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	if err := output.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateAgentOutputAllowsNeutralWithoutTradeLevels(t *testing.T) {
	output := AgentOutput{
		Direction:     DirectionNeutral,
		TakeProfit:    []float64{},
		Confidence:    0.31,
		Summary:       "Price is balanced inside the recent range.",
		PriceAction:   []string{"Range remains intact"},
		InvalidatedIf: "A close outside the range changes the setup.",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	if err := output.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateAgentOutputRejectsInvalidRisk(t *testing.T) {
	entryLow := 125.5
	entryHigh := 125.1
	stop := 124.2
	riskReward := -1.0
	output := AgentOutput{
		Direction:     DirectionLong,
		EntryZone:     &EntryZone{Low: entryLow, High: entryHigh},
		StopLoss:      &stop,
		TakeProfit:    []float64{126.4},
		RiskReward:    &riskReward,
		Confidence:    1.2,
		Summary:       "Invalid",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
		PriceAction:   []string{"Invalid"},
		InvalidatedIf: "Invalid",
	}

	err := output.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, fragment := range []string{"confidence", "risk_reward", "entry_zone"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error = %q, want fragment %q", err.Error(), fragment)
		}
	}
}
