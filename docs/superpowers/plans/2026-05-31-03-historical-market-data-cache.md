# Historical Market Data Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only local historical bar cache to reduce repeated IBKR historical data requests during analysis.

**Architecture:** Store normalized historical bars in the existing SQLite app database and wrap the current `MarketDataProvider` with a cache-aware provider. Analysis continues to use the same service flow, but historical bars are served from cache first, refreshed from IBKR only when needed, and explicitly marked stale when fallback cache data is used.

**Tech Stack:** Go, SQLite via `modernc.org/sqlite`, Wails bindings, React TypeScript, Vitest.

---

### Task 1: Persist Historical Bars

**Files:**
- Modify: `internal/storage/settings.go`
- Modify: `internal/storage/settings_test.go`

- [ ] Add a `historical_bars` table in `ensureSchema` with columns: `symbol`, `timeframe`, `bar_time`, `open`, `high`, `low`, `close`, `volume`, and `fetched_at`.
- [ ] Add a unique key on `(symbol, timeframe, bar_time)` and indexes for `(symbol, timeframe, bar_time DESC)` so repeated IBKR responses update existing bars instead of duplicating them.
- [ ] Add storage methods:
  - `UpsertHistoricalBars(symbol string, timeframe domain.Timeframe, bars []domain.Bar) error`
  - `QueryHistoricalBars(symbol string, timeframe domain.Timeframe, limit int) ([]domain.Bar, error)`
  - `PruneHistoricalBars(before time.Time) error`
  - `ClearHistoricalBars() error`
- [ ] Normalize symbols to uppercase and store all bar times in UTC RFC3339Nano format.
- [ ] Keep the cache retention fixed at 30 days for this version.
- [ ] Test upsert de-duplication, ordered limited reads, pruning old rows, and clearing only historical bars without deleting settings or analysis history.
- [ ] Commit: `feat: persist historical bar cache`.

### Task 2: Add Market Data Freshness Metadata

**Files:**
- Modify: `internal/market/provider.go`
- Modify: `internal/market/mock_provider.go`
- Modify: `internal/market/mock_provider_test.go`
- Modify: `internal/market/ibkr_provider.go`
- Modify: `internal/market/ibkr_provider_test.go`

- [ ] Replace the `HistoricalBars` return shape with a metadata-bearing result:

```go
type HistoricalDataSource string

const (
	HistoricalDataSourceIBKR       HistoricalDataSource = "ibkr"
	HistoricalDataSourceCache      HistoricalDataSource = "cache"
	HistoricalDataSourceStaleCache HistoricalDataSource = "stale_cache"
	HistoricalDataSourceMock       HistoricalDataSource = "mock"
)

type HistoricalBarsResult struct {
	Bars        []domain.Bar          `json:"bars"`
	Source      HistoricalDataSource  `json:"source"`
	Stale       bool                  `json:"stale"`
	Warning     string                `json:"warning,omitempty"`
	LastBarTime *time.Time            `json:"last_bar_time,omitempty"`
}
```

- [ ] Update `MarketDataProvider.HistoricalBars` to return `(HistoricalBarsResult, error)`.
- [ ] Update `IBKRProvider.HistoricalBars` to return source `ibkr`, stale `false`, and last bar time when data is available.
- [ ] Update `MockProvider.HistoricalBars` to return source `mock`, stale `false`, and preserve existing missing-data behavior.
- [ ] Update all compile-time interface and provider tests for the new return type.
- [ ] Commit: `feat: attach freshness metadata to historical bars`.

### Task 3: Implement Cache-Aware Provider

**Files:**
- Create: `internal/market/cached_provider.go`
- Create: `internal/market/cached_provider_test.go`
- Modify: `internal/market/bars.go`
- Modify: `internal/market/bars_test.go`

- [ ] Add `CachedProvider` with constructor `NewCachedProvider(inner MarketDataProvider, store HistoricalBarStore, options CacheOptions) MarketDataProvider`.
- [ ] Define a narrow `HistoricalBarStore` interface in `internal/market` so tests can use fakes and `market` does not depend on storage internals beyond the methods it needs.
- [ ] Use default options:
  - `Retention: 30 * 24 * time.Hour`
  - `StaleMaxAge: previousUSMarketWorkday(now)`
  - `Now: time.Now`
- [ ] On `HistoricalBars`, first query cached bars for `symbol + timeframe + limit`.
- [ ] Treat cache as fresh when it has at least `limit` bars and the last cached bar is close enough for the requested timeframe:
  - `1m`, `5m`, `15m`: last bar must be within one timeframe duration of the latest expected closed boundary.
  - `1h`: last bar must be within one hour of the latest expected closed boundary.
- [ ] If cache is fresh, return source `cache` without calling the inner provider.
- [ ] If cache is missing, short, or stale, call the inner provider, upsert returned bars, prune rows older than 30 days, then re-query and return source `ibkr`.
- [ ] If the inner provider fails but cached bars exist and the last bar is from the current or previous US market workday, return source `stale_cache`, stale `true`, and a warning such as `Using cached market data because IBKR refresh failed: <error>`.
- [ ] If the inner provider fails and cached bars are older than the stale limit, return the inner error and no bars.
- [ ] Implement the US market workday rule as a simple New York date helper that skips Saturday and Sunday; do not add a holiday calendar in this version.
- [ ] Keep `Connect`, `Disconnect`, `State`, and `Subscribe` as pass-through methods.
- [ ] Test cache hit avoids inner provider calls, refresh writes through, refresh failure falls back to stale cache, too-old cache fails, retention pruning runs after refresh, and pass-through methods preserve behavior.
- [ ] Commit: `feat: cache historical market data requests`.

### Task 4: Wire Cache Into App Analysis

**Files:**
- Modify: `app.go`
- Modify: `internal/app/service.go`
- Modify: `internal/app/service_test.go`
- Modify: `internal/domain/models.go`
- Modify: `internal/domain/models_test.go`
- Modify: `frontend/wailsjs/go/models.ts`

- [ ] Wrap the real IBKR provider with `market.NewCachedProvider(market.NewIBKRProvider(), store, market.DefaultCacheOptions())` in `NewApp` and `startup`.
- [ ] Add `MarketDataSource`, `MarketDataStale`, and `MarketDataWarning` fields to `domain.AnalysisResult`.
- [ ] Do not reuse `AnalysisResult.Stale`; keep it reserved for previous-valid-result fallback after agent failures.
- [ ] Update `Service.buildAgentInput` and `buildMultiTimeframeContext` to use `HistoricalBarsResult.Bars`.
- [ ] Track the primary timeframe freshness metadata on successful analysis results.
- [ ] When stale cache data is used, include a clear warning in the analysis result and frontend state.
- [ ] For higher-timeframe context failures, keep current partial-context behavior; stale higher-timeframe context may be included, but the primary timeframe controls the result-level market data warning.
- [ ] Add Wails-bound method `ClearHistoricalDataCache() error` on `App` and `Service`, implemented through `store.ClearHistoricalBars()`.
- [ ] Regenerate Wails bindings by running `make build` after backend model and method changes.
- [ ] Test successful analysis with fresh provider data, stale cache fallback with warning, no-data behavior when cache is too old, and cache clearing through the service.
- [ ] Commit: `feat: use cached historical bars in analysis`.

### Task 5: Add Minimal Frontend Controls

**Files:**
- Modify: `frontend/src/api/backend.ts`
- Modify: `frontend/src/types/domain.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/ConnectionPanel.tsx`
- Modify: `frontend/src/components/AnalysisDetail.tsx`
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/components/AnalysisDetail.test.tsx`

- [ ] Add `ClearHistoricalDataCache` to the frontend backend wrapper.
- [ ] Add the new market data freshness fields to TypeScript domain types.
- [ ] Add one small cache cleanup button near settings or connection controls labeled `Clear market data cache`.
- [ ] After successful cache clearing, keep app settings and current analysis state intact; show a concise transient status message or reuse existing error/status state patterns.
- [ ] Display stale market data warnings in the selected symbol detail view when `market_data_stale` is true.
- [ ] Do not add cache size, hit rate, advanced retention settings, or any trading-related controls.
- [ ] Test the clear-cache action calls the Wails method and stale market data warnings render in the detail panel.
- [ ] Commit: `feat: expose market cache cleanup in UI`.

### Task 6: Verification And Safety Review

**Files:**
- Modify: `docs/manual-acceptance.md`
- Review: `internal/safety/safety_test.go`

- [ ] Add a manual acceptance note for cache behavior: run analysis once, disconnect or block IBKR refresh, confirm stale cache warning appears only when cached bars are within the allowed US market workday window.
- [ ] Confirm `internal/safety/safety_test.go` still verifies that no order placement, cancellation, account update, positions, or open-order APIs are exposed.
- [ ] Run `make test`.
- [ ] Run `make build`.
- [ ] Confirm generated Wails bindings changed only because of the new cache clear method and market data freshness fields.
- [ ] Commit: `test: verify historical market data cache`.

## Acceptance Criteria

- Repeated analysis for the same symbol and timeframe avoids unnecessary IBKR historical data requests when cached bars are fresh.
- Cached bars are persisted in the existing app SQLite database and retained for 30 days.
- IBKR refresh failures can fall back to cached bars only when the cache is within the current or previous US market workday.
- Stale market data is visibly labeled and does not reuse the existing analysis-result stale semantics.
- Users can clear historical market data cache without deleting settings or analysis history.
- The implementation remains read-only and adds no order, account, position, sizing, allocation, or automated trading capability.
