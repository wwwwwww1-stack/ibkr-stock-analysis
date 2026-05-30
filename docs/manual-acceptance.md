# Manual Acceptance

## TWS Or IB Gateway Setup

1. Start TWS paper trading or IB Gateway locally.
2. Enable API access in TWS/Gateway settings.
3. Confirm the API socket port is `7497` for paper trading.
4. Confirm `codex` is installed and authenticated before launching the app.

## App Flow

1. Run `make dev`.
2. Connect to `127.0.0.1:7497` with client ID `1001`.
3. Enter watchlist `NVDA, AAPL, TSLA`.
4. Select `5m`.
5. Wait for a five-minute bar close or click Analyze for a manual run.
6. Confirm each symbol moves through queued/analyzing and ends complete or a scoped failure state.
7. For successful rows, confirm direction, current price, entry zone, stop loss, take profit, risk-reward, confidence, summary, price action, generated time, and invalidation condition are visible.

## Failure Checks

- Disconnect IBKR and confirm scheduling is blocked.
- Use a missing/invalid symbol and confirm only that row shows no data.
- Force the agent worker to return invalid JSON and confirm the UI shows an AI parse error while preserving prior valid stale output.
- Make `codex` unavailable or unauthenticated and confirm agent failures do not block the app.

## Safety Check

The UI and backend must expose no order placement, order cancellation, position sizing, account allocation, or automated trading capability.
