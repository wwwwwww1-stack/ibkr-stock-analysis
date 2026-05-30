# Agent Guide

## Project Summary

This repository is a Wails v2 desktop app for read-only IBKR market data analysis with a local Codex CLI analysis runtime.

Version 1 is analysis assistance only. Do not add order placement, order cancellation, position sizing, account allocation, automated trading, or any IBKR account mutation capability in backend code, frontend UI, tests, docs, or Codex prompts.

## Runtime Defaults

- IBKR host: `127.0.0.1`
- IBKR paper trading port: `7497`
- IBKR client ID: `1001`
- Default timeframe: `5m`
- Codex model: Codex CLI default, optionally overridable with `CODEX_MODEL`
- Real agent analysis requires an installed and authenticated `codex` CLI

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
- `internal/scheduler`: timeframe scheduling and next-run logic.
- `internal/analysis`: feature derivation and bounded analysis queue.
- `internal/agent`: Go client for local `codex exec` analysis and tests for the Codex CLI boundary.
- `frontend`: React TypeScript workspace UI using Wails-generated bindings in `frontend/wailsjs`.
- `docs/manual-acceptance.md`: manual acceptance flow for TWS or IB Gateway paper trading.
- `docs/superpowers/specs` and `docs/superpowers/plans`: original design and milestone plans.

## Safety Rules

- Keep `market.MarketDataProvider` read-only.
- Do not call IBKR APIs such as `PlaceOrder`, `CancelOrder`, `ReqAccountUpdates`, `ReqPositions`, or `ReqOpenOrders`.
- Do not add UI controls that imply executing, approving, sizing, allocating, or automating trades.
- Analysis output may include directional bias, entry zone, stop loss, take profit, risk-reward, confidence, summary, and invalidation notes, but it must remain advisory.
- Preserve the mock market provider so tests and local development do not require TWS/Gateway.

## Development Expectations

- Prefer small, focused changes that match existing Go, React, and TypeScript patterns.
- Add or update tests when behavior changes.
- For backend behavior, start with Go tests under the relevant `internal/...` package.
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
- Confirm there is no order placement UI, backend method, or IBKR order call
