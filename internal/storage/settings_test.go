package storage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ibkr-stock-analysis/internal/domain"
)

func TestStoreLoadsDefaultSettingsWhenFileMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "app.db"), 20)

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

func TestStoreSavesAndLoadsSettingsFromSQLite(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "app.db"), 20)
	settings := domain.Settings{
		IBKRHost:          "127.0.0.1",
		IBKRPort:          4002,
		IBKRClientID:      44,
		Watchlist:         []string{"nvda", "AAPL", "nvda"},
		SelectedTimeframe: domain.Timeframe15m,
	}

	if err := store.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
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
	if len(loaded.History) != 0 {
		t.Fatalf("history = %#v, want empty history until analysis rows are appended", loaded.History)
	}
}

func TestStoreAppendsQueriesAndLoadsLatestAnalysisHistory(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "app.db"), 20)
	first := analysisResult("NVDA", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC))
	second := analysisResult("NVDA", domain.Timeframe15m, domain.DirectionShort, time.Date(2026, 5, 30, 19, 0, 0, 0, time.UTC))
	third := analysisResult("AAPL", domain.Timeframe5m, domain.DirectionNeutral, time.Date(2026, 5, 30, 18, 30, 0, 0, time.UTC))

	if _, err := store.AppendAnalysisResult(first); err != nil {
		t.Fatalf("AppendAnalysisResult first returned error: %v", err)
	}
	secondID, err := store.AppendAnalysisResult(second)
	if err != nil {
		t.Fatalf("AppendAnalysisResult second returned error: %v", err)
	}
	if _, err := store.AppendAnalysisResult(third); err != nil {
		t.Fatalf("AppendAnalysisResult third returned error: %v", err)
	}

	page, err := store.QueryAnalysisHistory(domain.AnalysisHistoryQuery{Symbol: "nvda", Limit: 50})
	if err != nil {
		t.Fatalf("QueryAnalysisHistory returned error: %v", err)
	}
	if page.Total != 2 || len(page.Records) != 2 {
		t.Fatalf("page = %#v, want two NVDA records", page)
	}
	if page.Records[0].ID != secondID || page.Records[0].Result.Timeframe != domain.Timeframe15m {
		t.Fatalf("first record = %#v, want newest NVDA 15m result", page.Records[0])
	}

	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(snapshot.History) != 2 {
		t.Fatalf("latest history length = %d, want one latest row per symbol", len(snapshot.History))
	}
	latest := latestBySymbol(snapshot.History)
	if latest["NVDA"].Timeframe != domain.Timeframe15m || latest["AAPL"].Timeframe != domain.Timeframe5m {
		t.Fatalf("latest = %#v, want newest per symbol", latest)
	}
}

func TestStoreFiltersHistoryAndPaginates(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "app.db"), 20)
	rows := []domain.AnalysisResult{
		analysisResult("NVDA", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)),
		analysisResult("AAPL", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 1, 0, 0, time.UTC)),
		analysisResult("MSFT", domain.Timeframe15m, domain.DirectionShort, time.Date(2026, 5, 30, 18, 2, 0, 0, time.UTC)),
		analysisResult("TSLA", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 3, 0, 0, time.UTC)),
	}
	for _, row := range rows {
		if _, err := store.AppendAnalysisResult(row); err != nil {
			t.Fatalf("AppendAnalysisResult returned error: %v", err)
		}
	}

	page, err := store.QueryAnalysisHistory(domain.AnalysisHistoryQuery{
		Timeframe: domain.Timeframe5m,
		Direction: domain.DirectionLong,
		Limit:     2,
		Offset:    1,
	})
	if err != nil {
		t.Fatalf("QueryAnalysisHistory returned error: %v", err)
	}

	if page.Total != 3 || len(page.Records) != 2 {
		t.Fatalf("page = %#v, want total 3 and second page slice of 2", page)
	}
	if got := []string{page.Records[0].Result.Symbol, page.Records[1].Result.Symbol}; got[0] != "AAPL" || got[1] != "NVDA" {
		t.Fatalf("symbols = %#v, want AAPL then NVDA after newest row offset", got)
	}
}

func TestStoreWritesFilteredAnalysisHistoryCSV(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "app.db"), 20)
	if _, err := store.AppendAnalysisResult(analysisResult("NVDA", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("AppendAnalysisResult returned error: %v", err)
	}
	if _, err := store.AppendAnalysisResult(analysisResult("AAPL", domain.Timeframe5m, domain.DirectionNeutral, time.Date(2026, 5, 30, 18, 1, 0, 0, time.UTC))); err != nil {
		t.Fatalf("AppendAnalysisResult returned error: %v", err)
	}

	var out bytes.Buffer
	if err := store.WriteAnalysisHistoryCSV(&out, domain.AnalysisHistoryQuery{Symbol: "NVDA"}); err != nil {
		t.Fatalf("WriteAnalysisHistoryCSV returned error: %v", err)
	}

	csv := out.String()
	if !strings.Contains(csv, "updated_at,symbol,timeframe,direction,current_price,entry_low,entry_high,stop_loss,take_profit,risk_reward,confidence,summary,invalidated_if,price_action") {
		t.Fatalf("csv header missing expected columns:\n%s", csv)
	}
	if !strings.Contains(csv, "NVDA,5m,long") {
		t.Fatalf("csv = %q, want NVDA long row", csv)
	}
	if strings.Contains(csv, "AAPL") {
		t.Fatalf("csv = %q, filtered export should not include AAPL", csv)
	}
}

func TestStoreMigratesLegacyJSONOnceAndKeepsFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	legacyPath := filepath.Join(dir, "settings.json")
	legacy := Snapshot{
		Settings: domain.Settings{
			IBKRHost:          "localhost",
			IBKRPort:          4002,
			IBKRClientID:      55,
			Watchlist:         []string{"nvda"},
			SelectedTimeframe: domain.Timeframe15m,
		},
		History: []domain.AnalysisResult{
			analysisResult("NVDA", domain.Timeframe5m, domain.DirectionLong, time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)),
		},
	}
	if err := writeLegacySnapshot(legacyPath, legacy); err != nil {
		t.Fatal(err)
	}

	store := NewStoreWithLegacyPath(dbPath, legacyPath, 20)
	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if snapshot.Settings.IBKRPort != 4002 || len(snapshot.History) != 1 {
		t.Fatalf("snapshot = %#v, want migrated settings and history", snapshot)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy settings file should be preserved: %v", err)
	}

	if _, err := store.AppendAnalysisResult(analysisResult("AAPL", domain.Timeframe5m, domain.DirectionNeutral, time.Date(2026, 5, 30, 18, 1, 0, 0, time.UTC))); err != nil {
		t.Fatalf("AppendAnalysisResult returned error: %v", err)
	}
	if _, err := store.Load(); err != nil {
		t.Fatalf("second Load returned error: %v", err)
	}
	page, err := store.QueryAnalysisHistory(domain.AnalysisHistoryQuery{Limit: 50})
	if err != nil {
		t.Fatalf("QueryAnalysisHistory returned error: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want migrated row plus appended row without duplicate import", page.Total)
	}
}

func TestStoreRecoversFromCorruptLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(legacyPath, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStoreWithLegacyPath(filepath.Join(dir, "app.db"), legacyPath, 20)

	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if snapshot.Settings.SelectedTimeframe != domain.Timeframe5m {
		t.Fatalf("timeframe = %q, want default", snapshot.Settings.SelectedTimeframe)
	}
}

func latestBySymbol(history []domain.AnalysisResult) map[string]domain.AnalysisResult {
	out := make(map[string]domain.AnalysisResult, len(history))
	for _, result := range history {
		out[result.Symbol] = result
	}
	return out
}

func analysisResult(symbol string, timeframe domain.Timeframe, direction domain.Direction, at time.Time) domain.AnalysisResult {
	stop := 124.2
	riskReward := 2.4
	output := domain.AgentOutput{
		Direction:     direction,
		TakeProfit:    []float64{},
		Confidence:    0.7,
		Summary:       "Held support.",
		PriceAction:   []string{"Higher low"},
		InvalidatedIf: "Close below 124.2",
		GeneratedAt:   at,
	}
	if direction == domain.DirectionLong || direction == domain.DirectionShort {
		output.EntryZone = &domain.EntryZone{Low: 125.1, High: 125.5}
		output.StopLoss = &stop
		output.TakeProfit = []float64{126.4}
		output.RiskReward = &riskReward
	}
	return domain.AnalysisResult{
		Symbol:       symbol,
		Timeframe:    timeframe,
		CurrentPrice: 125.3,
		Output:       output,
		UpdatedAt:    at.Add(time.Second),
	}
}

func writeLegacySnapshot(path string, snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
