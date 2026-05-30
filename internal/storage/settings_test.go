package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestStoreLoadsDefaultSettingsWhenFileMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "settings.json"), 20)

	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if snapshot.Settings.IBKRHost != "127.0.0.1" {
		t.Fatalf("host = %q, want default", snapshot.Settings.IBKRHost)
	}
	if snapshot.Settings.IBKRPort != 7497 {
		t.Fatalf("port = %d, want 7497", snapshot.Settings.IBKRPort)
	}
	if snapshot.Settings.IBKRClientID != 1001 {
		t.Fatalf("client id = %d, want 1001", snapshot.Settings.IBKRClientID)
	}
	if snapshot.Settings.SelectedTimeframe != domain.Timeframe5m {
		t.Fatalf("timeframe = %q, want 5m", snapshot.Settings.SelectedTimeframe)
	}
}

func TestStoreSavesAndLoadsSettingsAndHistory(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "settings.json"), 20)
	stop := 124.2
	riskReward := 2.4
	snapshot := Snapshot{
		Settings: domain.Settings{
			IBKRHost:          "127.0.0.1",
			IBKRPort:          4002,
			IBKRClientID:      44,
			Watchlist:         []string{"nvda", "AAPL", "nvda"},
			SelectedTimeframe: domain.Timeframe15m,
		},
		History: []domain.AnalysisResult{{
			Symbol:       "NVDA",
			Timeframe:    domain.Timeframe15m,
			CurrentPrice: 125.3,
			Output: domain.AgentOutput{
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
			},
			UpdatedAt: time.Date(2026, 5, 30, 18, 0, 1, 0, time.UTC),
		}},
	}

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := loaded.Settings.IBKRPort, 4002; got != want {
		t.Fatalf("port = %d, want %d", got, want)
	}
	if got, want := loaded.Settings.Watchlist, []string{"NVDA", "AAPL"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("watchlist = %#v, want %#v", got, want)
	}
	if len(loaded.History) != 1 || loaded.History[0].Symbol != "NVDA" {
		t.Fatalf("history = %#v, want one NVDA result", loaded.History)
	}
}

func TestStoreRecoversFromCorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(path, 20)

	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if snapshot.Settings.SelectedTimeframe != domain.Timeframe5m {
		t.Fatalf("timeframe = %q, want default", snapshot.Settings.SelectedTimeframe)
	}
}

func TestStoreTrimsHistoryOnSave(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "settings.json"), 2)
	history := []domain.AnalysisResult{
		{Symbol: "AAPL", UpdatedAt: time.Date(2026, 5, 30, 17, 0, 0, 0, time.UTC)},
		{Symbol: "MSFT", UpdatedAt: time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)},
		{Symbol: "NVDA", UpdatedAt: time.Date(2026, 5, 30, 19, 0, 0, 0, time.UTC)},
	}

	if err := store.Save(Snapshot{Settings: domain.DefaultSettings(), History: history}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded.History) != 2 {
		t.Fatalf("history length = %d, want 2", len(loaded.History))
	}
	if loaded.History[0].Symbol != "MSFT" || loaded.History[1].Symbol != "NVDA" {
		t.Fatalf("history = %#v, want newest two in original order", loaded.History)
	}
}
