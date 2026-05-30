# Manual Acceptance

## TWS Or IB Gateway Setup

1. Start TWS paper trading or IB Gateway locally.
2. Enable API access in TWS/Gateway settings.
3. Confirm the API socket port is `7497` for paper trading.
4. Confirm `codex` is installed and authenticated before launching the app.
5. Optional: set `PRICEACTION_KB_PATH=/path/to/priceaction` to test an external Markdown knowledge base. If unset, the app uses the bundled `priceaction/*.md` files.

## App Flow

1. Run `make dev`.
2. Connect to `127.0.0.1:7497` with client ID `1001`.
3. Enter watchlist `NVDA, AAPL, TSLA`.
4. Select `5m`.
5. Wait for a five-minute bar close or click Analyze for a manual run.
6. Confirm each symbol moves through queued/analyzing and ends complete or a scoped failure state.
7. For successful rows, confirm direction, current price, entry zone, stop loss, take profit, risk-reward, confidence, summary, price action, generated time, and invalidation condition are visible.
8. Confirm successful `price_action` details reference supplied PriceAction concepts such as trend, breakout, range, MTR, or climax when relevant.

## Failure Checks

- Disconnect IBKR and confirm scheduling is blocked.
- Use a missing/invalid symbol and confirm only that row shows no data.
- Force the Codex analysis runtime to return invalid JSON and confirm the UI shows an AI parse error while preserving prior valid stale output.
- Make `codex` unavailable or unauthenticated and confirm agent failures do not block the app.
- Set `PRICEACTION_KB_PATH` to a missing directory and confirm only analysis jobs fail with a PriceAction knowledge-base error.

## Safety Check

The UI and backend must expose no order placement, order cancellation, position sizing, account allocation, or automated trading capability.
Codex must receive Go-injected market data and PriceAction excerpts only; it must not be asked to browse repository files, inspect accounts, or prepare orders.
