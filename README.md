# IBKR Stock Analysis

Wails-based macOS desktop app for read-only IBKR market data analysis with a local Codex CLI analysis runtime.

Version 1 is analysis assistance only. It does not place orders, cancel orders, size positions, allocate accounts, or expose automated trading controls.

## Requirements

- Go 1.26+
- Node 22+
- npm 11+
- Wails CLI v2.12.0: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`
- Codex CLI installed and authenticated for real AI analysis
- Local TWS or IB Gateway for live market data

## Setup

```bash
npm install --prefix frontend
```

## Development

```bash
make test
make dev
```

Useful focused checks:

```bash
make test-go
make test-frontend
make build
```

## Runtime Defaults

- IBKR host: `127.0.0.1`
- IBKR paper trading port: `7497`
- IBKR client ID: `1001`
- Default timeframe: `5m`
- Codex model: Codex CLI default, optionally overridable with `CODEX_MODEL`
- PriceAction knowledge: bundled from `priceaction/*.md`

For local knowledge-base experiments without rebuilding the app, point the runtime at another Markdown directory:

```bash
PRICEACTION_KB_PATH=/path/to/priceaction make dev
```

The Go backend selects and injects PriceAction excerpts into the Codex prompt. Codex analysis is instructed not to browse the repository or read `priceaction` files itself.

## Manual Acceptance

See [docs/manual-acceptance.md](docs/manual-acceptance.md).
