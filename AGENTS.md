# Agent Guide

## Project Summary

This repository is a Wails v2 desktop app for IBKR market and account-aware analysis with a local Codex CLI analysis runtime.

Version 1 is market-data analysis assistance only.

Version 2 may add read-only account snapshots for cash, buying power, and current stock positions, plus advisory position management based on a user-defined maximum stock trade amount. This must remain advisory and manually reviewed. Do not add order placement, order cancellation, order staging, automated trading, or any IBKR account mutation capability in backend code, frontend UI, tests, docs, or Codex prompts.

## Runtime Defaults

- IBKR host: `127.0.0.1`
- IBKR paper trading port: `7497`
- IBKR client ID: `1001`
- Default timeframe: `5m`
- Codex model: Codex CLI default, optionally overridable with `CODEX_MODEL`
- Real agent analysis requires an installed and authenticated `codex` CLI
- PriceAction knowledge is bundled from `priceaction/*.md`; `PRICEACTION_KB_PATH=/path/to/priceaction` overrides it for local experiments

## Required Tooling

- Go `1.26+`
- Node `22+`
- npm `11+`
- Wails CLI `v2.12.0`
- Codex CLI installed and authenticated for real AI analysis

Install Wails when needed:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

Initialize dependencies:

```bash
npm install --prefix frontend
```

## Common Commands

Run the full verification suite:

```bash
make test
```

Focused checks:

```bash
make test-go
make test-frontend
make build
```

Run the desktop app in development mode:

```bash
make dev
```

## Architecture Map

- `main.go`, `app.go`: Wails app entry and bindings.
- `internal/app`: Go application service that coordinates settings, IBKR connection, market data, analysis queue, persistence, and Wails events.
- `internal/domain`: shared Go domain models and validation defaults.
- `internal/storage`: persisted settings and last known analysis history.
- `internal/market`: read-only market data provider interface, mock provider, historical bar helpers, and IBKR adapter.
- `internal/account`: planned V2 read-only account snapshot provider for cash, buying power, and current stock positions.
- `internal/scheduler`: timeframe scheduling and next-run logic.
- `internal/analysis`: feature derivation and bounded analysis queue.
- `internal/agent`: Go client for local `codex exec` analysis, PriceAction knowledge selection, and tests for the Codex CLI boundary.
- `priceaction`: bundled Markdown knowledge base used by real Codex analysis.
- `frontend`: React TypeScript workspace UI using Wails-generated bindings in `frontend/wailsjs`.
- `docs/manual-acceptance.md`: manual acceptance flow for TWS or IB Gateway paper trading.
- `docs/superpowers/specs` and `docs/superpowers/plans`: original design and milestone plans.

## Safety Rules

- Keep `market.MarketDataProvider` market-data read-only.
- Use a separate account provider boundary for V2 account snapshots. Read-only account APIs for cash, buying power, and current positions are allowed only inside that boundary.
- Do not call IBKR order mutation APIs such as `PlaceOrder` or `CancelOrder`.
- Do not request open orders or staged order state through `ReqOpenOrders` unless a future spec explicitly expands the safety boundary.
- Do not add UI controls that imply executing, approving, staging, or automating trades.
- Analysis output may include directional bias, entry zone, stop loss, take profit, risk-reward, confidence, summary, invalidation notes, and advisory position-management fields, but it must remain advisory.
- V2 Codex prompts may use only supplied market data, Go-injected PriceAction excerpts, a sanitized read-only account snapshot, and the user's configured maximum stock trade amount. Do not let Codex browse the repo, request credentials, inspect order state, or mutate brokerage/account state.
- Advisory sizing must be bounded by the user's maximum stock trade amount and available account snapshot. It must never place, stage, approve, or automate an IBKR order.
- Preserve the mock market provider so tests and local development do not require TWS/Gateway.

## Development Expectations

- Prefer small, focused changes that match existing Go, React, and TypeScript patterns.
- Add or update tests when behavior changes.
- For backend behavior, start with Go tests under the relevant `internal/...` package.
- For account snapshot behavior, use Go tests under `internal/account` and service tests under `internal/app`.
- For Codex analysis behavior, use Go tests under `internal/agent`.
- For frontend behavior, use Vitest and Testing Library tests under `frontend/src`.
- After changing Wails-bound Go methods or domain models, regenerate/check frontend bindings by running the relevant build or `make build`.
- Keep generated artifacts and dependency lockfiles stable unless the change genuinely updates dependencies.

## Manual Acceptance Checklist

Before considering a live IBKR integration change complete, run the paper-trading flow in `docs/manual-acceptance.md`:

- TWS or IB Gateway paper mode
- API socket enabled on `127.0.0.1:7497`
- Client ID `1001`
- Watchlist `NVDA, AAPL, TSLA`
- Timeframe `5m`
- Confirm queued/analyzing/complete or scoped failure states per symbol
- If V2 account-aware analysis is enabled, confirm account snapshot reads cash/buying power/positions only
- Confirm there is no order placement UI, backend order method, staged order state, automated execution, or IBKR order mutation call
