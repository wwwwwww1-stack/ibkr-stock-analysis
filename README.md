# IBKR Stock Analysis

Wails-based macOS desktop app for read-only IBKR market data analysis with an OpenAI Agents SDK worker.

Version 1 is analysis assistance only. It does not place orders, cancel orders, size positions, allocate accounts, or expose automated trading controls.

## Requirements

- Go 1.25+
- Node 22+
- npm 11+
- Wails CLI v2.12.0: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`
- Local TWS or IB Gateway for live market data
- `OPENAI_API_KEY` for real agent analysis

The IBKR adapter uses `github.com/scmhub/ibapi@v0.10.44` because `v0.10.46` currently requires Go 1.26, which is not available in this environment.

## Setup

```bash
npm install --prefix frontend
npm install --prefix agent-worker
npm --prefix agent-worker run build
```

## Development

```bash
make test
make dev
```

Useful focused checks:

```bash
make test-go
make test-agent
make test-frontend
make build
```

## Runtime Defaults

- IBKR host: `127.0.0.1`
- IBKR paper trading port: `7497`
- IBKR client ID: `1001`
- Default timeframe: `5m`
- Agent model: `gpt-5.5`, overridable with `OPENAI_MODEL`

## Manual Acceptance

See [docs/manual-acceptance.md](docs/manual-acceptance.md).

