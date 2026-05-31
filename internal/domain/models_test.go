package domain

import (
	"encoding/json"
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
		Direction:        DirectionLong,
		SetupQuality:     SetupQualityAPlus,
		EntryZone:        &EntryZone{Low: entryLow, High: entryHigh},
		StopLoss:         &stop,
		TakeProfit:       []float64{126.4, 127.2},
		RiskReward:       &riskReward,
		Confidence:       0.68,
		MarketRegime:     "趋势回踩",
		TradeThesis:      "价格收复前高后在支撑上方形成更高低点。",
		Counterargument:  "上方仍有前高供应，若突破失败容易回到区间。",
		NoTradeReason:    "",
		RejectionReasons: []string{},
		Summary:          "Price reclaimed the prior high and held above support.",
		PriceAction:      []string{"Higher low formed", "Breakout retest held"},
		InvalidatedIf:    "A 5m candle closes below 124.2",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	if err := output.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateAgentOutputAllowsNeutralWithoutTradeLevels(t *testing.T) {
	output := AgentOutput{
		Direction:        DirectionNeutral,
		SetupQuality:     SetupQualityNone,
		TakeProfit:       []float64{},
		Confidence:       0.31,
		MarketRegime:     "震荡",
		TradeThesis:      "",
		Counterargument:  "区间两边都没有被价格行为否定。",
		NoTradeReason:    "价格在区间中部，没有 A+ 触发和足够空间。",
		RejectionReasons: []string{"区间中部", "空间不足"},
		Summary:          "Price is balanced inside the recent range.",
		PriceAction:      []string{"Range remains intact"},
		InvalidatedIf:    "A close outside the range changes the setup.",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	if err := output.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateAgentOutputRejectsInvalidSetupQuality(t *testing.T) {
	output := AgentOutput{
		Direction:        DirectionNeutral,
		SetupQuality:     SetupQuality("maybe"),
		TakeProfit:       []float64{},
		Confidence:       0.31,
		MarketRegime:     "混乱",
		Counterargument:  "方向不清晰。",
		NoTradeReason:    "没有清晰触发。",
		RejectionReasons: []string{"方向不清晰"},
		Summary:          "No trade.",
		PriceAction:      []string{"Mixed wicks"},
		InvalidatedIf:    "A clean breakout forms.",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	err := output.Validate()
	if err == nil || !strings.Contains(err.Error(), "setup_quality") {
		t.Fatalf("err = %v, want setup_quality validation error", err)
	}
}

func TestValidateAgentOutputRejectsAPlusNeutral(t *testing.T) {
	output := AgentOutput{
		Direction:        DirectionNeutral,
		SetupQuality:     SetupQualityAPlus,
		TakeProfit:       []float64{},
		Confidence:       0.31,
		MarketRegime:     "震荡",
		Counterargument:  "区间还没有被打破。",
		NoTradeReason:    "没有交易触发。",
		RejectionReasons: []string{"无触发"},
		Summary:          "No trade.",
		PriceAction:      []string{"Balanced range"},
		InvalidatedIf:    "A clean breakout forms.",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
	}

	err := output.Validate()
	if err == nil || !strings.Contains(err.Error(), "a_plus") {
		t.Fatalf("err = %v, want a_plus neutral validation error", err)
	}
}

func TestValidateAgentOutputRejectsInvalidRisk(t *testing.T) {
	entryLow := 125.5
	entryHigh := 125.1
	stop := 124.2
	riskReward := -1.0
	output := AgentOutput{
		Direction:        DirectionLong,
		SetupQuality:     SetupQualityAPlus,
		EntryZone:        &EntryZone{Low: entryLow, High: entryHigh},
		StopLoss:         &stop,
		TakeProfit:       []float64{126.4},
		RiskReward:       &riskReward,
		Confidence:       1.2,
		MarketRegime:     "趋势",
		TradeThesis:      "Invalid",
		Counterargument:  "Invalid",
		RejectionReasons: []string{},
		Summary:          "Invalid",
		GeneratedAt:      time.Date(2026, 5, 30, 18, 36, 10, 0, time.UTC),
		PriceAction:      []string{"Invalid"},
		InvalidatedIf:    "Invalid",
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

func TestAgentInputMarshalOmitsChartImagePath(t *testing.T) {
	capturedAt := time.Date(2026, 5, 30, 19, 15, 0, 0, time.UTC)
	input := AgentInput{
		Symbol:       "NVDA",
		Timeframe:    Timeframe5m,
		CurrentPrice: 125.5,
		ChartImage: &ChartImageInput{
			Provided:   true,
			Source:     "desktop_region",
			MimeType:   "image/png",
			CapturedAt: &capturedAt,
			Path:       "/tmp/private-chart.png",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	encoded := string(data)
	for _, fragment := range []string{`"chart_image"`, `"provided":true`, `"source":"desktop_region"`, `"mime_type":"image/png"`} {
		if !strings.Contains(encoded, fragment) {
			t.Fatalf("encoded input = %s, want fragment %s", encoded, fragment)
		}
	}
	if strings.Contains(encoded, "/tmp/private-chart.png") || strings.Contains(encoded, `"path"`) {
		t.Fatalf("encoded input = %s, chart image path must not be serialized", encoded)
	}
}
