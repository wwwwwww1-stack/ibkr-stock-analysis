# IBKR Market Data Scheduler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read-only market data providers, normalized bars, closed-bar detection, and duplicate-safe batch scheduling.

**Architecture:** Application logic depends on a `MarketDataProvider` interface. Mock and IBKR providers emit the same normalized data so scheduler tests do not need a live TWS/Gateway session.

**Tech Stack:** Go, `github.com/scmhub/ibapi@v0.10.46`, Wails events.

---

### Task 1: Provider Interface And Mock

**Files:**
- Create: `internal/market/provider.go`
- Create: `internal/market/mock_provider.go`
- Create: `internal/market/mock_provider_test.go`

- [ ] Test connection state transitions, historical bar responses, subscription event emission, and missing symbol behavior.
- [ ] Define read-only provider methods: `Connect`, `Disconnect`, `State`, `Subscribe`, `HistoricalBars`.
- [ ] Do not add order, account, portfolio, or position mutation methods.
- [ ] Commit: `feat: add market data provider interface`.

### Task 2: Bar Utilities

**Files:**
- Create: `internal/market/bars.go`
- Create: `internal/market/bars_test.go`

- [ ] Test close-time calculation for `1m`, `5m`, `15m`, and `1h`.
- [ ] Test duplicate closed-bar key format as `SYMBOL|TIMEFRAME|RFC3339`.
- [ ] Implement UTC storage and New York market close-time interpretation helpers.
- [ ] Commit: `feat: add bar close helpers`.

### Task 3: Scheduler

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/scheduler_test.go`

- [ ] Test no enqueue when disconnected.
- [ ] Test one job per symbol on a new closed bar.
- [ ] Test duplicate suppression for same `symbol + timeframe + closedAt`.
- [ ] Test one pending batch per timeframe while another batch is active.
- [ ] Commit: `feat: schedule watchlist analysis on closed bars`.

### Task 4: IBKR Provider Adapter

**Files:**
- Create: `internal/market/ibkr_provider.go`
- Create: `internal/market/ibkr_provider_test.go`

- [ ] Add compile-time tests for IBKR provider satisfying `MarketDataProvider`.
- [ ] Wrap `github.com/scmhub/ibapi` behind the interface, using US stock SMART/USD contracts.
- [ ] Implement connection settings only for host, port, and client ID.
- [ ] Implement historical bars and subscription stubs with explicit unavailable errors when a live TWS session is not present.
- [ ] Document the TWS/Gateway API setup in README.
- [ ] Commit: `feat: add ibkr market data adapter`.

