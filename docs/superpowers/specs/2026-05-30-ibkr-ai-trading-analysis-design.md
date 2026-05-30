# IBKR AI Trading Analysis Mac App Design

## Status

Approved for implementation planning.

## Goal

Build a Wails-based macOS desktop application that connects to local IBKR TWS or IB Gateway in read-only mode, analyzes a user-defined US stock watchlist on intraday bar closes, and uses the OpenAI Agents SDK to produce structured price-action analysis: long, short, or neutral, plus entry zone, stop loss, take profit, risk-reward, confidence, and invalidation conditions.

This application is an analysis assistant only. Version 1 must not place orders, modify account state, or expose any trade execution capability.

## Confirmed Scope

- Platform: macOS desktop app.
- Desktop stack: Wails with Go backend and web frontend.
- Market data: IBKR TWS or IB Gateway running locally.
- IBKR mode: read-only market data and historical bars.
- Asset class: US stocks.
- Timeframes: `1m`, `5m`, `15m`, `1h`.
- Analysis mode: watchlist batch analysis.
- Trigger mode: analyze after each selected timeframe bar closes.
- AI layer: OpenAI Agents SDK for Node.js via `@openai/agents`.
- Output: structured, validated JSON consumed by the Go backend and displayed in the frontend.

## Out of Scope for Version 1

- Order placement.
- Automated trading.
- Position sizing and account allocation.
- Options chains, Greeks, and implied volatility analysis.
- Backtesting.
- Multi-account management.
- News, earnings, macro, or alternative data.
- Custom strategy language.

## Product Workflow

The first screen is the analysis workspace.

The left panel contains the watchlist editor and connection controls:

- IBKR host, port, and client ID.
- Default host: `127.0.0.1`.
- Default paper trading port: `7497`.
- Default client ID: `1001`.
- Watchlist input such as `NVDA, TSLA, AAPL`.
- Timeframe selector: `1m`, `5m`, `15m`, `1h`.
- Per-symbol status: market data status, last closed bar time, last analysis time.

The center panel contains the batch signal table. Each row represents one symbol and shows:

- Direction: `long`, `short`, or `neutral`.
- Current price.
- Entry zone.
- Stop loss.
- Take profit targets.
- Risk-reward.
- Confidence.
- Last generated time.
- Status: idle, queued, analyzing, complete, or failed.

The right panel contains the selected symbol's detailed analysis:

- Summary.
- Price-action rationale.
- Trend structure.
- Recent swing highs and lows.
- Support and resistance context.
- Breakout, breakdown, or retest context.
- Invalidation condition.
- Error details when the latest analysis failed.

The UI must clearly mark the output as analysis assistance, not financial advice and not automated trading.

## Architecture

The application is split into four bounded runtime areas.

### Wails Frontend

The frontend provides the desktop workspace, state views, controls, and analysis history display. It should not call IBKR or OpenAI directly. It only calls Wails-bound Go methods and subscribes to backend events.

Responsibilities:

- Edit and persist watchlist settings.
- Display IBKR connection state.
- Display market data status.
- Display batch analysis status.
- Display validated agent results.
- Allow manual refresh of settings and view state.

### Go Backend

The Go backend owns local application state, IBKR integration, bar aggregation, scheduling, validation, persistence, and communication with the Node agent worker.

Responsibilities:

- Connect to TWS or IB Gateway.
- Request and subscribe to read-only market data.
- Normalize bars into application models.
- Maintain in-memory market cache.
- Detect selected timeframe bar closes.
- Queue watchlist batch analysis jobs.
- Enforce agent concurrency limits.
- Validate agent JSON output before emitting it to the frontend.
- Persist local settings and recent analysis history.

### IBKR Connector

The IBKR connector implements a `MarketDataProvider` interface so the rest of the app can use either the real IBKR provider or a mock provider.

Required capabilities:

- Connect.
- Disconnect.
- Report connection state.
- Subscribe to market data for a symbol.
- Request recent historical bars for a symbol and timeframe.
- Emit normalized bar updates or closed bars.

The connector must not include order placement methods in version 1.

### Node Agent Worker

The Node worker uses `@openai/agents` and is invoked by the Go backend as a local child process or local IPC worker. The worker receives structured market summaries and returns structured JSON.

Responsibilities:

- Define the trading analysis agent.
- Accept one symbol analysis request at a time.
- Run the agent with constrained instructions.
- Return only JSON matching the output schema.
- Include no IBKR credentials.
- Include no account access.
- Include no order tools.

## Data Flow

1. User configures IBKR connection and watchlist.
2. Go backend connects to local TWS or IB Gateway.
3. Go backend subscribes to each symbol and loads enough historical bars for the selected timeframe.
4. Market data enters the market cache.
5. The scheduler detects a completed bar for the active timeframe.
6. The scheduler creates one analysis job per watchlist symbol.
7. The queue runs jobs with a small concurrency limit, initially `3`.
8. For each job, Go builds an agent input payload from recent bars and derived features.
9. The Node worker runs the OpenAI agent.
10. Go validates and stores the JSON result.
11. Go emits result updates to the Wails frontend.
12. The frontend updates the table and detail panel.

## Agent Input Schema

```json
{
  "symbol": "NVDA",
  "timeframe": "5m",
  "current_price": 125.34,
  "bars": [
    {
      "time": "2026-05-30T14:35:00-04:00",
      "open": 124.8,
      "high": 125.6,
      "low": 124.2,
      "close": 125.3,
      "volume": 123456
    }
  ],
  "derived": {
    "session_high": 126.1,
    "session_low": 122.7,
    "recent_swing_highs": [126.1, 125.8],
    "recent_swing_lows": [123.2, 124.1],
    "atr": 0.82,
    "volume_context": "above_average"
  }
}
```

The initial implementation should compute a compact derived feature set:

- Session high.
- Session low.
- Recent swing highs.
- Recent swing lows.
- ATR.
- Volume context.
- Last close relative to recent range.

## Agent Output Schema

```json
{
  "direction": "long",
  "entry_zone": {
    "low": 125.1,
    "high": 125.5
  },
  "stop_loss": 124.2,
  "take_profit": [126.4, 127.2],
  "risk_reward": 2.1,
  "confidence": 0.68,
  "summary": "Price reclaimed the prior high and held above short-term support.",
  "price_action": [
    "Higher low formed above prior support",
    "Breakout retest held on closing basis"
  ],
  "invalidated_if": "A 5m candle closes below 124.2",
  "generated_at": "2026-05-30T14:36:10-04:00"
}
```

Validation rules:

- `direction` must be one of `long`, `short`, or `neutral`.
- `confidence` must be between `0` and `1`.
- `risk_reward` must be positive when direction is `long` or `short`.
- `entry_zone.low` must be less than or equal to `entry_zone.high`.
- `stop_loss` and `take_profit` must be present for `long` and `short`.
- `neutral` may use an empty `take_profit` array and null-like numeric fields only if the Go model explicitly supports that.
- `generated_at` must parse as a timestamp.

## Scheduling Rules

The scheduler runs on closed bars, not on every tick.

- `1m`: analyze after each one-minute bar closes.
- `5m`: analyze after each five-minute bar closes.
- `15m`: analyze after each fifteen-minute bar closes.
- `1h`: analyze after each hourly bar closes.

The scheduler must avoid duplicate analysis for the same `symbol + timeframe + closedAt` key.

If a batch is already running for a timeframe, the scheduler should either queue the next batch or skip duplicates according to an explicit policy. Version 1 should queue only one pending batch per timeframe and drop repeated duplicate triggers for the same close time.

## Error Handling

### IBKR Not Connected

The frontend displays a disconnected state. The scheduler does not enqueue analysis.

### Missing Symbol Data

The affected row shows `No Data`. Other symbols continue.

### Agent Failure

The row shows `Agent Error`. The app keeps the previous valid analysis result visible and marks it as stale.

### Invalid Agent JSON

The Go backend rejects the result, stores a parse or validation error, and emits `AI Parse Error` to the frontend.

### Rate Limit or API Failure

The queue marks affected jobs failed, backs off before retrying, and does not block the full app.

## Persistence

Version 1 should persist:

- IBKR host.
- IBKR port.
- IBKR client ID.
- Watchlist symbols.
- Selected timeframe.
- Recent valid analysis results.

Secrets:

- OpenAI API key should come from environment or local secure configuration.
- The frontend must not receive the raw OpenAI API key.

## Testing Strategy

### Go Unit Tests

Cover:

- Watchlist parsing and normalization.
- Timeframe parsing.
- Bar close detection.
- Bar aggregation or normalization.
- Scheduler duplicate suppression.
- Scheduler queue limit behavior.
- Agent output validation.
- Mock market data provider behavior.

### Node Worker Tests

Cover:

- Input schema validation.
- Output schema validation.
- JSON-only response parsing.
- Fixed fixture requests that return deterministic mock outputs.

### Frontend Tests

Cover:

- Disconnected state.
- Connecting state.
- Connected state.
- Queued and analyzing states.
- Successful result state.
- Failed result state with stale prior output.
- Empty watchlist state.

### Manual Acceptance Test

1. Start TWS paper trading or IB Gateway locally.
2. Confirm API access is enabled in TWS or Gateway.
3. Launch the Wails app.
4. Connect to `127.0.0.1:7497` with `clientId=1001`.
5. Enter `NVDA, AAPL, TSLA`.
6. Select `5m`.
7. Wait for a five-minute bar close.
8. Confirm each symbol is queued and analyzed.
9. Confirm each successful symbol displays direction, entry zone, stop loss, take profit, risk-reward, confidence, summary, and invalidation condition.
10. Confirm no order placement UI or backend capability exists.

## Implementation Notes

- Use Wails for the desktop shell and Go-to-frontend bindings.
- Use Go for IBKR connectivity, scheduling, validation, persistence, and local app state.
- Use a Node worker for `@openai/agents` because the requested OpenAI Agents SDK integration is JavaScript/TypeScript oriented.
- Keep the AI agent stateless per analysis request in version 1.
- Use stable JSON contracts between Go and Node.
- Prefer mock data providers in tests so the app remains testable without a live IBKR session.

## References

- OpenAI Agents SDK TypeScript docs: https://openai.github.io/openai-agents-js/
- OpenAI Agents guide: https://openai.github.io/openai-agents-js/guides/agents/
- Wails introduction: https://wails.io/docs/introduction/
- Wails installation guide: https://wails.io/docs/gettingstarted/installation/

## Self-Review

- Placeholder scan: no placeholders remain.
- Internal consistency: architecture, data flow, scheduler, and testing sections all describe the same version 1 scope.
- Scope check: version 1 is focused on read-only watchlist analysis and does not include trading execution.
- Ambiguity check: analysis trigger, asset class, timeframes, AI SDK choice, and no-order safety boundary are explicit.
