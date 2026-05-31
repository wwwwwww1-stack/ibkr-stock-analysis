package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/account"
	"ibkr-stock-analysis/internal/agent"
	"ibkr-stock-analysis/internal/capture"
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

func TestServiceRefreshAccountSnapshotSuccess(t *testing.T) {
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 12500,
		BuyingPowerUSD:   25000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		Positions:        []domain.AccountPosition{{Symbol: "NVDA", Quantity: 10, AverageCost: 100, MarketValueUSD: 1000}},
	}
	var events []domain.AppState
	store := storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20)
	service := NewService(store, market.NewMockProvider(), agent.NewMockClient(), func(ctx context.Context, name string, payload any) {
		if name != "account:update" {
			return
		}
		state, ok := payload.(domain.AppState)
		if !ok {
			t.Fatalf("payload = %T, want domain.AppState", payload)
		}
		events = append(events, state)
	}, account.NewMockProvider(snapshot, nil))

	state, err := service.RefreshAccountSnapshot(context.Background())
	if err != nil {
		t.Fatalf("RefreshAccountSnapshot returned error: %v", err)
	}
	if state.AccountSnapshot.Status != domain.AccountSnapshotReady {
		t.Fatalf("account status = %q, want ready", state.AccountSnapshot.Status)
	}
	if state.AccountSnapshot.Snapshot == nil || state.AccountSnapshot.Snapshot.AvailableCashUSD != 12500 {
		t.Fatalf("account snapshot = %#v, want cash 12500", state.AccountSnapshot.Snapshot)
	}
	if len(events) < 2 || events[0].AccountSnapshot.Status != domain.AccountSnapshotLoading || events[len(events)-1].AccountSnapshot.Status != domain.AccountSnapshotReady {
		t.Fatalf("events = %#v, want loading then ready", events)
	}
}

func TestServiceRefreshAccountSnapshotFailure(t *testing.T) {
	service := NewService(
		storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20),
		market.NewMockProvider(),
		agent.NewMockClient(),
		nil,
		account.NewMockProvider(domain.AccountSnapshot{}, errors.New("account data unavailable")),
	)

	state, err := service.RefreshAccountSnapshot(context.Background())
	if err == nil || !strings.Contains(err.Error(), "account data unavailable") {
		t.Fatalf("err = %v, want account data unavailable", err)
	}
	if state.AccountSnapshot.Status != domain.AccountSnapshotFailed {
		t.Fatalf("account status = %q, want failed", state.AccountSnapshot.Status)
	}
	if !strings.Contains(state.AccountSnapshot.Error, "account data unavailable") {
		t.Fatalf("account error = %q, want scoped provider error", state.AccountSnapshot.Error)
	}
}

func TestServiceMarksAccountSnapshotStaleOnDisconnect(t *testing.T) {
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 12000,
		BuyingPowerUSD:   22000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
	}
	service := NewService(
		storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20),
		market.NewMockProvider(),
		agent.NewMockClient(),
		nil,
		account.NewMockProvider(snapshot, nil),
	)
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RefreshAccountSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}

	state, err := service.DisconnectIBKR(context.Background())
	if err != nil {
		t.Fatalf("DisconnectIBKR returned error: %v", err)
	}
	if state.AccountSnapshot.Status != domain.AccountSnapshotStale {
		t.Fatalf("account status = %q, want stale", state.AccountSnapshot.Status)
	}
}

func TestServiceMarksOldAccountSnapshotStale(t *testing.T) {
	base := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 12000,
		BuyingPowerUSD:   22000,
		SnapshotAt:       base,
	}
	service := NewService(
		storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20),
		market.NewMockProvider(),
		agent.NewMockClient(),
		nil,
		account.NewMockProvider(snapshot, nil),
	)
	service.now = func() time.Time { return base.Add(6 * time.Minute) }
	if _, err := service.RefreshAccountSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}

	state, err := service.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state.AccountSnapshot.Status != domain.AccountSnapshotStale {
		t.Fatalf("account status = %q, want stale", state.AccountSnapshot.Status)
	}
}

func TestServiceSaveSettingsPersistsMaxStockTradeAmountOnly(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	store := storage.NewStore(dbPath, 20)
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 12000,
		BuyingPowerUSD:   22000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
	}
	service := NewService(store, market.NewMockProvider(), agent.NewMockClient(), nil, account.NewMockProvider(snapshot, nil))
	if _, err := service.RefreshAccountSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	settings := domain.DefaultSettings()
	amount := 10000.0
	settings.MaxStockTradeAmountUSD = &amount

	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Settings.MaxStockTradeAmountUSD == nil || *loaded.Settings.MaxStockTradeAmountUSD != 10000 {
		t.Fatalf("stored max amount = %#v, want 10000", loaded.Settings.MaxStockTradeAmountUSD)
	}
	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Skipf("sqlite file not directly readable for snapshot scan: %v", err)
	}
	if strings.Contains(string(data), "available_cash_usd") || strings.Contains(string(data), "buying_power_usd") {
		t.Fatalf("store file contains account snapshot data")
	}
}

func TestServiceSaveSettingsRejectsInvalidMaxStockTradeAmount(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	invalid := 0.0
	settings.MaxStockTradeAmountUSD = &invalid

	_, err := service.SaveSettings(context.Background(), settings)
	if err == nil || !strings.Contains(err.Error(), "max_stock_trade_amount_usd") {
		t.Fatalf("err = %v, want max_stock_trade_amount_usd validation error", err)
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

func TestServiceConnectIBKRAutoDetectsTWSLivePort(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "settings.json"), 20)
	provider := &portSelectingProvider{
		successPort: 7496,
		state: market.ProviderState{
			Status: domain.ConnectionDisconnected,
		},
	}
	service := NewService(store, provider, agent.NewMockClient(), nil)

	connected, err := service.ConnectIBKR(context.Background())
	if err != nil {
		t.Fatalf("ConnectIBKR returned error: %v", err)
	}
	if connected.ConnectionStatus != domain.ConnectionConnected {
		t.Fatalf("connection = %q, want connected", connected.ConnectionStatus)
	}
	if got, want := provider.attemptedPorts(), []int{7497, 7496}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("attempted ports = %#v, want %#v", got, want)
	}
	if connected.Settings.IBKRPort != 7496 {
		t.Fatalf("state port = %d, want detected port 7496", connected.Settings.IBKRPort)
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if snapshot.Settings.IBKRPort != 7496 {
		t.Fatalf("stored port = %d, want detected port 7496", snapshot.Settings.IBKRPort)
	}
}

func TestServiceRunAnalysisNowUsesBarsAndAgentForSelectedSymbol(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA", "AAPL"}
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
	service.provider.(*market.MockProvider).SetHistoricalBars("AAPL", domain.Timeframe5m, []domain.Bar{
		{Time: now.Add(-time.Minute), Open: 202, High: 204, Low: 201, Close: 203, Volume: 1000},
		{Time: now, Open: 203, High: 205, Low: 202, Close: 204, Volume: 1400},
	})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.RunAnalysisNow(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}

	if len(recorder.inputs) != 1 || recorder.inputs[0].Symbol != "AAPL" {
		t.Fatalf("agent inputs = %#v, want only AAPL", recorder.inputs)
	}
	if len(state.Symbols) != 2 || state.Symbols[0].JobStatus != domain.JobStatusIdle || state.Symbols[1].JobStatus != domain.JobStatusComplete {
		t.Fatalf("symbols = %#v, want only selected symbol complete", state.Symbols)
	}
	if state.Symbols[1].Result == nil || state.Symbols[1].Result.Output.Direction != domain.DirectionLong {
		t.Fatalf("result = %#v, want long", state.Symbols[1].Result)
	}
}

func TestServiceRunAnalysisNowOmitsAccountContextWhenMaxAmountIsNotConfigured(t *testing.T) {
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
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("agent calls = %d, want 1", len(recorder.inputs))
	}
	if recorder.inputs[0].AccountContext != nil {
		t.Fatalf("account context = %#v, want nil without max stock trade amount", recorder.inputs[0].AccountContext)
	}
}

func TestServiceRunAnalysisNowPassesSanitizedAccountContextWhenConfigured(t *testing.T) {
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 15000,
		BuyingPowerUSD:   30000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		Positions: []domain.AccountPosition{
			{Symbol: "NVDA", Quantity: 10, AverageCost: 100, MarketPrice: 120, MarketValueUSD: 1200, UnrealizedPnLUSD: 200},
			{Symbol: "AAPL", Quantity: 5, AverageCost: 200, MarketPrice: 210, MarketValueUSD: 1050, UnrealizedPnLUSD: 50},
			{Symbol: "MSFT", Quantity: 1, AverageCost: 300, MarketPrice: 320, MarketValueUSD: 320, UnrealizedPnLUSD: 20},
		},
	}
	service := NewService(
		storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20),
		market.NewMockProvider(),
		agent.NewMockClient(),
		nil,
		account.NewMockProvider(snapshot, nil),
	)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA", "AAPL"}
	amount := 5000.0
	settings.MaxStockTradeAmountUSD = &amount
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RefreshAccountSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: now.Add(-time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: now, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("agent calls = %d, want 1", len(recorder.inputs))
	}
	context := recorder.inputs[0].AccountContext
	if context == nil {
		t.Fatal("account context = nil, want sanitized account context")
	}
	if context.AvailableCashUSD != 15000 || context.BuyingPowerUSD != 30000 || context.MaxStockTradeAmountUSD != 5000 {
		t.Fatalf("account context = %#v, want cash/buying power/max", context)
	}
	if len(context.Positions) != 2 {
		t.Fatalf("positions = %#v, want only watchlist positions", context.Positions)
	}
	for _, position := range context.Positions {
		if position.Symbol == "MSFT" {
			t.Fatalf("account context leaked non-watchlist position: %#v", context.Positions)
		}
	}
	if context.SizingEnvelope == nil || context.SizingEnvelope.Symbol != "NVDA" || context.SizingEnvelope.AdvisoryNotionalCapUSD <= 0 {
		t.Fatalf("sizing envelope = %#v, want NVDA advisory envelope", context.SizingEnvelope)
	}
}

func TestServiceRunAnalysisNowStoresCurrentSymbolAccountContextInState(t *testing.T) {
	snapshot := domain.AccountSnapshot{
		AvailableCashUSD: 15000,
		BuyingPowerUSD:   30000,
		SnapshotAt:       time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
		Positions:        []domain.AccountPosition{{Symbol: "NVDA", Quantity: 10, AverageCost: 100, MarketValueUSD: 1000}},
	}
	service := NewService(
		storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20),
		market.NewMockProvider(),
		agent.NewMockClient(),
		nil,
		account.NewMockProvider(snapshot, nil),
	)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA"}
	amount := 5000.0
	settings.MaxStockTradeAmountUSD = &amount
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RefreshAccountSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: now.Add(-time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: now, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	service.agentClient = &recordingAgent{output: validServiceOutput()}

	state, err := service.RunAnalysisNow(context.Background(), "NVDA")
	if err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}
	if state.Symbols[0].AccountContext == nil || state.Symbols[0].AccountContext.SizingEnvelope == nil {
		t.Fatalf("symbol account context = %#v, want sizing context", state.Symbols[0].AccountContext)
	}
}

func TestServiceRunAnalysisNowPassesMultiTimeframeContextToAgent(t *testing.T) {
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
		{Time: now.Add(-5 * time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: now, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe15m, []domain.Bar{
		{Time: now.Add(-15 * time.Minute), Open: 122, High: 128, Low: 121, Close: 126.5, Volume: 3000},
		{Time: now, Open: 126.5, High: 129, Low: 125, Close: 128, Volume: 3600},
	})
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe1h, []domain.Bar{
		{Time: now.Add(-time.Hour), Open: 118, High: 130, Low: 117, Close: 127, Volume: 9000},
		{Time: now, Open: 127, High: 131, Low: 126, Close: 130, Volume: 9900},
	})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.RunAnalysisNow(context.Background(), "NVDA")
	if err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}

	if len(recorder.inputs) != 1 {
		t.Fatalf("agent calls = %d, want 1", len(recorder.inputs))
	}
	input := recorder.inputs[0]
	if len(input.MultiTimeframeContext) != 2 {
		t.Fatalf("multi-timeframe contexts = %#v, want 15m and 1h", input.MultiTimeframeContext)
	}
	if input.MultiTimeframeContext[0].Timeframe != domain.Timeframe15m || !input.MultiTimeframeContext[0].Available {
		t.Fatalf("first context = %#v, want available 15m", input.MultiTimeframeContext[0])
	}
	if got := *input.MultiTimeframeContext[0].CurrentPrice; got != 128 {
		t.Fatalf("15m current price = %.2f, want 128.00", got)
	}
	if input.MultiTimeframeContext[1].Timeframe != domain.Timeframe1h || !input.MultiTimeframeContext[1].Available {
		t.Fatalf("second context = %#v, want available 1h", input.MultiTimeframeContext[1])
	}
	if state.Symbols[0].Result == nil || len(state.Symbols[0].Result.ContextSummaries) != 2 {
		t.Fatalf("context summaries = %#v, want two summaries", state.Symbols[0].Result)
	}
	if state.Symbols[0].Result.ContextSummaries[0].Derived == nil || state.Symbols[0].Result.ContextSummaries[0].Derived.SessionHigh != 129 {
		t.Fatalf("15m summary = %#v, want derived session high 129", state.Symbols[0].Result.ContextSummaries[0])
	}
}

func TestServiceRunAnalysisNowContinuesWhenHigherTimeframeContextIsMissing(t *testing.T) {
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
		{Time: now.Add(-5 * time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: now, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe15m, []domain.Bar{
		{Time: now.Add(-15 * time.Minute), Open: 122, High: 128, Low: 121, Close: 126.5, Volume: 3000},
		{Time: now, Open: 126.5, High: 129, Low: 125, Close: 128, Volume: 3600},
	})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.RunAnalysisNow(context.Background(), "NVDA")
	if err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}

	if len(recorder.inputs) != 1 {
		t.Fatalf("agent calls = %d, want agent to run with partial higher timeframe context", len(recorder.inputs))
	}
	contexts := recorder.inputs[0].MultiTimeframeContext
	if len(contexts) != 2 || contexts[1].Timeframe != domain.Timeframe1h || contexts[1].Available {
		t.Fatalf("contexts = %#v, want unavailable 1h context", contexts)
	}
	if !strings.Contains(contexts[1].Error, "no data for NVDA 1h") {
		t.Fatalf("1h error = %q, want missing data message", contexts[1].Error)
	}
	if state.Symbols[0].Result == nil || len(state.Symbols[0].Result.ContextSummaries) != 2 || state.Symbols[0].Result.ContextSummaries[1].Available {
		t.Fatalf("context summaries = %#v, want unavailable 1h summary", state.Symbols[0].Result)
	}
}

func TestServiceRunAnalysisNowAppendsEverySuccessfulResultToHistory(t *testing.T) {
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
	service.agentClient = &recordingAgent{output: validServiceOutput()}

	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("first RunAnalysisNow returned error: %v", err)
	}
	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("second RunAnalysisNow returned error: %v", err)
	}

	page, err := service.GetAnalysisHistory(context.Background(), domain.AnalysisHistoryQuery{Symbol: "NVDA", Limit: 50})
	if err != nil {
		t.Fatalf("GetAnalysisHistory returned error: %v", err)
	}
	if page.Total != 2 || len(page.Records) != 2 {
		t.Fatalf("history page = %#v, want two successful NVDA records", page)
	}
}

func TestServiceRunAnalysisNowDoesNotAppendFailedAnalysisToHistory(t *testing.T) {
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
	service.agentClient = &recordingAgent{output: validServiceOutput()}
	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("successful RunAnalysisNow returned error: %v", err)
	}
	service.agentClient = &recordingAgent{err: errors.New("agent unavailable")}
	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err == nil {
		t.Fatal("failed RunAnalysisNow returned nil error")
	}

	page, err := service.GetAnalysisHistory(context.Background(), domain.AnalysisHistoryQuery{Symbol: "NVDA", Limit: 50})
	if err != nil {
		t.Fatalf("GetAnalysisHistory returned error: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("history total = %d, want only the successful analysis row", page.Total)
	}
}

func TestServiceRunBacktestUsesRegularSessionAndDoesNotAppendHistory(t *testing.T) {
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
	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100.5, High: 101.2, Low: 100.2, Close: 101, Volume: 1100},
		{Time: start.Add(10 * time.Minute), Open: 101, High: 105.5, Low: 100.8, Close: 105, Volume: 1200},
		{Time: start.Add(15 * time.Minute), Open: 105, High: 105.2, Low: 102.5, Close: 103, Volume: 1200},
	})
	stop := 98.0
	service.agentClient = &recordingAgent{output: serviceDirectionalOutput(stop, 105, start, "Held support.")}

	report, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	})
	if err != nil {
		t.Fatalf("RunBacktest returned error: %v", err)
	}

	if report.Symbol != "NVDA" || report.Timeframe != domain.Timeframe5m || report.BarCount != 4 {
		t.Fatalf("report = %#v, want NVDA 5m with four bars", report)
	}
	if len(report.Trades) != 1 || report.TotalNetPnL != 346 {
		t.Fatalf("trades = %#v, total = %.2f, want one 346 pnl grouped trade", report.Trades, report.TotalNetPnL)
	}
	page, err := service.GetAnalysisHistory(context.Background(), domain.AnalysisHistoryQuery{Symbol: "NVDA", Limit: 50})
	if err != nil {
		t.Fatalf("GetAnalysisHistory returned error: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("history total = %d, want backtest not persisted to signal history", page.Total)
	}
}

func TestServiceRunBacktestEmitsProgress(t *testing.T) {
	var events []domain.BacktestProgress
	store := storage.NewStore(filepath.Join(t.TempDir(), "settings.json"), 20)
	provider := market.NewMockProvider()
	service := NewService(store, provider, agent.NewMockClient(), func(ctx context.Context, name string, payload any) {
		if name != "backtest:progress" {
			return
		}
		progress, ok := payload.(domain.BacktestProgress)
		if !ok {
			t.Fatalf("progress payload = %T, want domain.BacktestProgress", payload)
		}
		events = append(events, progress)
	})
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA"}
	settings.SelectedTimeframe = domain.Timeframe5m
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 5, 29, 13, 30, 0, 0, time.UTC)
	provider.SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: start, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: start.Add(5 * time.Minute), Open: 100.5, High: 101.2, Low: 100.2, Close: 101, Volume: 1100},
		{Time: start.Add(10 * time.Minute), Open: 101, High: 105.5, Low: 100.8, Close: 105, Volume: 1200},
	})
	stop := 98.0
	service.agentClient = &recordingAgent{output: serviceDirectionalOutput(stop, 105, start, "Held support.")}

	if _, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	}); err != nil {
		t.Fatalf("RunBacktest returned error: %v", err)
	}

	if len(events) < 3 {
		t.Fatalf("progress events = %#v, want fetching, analyzing, and complete events", events)
	}
	if events[0].Stage != domain.BacktestProgressFetching || events[0].Symbol != "NVDA" {
		t.Fatalf("first progress = %#v, want NVDA fetching", events[0])
	}
	if events[1].Stage != domain.BacktestProgressAnalyzing || events[1].TotalBars != 3 {
		t.Fatalf("second progress = %#v, want analyzing with total bars", events[1])
	}
	last := events[len(events)-1]
	if last.Stage != domain.BacktestProgressComplete || last.ProcessedBars != 3 || last.TotalBars != 3 {
		t.Fatalf("last progress = %#v, want complete 3/3 bars", last)
	}
}

func TestServiceRunBacktestRejectsExcessiveSlippageBeforeConnection(t *testing.T) {
	service := newTestService(t)

	_, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		ShareQuantity:      100,
		SlippagePerShare:   100,
		CommissionPerOrder: 0.35,
	})
	if err == nil || !strings.Contains(err.Error(), "slippage_per_share must be at most $1.00 per share") {
		t.Fatalf("err = %v, want excessive slippage validation error", err)
	}
}

func TestServiceRunBacktestUsesSelectedEasternTimeWindow(t *testing.T) {
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
	windowStart := time.Date(2026, 5, 29, 14, 0, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: windowStart.Add(-5 * time.Minute), Open: 99, High: 99.5, Low: 98.5, Close: 99, Volume: 900},
		{Time: windowStart, Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1000},
		{Time: windowStart.Add(5 * time.Minute), Open: 100.5, High: 101.2, Low: 100.2, Close: 101, Volume: 1100},
		{Time: windowStart.Add(10 * time.Minute), Open: 101, High: 105.5, Low: 100.8, Close: 105, Volume: 1200},
		{Time: windowStart.Add(15 * time.Minute), Open: 106, High: 106.5, Low: 105.5, Close: 106, Volume: 1300},
	})
	stop := 98.0
	service.agentClient = &recordingAgent{output: serviceDirectionalOutput(stop, 105, windowStart, "Held support.")}

	report, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		Date:               "2026-05-29",
		StartTime:          "10:00",
		EndTime:            "10:15",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	})
	if err != nil {
		t.Fatalf("RunBacktest returned error: %v", err)
	}

	if report.BarCount != 3 {
		t.Fatalf("bar count = %d, want bars inside selected [10:00,10:15) Eastern window", report.BarCount)
	}
	if !report.StartTime.Equal(windowStart) || !report.EndTime.Equal(windowStart.Add(15*time.Minute)) {
		t.Fatalf("report window = %s to %s, want selected bar window close", report.StartTime, report.EndTime)
	}
}

func TestServiceRunBacktestUsesDateRangeAndDailyEasternWindow(t *testing.T) {
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
	firstDayWindow := time.Date(2026, 5, 28, 14, 0, 0, 0, time.UTC)
	secondDayWindow := time.Date(2026, 5, 29, 14, 0, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: firstDayWindow.Add(-5 * time.Minute), Open: 98, High: 99, Low: 97, Close: 98, Volume: 900},
		{Time: firstDayWindow, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1000},
		{Time: firstDayWindow.Add(5 * time.Minute), Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 1100},
		{Time: firstDayWindow.Add(10 * time.Minute), Open: 100.5, High: 101, Low: 100, Close: 100.8, Volume: 1200},
		{Time: firstDayWindow.Add(6 * time.Hour), Open: 110, High: 111, Low: 109, Close: 110, Volume: 1300},
		{Time: secondDayWindow, Open: 101, High: 102, Low: 100, Close: 101, Volume: 1000},
		{Time: secondDayWindow.Add(5 * time.Minute), Open: 101, High: 102, Low: 100, Close: 101.5, Volume: 1100},
		{Time: secondDayWindow.Add(10 * time.Minute), Open: 101.5, High: 102, Low: 101, Close: 101.8, Volume: 1200},
		{Time: secondDayWindow.Add(15 * time.Minute), Open: 120, High: 121, Low: 119, Close: 120, Volume: 1300},
	})
	stop := 95.0
	service.agentClient = &recordingAgent{output: serviceDirectionalOutput(stop, 130, firstDayWindow, "Held support.")}

	report, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:             "NVDA",
		StartDate:          "2026-05-28",
		EndDate:            "2026-05-29",
		StartTime:          "10:00",
		EndTime:            "10:15",
		ShareQuantity:      100,
		SlippagePerShare:   0.01,
		CommissionPerOrder: 1,
	})
	if err != nil {
		t.Fatalf("RunBacktest returned error: %v", err)
	}

	if report.StartDate != "2026-05-28" || report.EndDate != "2026-05-29" {
		t.Fatalf("report date range = %q-%q, want 2026-05-28-2026-05-29", report.StartDate, report.EndDate)
	}
	if report.BarCount != 6 {
		t.Fatalf("bar count = %d, want three bars from each daily [10:00,10:15) Eastern window", report.BarCount)
	}
	if report.OpenPosition == nil {
		t.Fatalf("open position = nil, want range-end open long")
	}
	if !report.StartTime.Equal(firstDayWindow) || !report.EndTime.Equal(secondDayWindow.Add(15*time.Minute)) {
		t.Fatalf("report window = %s to %s, want selected multi-day bar window", report.StartTime, report.EndTime)
	}
}

func TestServiceRunBacktestRejectsInvalidDateRange(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA"}
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}

	_, err := service.RunBacktest(context.Background(), domain.BacktestRequest{
		Symbol:        "NVDA",
		StartDate:     "2026-05-30",
		EndDate:       "2026-05-29",
		ShareQuantity: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "end_date must be on or after start_date") {
		t.Fatalf("err = %v, want invalid date range error", err)
	}
}

func TestServiceSaveSettingsDoesNotDuplicateAnalysisHistory(t *testing.T) {
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
	service.agentClient = &recordingAgent{output: validServiceOutput()}
	if _, err := service.RunAnalysisNow(context.Background(), "NVDA"); err != nil {
		t.Fatalf("RunAnalysisNow returned error: %v", err)
	}
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	page, err := service.GetAnalysisHistory(context.Background(), domain.AnalysisHistoryQuery{Symbol: "NVDA", Limit: 50})
	if err != nil {
		t.Fatalf("GetAnalysisHistory returned error: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("history total = %d, want SaveSettings not to duplicate history", page.Total)
	}
}

func TestServiceRunScreenshotAnalysisCapturesChartForSelectedSymbol(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA", "AAPL"}
	settings.SelectedTimeframe = domain.Timeframe5m
	settings.ChartWindow = &domain.ChartWindow{ID: 42, AppName: "Trader Workstation", Title: "NVDA 5m"}
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
	imagePath := filepath.Join(t.TempDir(), "chart.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	capturedAt := now.Add(30 * time.Second)
	capturer := &fakeChartCapturer{result: capture.ChartCapture{
		Path:       imagePath,
		MimeType:   "image/png",
		Source:     "window:Trader Workstation - NVDA 5m",
		CapturedAt: capturedAt,
	}}
	service.SetChartCapturer(capturer)
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.RunScreenshotAnalysis(context.Background(), "NVDA")
	if err != nil {
		t.Fatalf("RunScreenshotAnalysis returned error: %v", err)
	}

	if len(recorder.inputs) != 1 {
		t.Fatalf("agent calls = %d, want 1", len(recorder.inputs))
	}
	input := recorder.inputs[0]
	if input.Symbol != "NVDA" {
		t.Fatalf("symbol = %q, want NVDA", input.Symbol)
	}
	if input.ChartImage == nil || input.ChartImage.Path != imagePath || input.ChartImage.MimeType != "image/png" {
		t.Fatalf("chart image = %#v, want captured PNG metadata", input.ChartImage)
	}
	if len(capturer.targets) != 1 || capturer.targets[0].ID != 42 || capturer.targets[0].AppName != "Trader Workstation" {
		t.Fatalf("capture targets = %#v, want selected chart window", capturer.targets)
	}
	if len(state.Symbols) != 2 || state.Symbols[0].JobStatus != domain.JobStatusComplete || state.Symbols[1].JobStatus != domain.JobStatusIdle {
		t.Fatalf("symbols = %#v, want only selected symbol complete", state.Symbols)
	}
}

func TestServiceListChartWindowsReturnsCapturerWindows(t *testing.T) {
	service := newTestService(t)
	service.SetChartCapturer(&fakeChartCapturer{windows: []capture.WindowInfo{
		{ID: 42, AppName: "Trader Workstation", Title: "NVDA 5m"},
		{ID: 43, AppName: "Safari", Title: "Market Notes"},
	}})

	windows, err := service.ListChartWindows(context.Background())
	if err != nil {
		t.Fatalf("ListChartWindows returned error: %v", err)
	}

	if len(windows) != 2 || windows[0].ID != 42 || windows[0].AppName != "Trader Workstation" || windows[0].Title != "NVDA 5m" {
		t.Fatalf("windows = %#v, want capturer windows", windows)
	}
}

func TestServiceRunScreenshotAnalysisReportsCaptureErrorWithoutCallingAgent(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA"}
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	service.SetChartCapturer(&fakeChartCapturer{err: errors.New("screenshot capture cancelled")})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.RunScreenshotAnalysis(context.Background(), "NVDA")
	if err == nil || !strings.Contains(err.Error(), "screenshot capture cancelled") {
		t.Fatalf("err = %v, want screenshot capture cancelled", err)
	}
	if len(recorder.inputs) != 0 {
		t.Fatalf("agent calls = %d, want 0", len(recorder.inputs))
	}
	if len(state.Symbols) != 1 || state.Symbols[0].JobStatus != domain.JobStatusFailed {
		t.Fatalf("symbols = %#v, want failed NVDA", state.Symbols)
	}
}

func TestServiceRunScheduledScreenshotAnalysisRunsWatchlistOnceForBoundary(t *testing.T) {
	service := newTestService(t)
	settings := domain.DefaultSettings()
	settings.Watchlist = []string{"NVDA", "AAPL"}
	settings.SelectedTimeframe = domain.Timeframe5m
	if _, err := service.SaveSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConnectIBKR(context.Background()); err != nil {
		t.Fatal(err)
	}
	service.SetScheduledAnalysisEnabled(true)

	boundary := time.Date(2026, 5, 30, 9, 5, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: boundary.Add(-time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: boundary, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	service.provider.(*market.MockProvider).SetHistoricalBars("AAPL", domain.Timeframe5m, []domain.Bar{
		{Time: boundary.Add(-time.Minute), Open: 202, High: 204, Low: 201, Close: 203, Volume: 1000},
		{Time: boundary, Open: 203, High: 205, Low: 202, Close: 204, Volume: 1400},
	})
	capturer := &recordingChartCapturer{dir: t.TempDir(), capturedAt: boundary.Add(-10 * time.Second)}
	service.SetChartCapturer(capturer)
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.runScheduledScreenshotAnalysis(context.Background(), boundary)
	if err != nil {
		t.Fatalf("runScheduledScreenshotAnalysis returned error: %v", err)
	}
	if len(recorder.inputs) != 2 {
		t.Fatalf("agent calls = %d, want 2", len(recorder.inputs))
	}
	for _, input := range recorder.inputs {
		if input.ChartImage == nil || input.ChartImage.CapturedAt == nil || !input.ChartImage.CapturedAt.Equal(capturer.capturedAt) {
			t.Fatalf("input chart image = %#v, want scheduled screenshot metadata", input.ChartImage)
		}
	}
	if len(state.Symbols) != 2 || state.Symbols[0].JobStatus != domain.JobStatusComplete || state.Symbols[1].JobStatus != domain.JobStatusComplete {
		t.Fatalf("symbols = %#v, want both symbols complete", state.Symbols)
	}

	if _, err := service.runScheduledScreenshotAnalysis(context.Background(), boundary); err != nil {
		t.Fatalf("duplicate runScheduledScreenshotAnalysis returned error: %v", err)
	}
	if len(recorder.inputs) != 2 {
		t.Fatalf("agent calls = %d, want duplicate boundary suppressed", len(recorder.inputs))
	}
	if capturer.calls != 2 {
		t.Fatalf("capture calls = %d, want one screenshot per scheduled symbol", capturer.calls)
	}
}

func TestServicePausedScheduledAnalysisSkipsBoundaryRun(t *testing.T) {
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
	if _, err := service.UpdateScheduledAnalysisEnabled(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	boundary := time.Date(2026, 5, 30, 9, 5, 0, 0, time.UTC)
	service.provider.(*market.MockProvider).SetHistoricalBars("NVDA", domain.Timeframe5m, []domain.Bar{
		{Time: boundary.Add(-time.Minute), Open: 124, High: 126, Low: 123, Close: 125, Volume: 1000},
		{Time: boundary, Open: 125, High: 127, Low: 124, Close: 126, Volume: 1400},
	})
	service.SetChartCapturer(&recordingChartCapturer{dir: t.TempDir(), capturedAt: boundary.Add(-10 * time.Second)})
	recorder := &recordingAgent{output: validServiceOutput()}
	service.agentClient = recorder

	state, err := service.runScheduledScreenshotAnalysis(context.Background(), boundary)
	if err != nil {
		t.Fatalf("runScheduledScreenshotAnalysis returned error: %v", err)
	}

	if state.ScheduledAnalysisEnabled {
		t.Fatalf("scheduled analysis enabled = true, want false")
	}
	if len(recorder.inputs) != 0 {
		t.Fatalf("agent calls = %d, want paused scheduler to skip analysis", len(recorder.inputs))
	}
	if len(state.Symbols) != 1 || state.Symbols[0].JobStatus != domain.JobStatusIdle {
		t.Fatalf("symbols = %#v, want paused symbol to remain idle", state.Symbols)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	store := storage.NewStore(filepath.Join(t.TempDir(), "app.db"), 20)
	mockAgent := agent.NewMockClient()
	return NewService(store, market.NewMockProvider(), mockAgent, nil)
}

type fakeChartCapturer struct {
	result  capture.ChartCapture
	windows []capture.WindowInfo
	targets []capture.WindowTarget
	err     error
}

func (f *fakeChartCapturer) CaptureChart(ctx context.Context, target capture.WindowTarget) (capture.ChartCapture, error) {
	if err := ctx.Err(); err != nil {
		return capture.ChartCapture{}, err
	}
	f.targets = append(f.targets, target)
	return f.result, f.err
}

func (f *fakeChartCapturer) ListWindows(ctx context.Context) ([]capture.WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.windows, f.err
}

type recordingChartCapturer struct {
	dir        string
	capturedAt time.Time
	calls      int
}

func (r *recordingChartCapturer) CaptureChart(ctx context.Context, target capture.WindowTarget) (capture.ChartCapture, error) {
	if err := ctx.Err(); err != nil {
		return capture.ChartCapture{}, err
	}
	file, err := os.CreateTemp(r.dir, "chart-*.png")
	if err != nil {
		return capture.ChartCapture{}, err
	}
	if _, err := file.Write([]byte("png")); err != nil {
		_ = file.Close()
		return capture.ChartCapture{}, err
	}
	if err := file.Close(); err != nil {
		return capture.ChartCapture{}, err
	}
	r.calls++
	return capture.ChartCapture{
		Path:       file.Name(),
		MimeType:   "image/png",
		Source:     "desktop_region",
		CapturedAt: r.capturedAt,
	}, nil
}

type portSelectingProvider struct {
	successPort int
	state       market.ProviderState
	attempts    []market.ConnectionSettings
}

func (p *portSelectingProvider) Connect(ctx context.Context, settings market.ConnectionSettings) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.attempts = append(p.attempts, settings)
	if settings.Port != p.successPort {
		p.state = market.ProviderState{Status: domain.ConnectionFailed}
		return errors.New("port closed")
	}
	p.state = market.ProviderState{Status: domain.ConnectionConnected}
	return nil
}

func (p *portSelectingProvider) Disconnect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.state = market.ProviderState{Status: domain.ConnectionDisconnected}
	return nil
}

func (p *portSelectingProvider) State() market.ProviderState {
	return p.state
}

func (p *portSelectingProvider) Subscribe(ctx context.Context, symbol string, timeframe domain.Timeframe) (<-chan market.BarUpdate, market.Unsubscribe, error) {
	return nil, nil, errors.New("not implemented")
}

func (p *portSelectingProvider) HistoricalBars(ctx context.Context, symbol string, timeframe domain.Timeframe, limit int) ([]domain.Bar, error) {
	return nil, errors.New("not implemented")
}

func (p *portSelectingProvider) HistoricalBarsRange(ctx context.Context, symbol string, timeframe domain.Timeframe, start time.Time, end time.Time) ([]domain.Bar, error) {
	return nil, errors.New("not implemented")
}

func (p *portSelectingProvider) attemptedPorts() []int {
	ports := make([]int, 0, len(p.attempts))
	for _, attempt := range p.attempts {
		ports = append(ports, attempt.Port)
	}
	return ports
}

type recordingAgent struct {
	output domain.AgentOutput
	err    error
	inputs []domain.AgentInput
}

func (r *recordingAgent) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	r.inputs = append(r.inputs, input)
	if r.err != nil {
		return domain.AgentOutput{}, r.err
	}
	return r.output, nil
}

func validServiceOutput() domain.AgentOutput {
	stop := 124.2
	return serviceDirectionalOutput(stop, 126.4, time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC), "Held support.")
}

func serviceDirectionalOutput(stop float64, target float64, generatedAt time.Time, summary string) domain.AgentOutput {
	riskReward := 2.0
	return domain.AgentOutput{
		Direction:        domain.DirectionLong,
		SetupQuality:     domain.SetupQualityAPlus,
		EntryZone:        &domain.EntryZone{Low: 99, High: 101},
		StopLoss:         &stop,
		TakeProfit:       []float64{target},
		RiskReward:       &riskReward,
		Confidence:       0.7,
		MarketRegime:     "趋势回踩",
		TradeThesis:      "关键位置回踩后重新出现主动买盘。",
		Counterargument:  "如果跌回触发区下方，突破可能失败。",
		NoTradeReason:    "",
		RejectionReasons: []string{},
		Summary:          summary,
		PriceAction:      []string{"Higher low"},
		InvalidatedIf:    "Close below stop",
		GeneratedAt:      generatedAt,
	}
}
