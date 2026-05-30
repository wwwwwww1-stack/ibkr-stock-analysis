package app

import (
	"context"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/agent"
	"ibkr-stock-analysis/internal/domain"
	"ibkr-stock-analysis/internal/market"
	"ibkr-stock-analysis/internal/storage"
)

func TestServiceLoadsInitialState(t *testing.T) {
	service := newTestService(t)

	state, err := service.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}

	if state.Settings.IBKRHost != "127.0.0.1" {
		t.Fatalf("host = %q, want default", state.Settings.IBKRHost)
	}
	if state.ConnectionStatus != domain.ConnectionDisconnected {
		t.Fatalf("connection = %q, want disconnected", state.ConnectionStatus)
	}
}

func TestServiceSavesSettingsAndNormalizesWatchlist(t *testing.T) {
	service := newTestService(t)

	state, err := service.SaveSettings(context.Background(), domain.Settings{
		IBKRHost:          "localhost",
		IBKRPort:          4002,
		IBKRClientID:      7,
		Watchlist:         []string{"nvda", "AAPL", "nvda"},
		SelectedTimeframe: domain.Timeframe15m,
	})
	if err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	if len(state.Symbols) != 2 || state.Symbols[0].Symbol != "NVDA" || state.Symbols[1].Symbol != "AAPL" {
		t.Fatalf("symbols = %#v", state.Symbols)
	}
}

func TestServiceConnectAndDisconnectIBKR(t *testing.T) {
	service := newTestService(t)

	connected, err := service.ConnectIBKR(context.Background())
	if err != nil {
		t.Fatalf("ConnectIBKR returned error: %v", err)
	}
	if connected.ConnectionStatus != domain.ConnectionConnected {
		t.Fatalf("connection = %q, want connected", connected.ConnectionStatus)
	}
	disconnected, err := service.DisconnectIBKR(context.Background())
	if err != nil {
		t.Fatalf("DisconnectIBKR returned error: %v", err)
	}
	if disconnected.ConnectionStatus != domain.ConnectionDisconnected {
		t.Fatalf("connection = %q, want disconnected", disconnected.ConnectionStatus)
	}
}

func TestServiceRunAnalysisNowUsesBarsAndAgent(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA"}
	settings.SelectedTimeframe = domain.Timeframe5m
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: now.Add(-time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: now, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	mockAgent := service.agentClient.(*agent.MockClient)
	mockAgent.SetOutput("NVDA", validServiceOutput())

	state, err := service.RunAnalysisNow(context.Background())
	if err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}

	if len(state.Symbols) != 1 || state.Symbols[0].JobStatus != domain.JobStatusComplete {
		t.Fatalf("symbols = %#v, want complete NVDA", state.Symbols)
	}
	if state.Symbols[0].Result == nil || state.Symbols[0].Result.Output.Direction != domain.DirectionLong {
		t.Fatalf("result = %#v, want long", state.Symbols[0].Result)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	store := storage.NewStore(t.TempDir()+"/settings.json", 20)
	mockAgent := agent.NewMockClient()
	return NewService(store, market.NewMockProvider(), mockAgent, nil)
}

func validServiceOutput() domain.AgentOutput {
	stop := 124.2
	riskReward := 2.0
	return domain.AgentOutput{
		Direction:     domain.DirectionLong,
		EntryZone:     &domain.EntryZone{Low: 125.1, High: 125.5},
		StopLoss:      &stop,
		TakeProfit:    []float64{126.4},
		RiskReward:    &riskReward,
		Confidence:    0.7,
		Summary:       "Held support.",
		PriceAction:   []string{"Higher low"},
		InvalidatedIf: "Close below 124.2",
		GeneratedAt:   time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC),
	}
}
