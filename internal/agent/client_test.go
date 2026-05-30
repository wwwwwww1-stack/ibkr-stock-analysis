package agent

import (
	"context"
	"errors"
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

func TestProcessClientRejectsInvalidWorkerJSON(t *testing.T) {
	response := []byte(`{"ok":true,"result":{"direction":"long","confidence":2}}` + "\n")

	_, err := DecodeResponse(response)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProcessClientDecodesWorkerError(t *testing.T) {
	response := []byte(`{"ok":false,"error":"AI Parse Error"}` + "\n")

	_, err := DecodeResponse(response)
	if err == nil || err.Error() != "AI Parse Error" {
		t.Fatalf("err = %v, want AI Parse Error", err)
	}
}

func ptr(v float64) *float64 { return &v }
