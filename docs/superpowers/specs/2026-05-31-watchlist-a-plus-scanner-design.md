# Watchlist A+ Scanner and Manual Trade Observation Design

## Status

Draft for review.

## Goal

Upgrade Watchlist analysis into an A+ opportunity scanner that behaves more like a cautious discretionary trader: it should rank the watchlist by the quality of current market setups, make no-trade reasons obvious, and keep the user focused on only the cleanest opportunities.

Add optional manual trade observations so the user can record personal context such as an observed entry, direction, stop, target, and notes. These observations are local, user-entered context for follow-up review. They are not broker state, not account state, and not inputs for position sizing.

The app remains read-only analysis assistance. It must not place orders, cancel orders, size positions, allocate accounts, automate trading, or read IBKR account, cash, buying power, position, or order state.

## Non-Goals

- No order placement, order cancellation, bracket order preparation, or trade approval UI.
- No editable share quantity, cash balance, buying power, affordability, allocation, or "how many shares should I buy" workflow.
- No IBKR account APIs such as `ReqAccountUpdates`, `ReqPositions`, `ReqOpenOrders`, `PlaceOrder`, or `CancelOrder`.
- No Codex prompt access to user cash, manual trade observations, broker holdings, account balances, or repository files.
- No change to Backtest's fixed `100` share simulation assumption.
- No automatic trade management. Manual observations may show deterministic status hints, but never issue hold, sell, buy, add, reduce, or exit instructions.

## Approach

Use the existing per-symbol Codex analysis as the scanner source of truth. The backend already asks Codex to default to neutral unless the setup is clearly A+ and returns structured `setup_quality`, `direction`, `no_trade_reason`, `rejection_reasons`, and invalidation fields.

The selected design is:

- Derive scanner rows from existing analysis results.
- Add scanner-specific prompt emphasis, but keep Codex market-only.
- Add local manual trade observations that are displayed beside the selected symbol.
- Compute follow-up hints deterministically from current price, latest analysis, and the user's observation fields.

Rejected alternatives:

- Backtest evidence in scanner: useful later, but the user chose the simpler A+ scanner path first.
- Cash-aware AI sizing: rejected because it is account allocation and position sizing.
- Passing manual trade observations into Codex: rejected because current safety rules limit Codex prompts to supplied market data and PriceAction excerpts.

## Product Behavior

### A+ Scanner

Watchlist gains a scanner mode inside the existing Watchlist section. It does not create a new workspace section.

Each watchlist symbol receives a derived scanner status:

- `a_plus`: latest analysis is fresh, directional, and `setup_quality` is `a_plus`.
- `watch`: latest analysis is directional but below A+, or the setup is close but rejected.
- `no_trade`: latest analysis is neutral or `setup_quality` is `none`.
- `stale`: latest valid result exists but is stale for the active timeframe.
- `failed`: latest market data or agent analysis failed.
- `pending`: queued or analyzing.

The scanner view sorts rows in this order:

1. Fresh A+ setups.
2. Pending analysis.
3. Watch setups.
4. No-trade rows.
5. Stale rows.
6. Failed rows.

Rows show symbol, current price, status, direction, setup quality, confidence, risk-reward, invalidation summary, and the first rejection or no-trade reason. A filter lets the user show all rows or only A+ rows.

The selected symbol detail continues to show the full current analysis, including market regime, thesis, counterargument, price-action rationale, invalidation, and multi-timeframe context.

### Manual Trade Observations

The selected symbol detail gains a local "Observation" panel. The user can create or edit one active observation per symbol.

Fields:

- Symbol, inferred from selected Watchlist symbol.
- Direction: long or short.
- Status: watching, entered, closed.
- Observed entry price.
- Planned stop loss.
- Planned take profit.
- Notes.
- Created at and updated at timestamps.

Fields intentionally not allowed:

- Shares.
- Cash.
- Buying power.
- Account value.
- Portfolio percentage.
- Order ID.
- Broker account ID.
- Any computed position size.

The panel can show deterministic hints:

- Current price is inside or outside the latest AI entry zone.
- Current price is near or beyond the user's planned stop.
- Current price is near or beyond the user's planned target.
- Latest AI invalidation conflicts with the user's note.
- Latest scanner status changed from A+ to watch or no-trade.

Hints must use review language, such as "Review manually: price is below the planned stop." They must not say buy, sell, hold, add, reduce, exit, or size.

### Cash And 100-Share Requests

The app must not collect cash or buying power. It must not answer whether the user can afford `100` shares, and it must not suggest a quantity.

Backtest may continue showing its fixed `100` share simulation assumption because it is historical simulation, not user account allocation. Watchlist scanner and manual observations must not reuse that assumption as a live affordability or sizing hint.

## Architecture

### Domain

Add a local observation model:

```go
type TradeObservationStatus string

const (
    TradeObservationWatching TradeObservationStatus = "watching"
    TradeObservationEntered  TradeObservationStatus = "entered"
    TradeObservationClosed   TradeObservationStatus = "closed"
)

type TradeObservation struct {
    ID               int64                  `json:"id"`
    Symbol           string                 `json:"symbol"`
    Direction        Direction              `json:"direction"`
    Status           TradeObservationStatus `json:"status"`
    ObservedEntry    *float64               `json:"observed_entry,omitempty"`
    PlannedStopLoss  *float64               `json:"planned_stop_loss,omitempty"`
    PlannedTakeProfit *float64              `json:"planned_take_profit,omitempty"`
    Notes            string                 `json:"notes,omitempty"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
}
```

Validation:

- Symbol must pass existing watchlist symbol parsing.
- Direction must be long or short.
- Status must be watching, entered, or closed.
- Prices, when present, must be finite positive numbers.
- For long observations, stop should be below entry and target above entry when all are present.
- For short observations, stop should be above entry and target below entry when all are present.

Add frontend-mirrored TypeScript types for observations and derived scanner rows.

### Storage

Persist observations in the local SQLite database alongside analysis history.

Suggested table:

```sql
CREATE TABLE IF NOT EXISTS trade_observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  symbol TEXT NOT NULL,
  direction TEXT NOT NULL,
  status TEXT NOT NULL,
  observed_entry REAL,
  planned_stop_loss REAL,
  planned_take_profit REAL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

Indexes:

- `symbol, status, updated_at DESC`
- `updated_at DESC`

Deleting a watchlist symbol does not delete observations. The observation remains local history and appears again if the symbol is re-added.

### Backend Service

Add Wails-bound methods:

- `ListTradeObservations(symbol string) ([]domain.TradeObservation, error)`
- `SaveTradeObservation(input domain.TradeObservation) (domain.TradeObservation, error)`
- `CloseTradeObservation(id int64) (domain.TradeObservation, error)`

These methods operate only on local storage. They must not call IBKR.

Add a pure helper to derive observation hints:

```go
func DeriveObservationHints(
    observation domain.TradeObservation,
    symbol domain.SymbolState,
) []domain.ObservationHint
```

This helper uses only current price, latest analysis result, and the observation's local fields.

### Agent Prompt

Keep Codex market-only.

Prompt changes are limited to A+ scanner clarity:

- Re-emphasize that A+ requires clear location, trigger, invalidation, target space, and multi-timeframe agreement.
- Ask Codex to make `no_trade_reason` and `rejection_reasons` concise and actionable for scanner display.
- Preserve existing instructions that default to neutral unless the setup is clearly A+.
- Preserve existing instructions that Codex must not access accounts, portfolio data, positions, balances, or live brokerage state.

Do not add observation, cash, buying power, manual holding, or account fields to `AgentInput`.

### Frontend

Watchlist remains the owning section for signals and selected analysis.

Add:

- Scanner filter: All, A+ only.
- Scanner row status badges.
- A compact rejection/no-trade reason column.
- Observation panel in selected symbol detail.
- Observation edit form with direction, status, entry, stop, target, and notes.
- Observation hints below the form.

Do not add:

- Cash input.
- Shares input.
- Buying power display.
- Account allocation display.
- Buy, sell, hold, exit, add, reduce, approve, automate, or execute buttons.

## Data Flow

1. User saves a watchlist and connects to IBKR.
2. Existing manual or scheduled analysis runs per symbol.
3. Codex analyzes only supplied market data and PriceAction excerpts.
4. The frontend derives scanner rows from latest analysis results.
5. User optionally creates a manual observation for the selected symbol.
6. The backend persists the observation locally.
7. The frontend displays deterministic observation hints using current symbol state and the saved observation.
8. Future analysis updates may change scanner status and observation hints, but do not mutate the observation automatically.

## Error Handling

- Invalid observation prices show scoped form validation errors.
- Observation save failures show a Watchlist-scoped error and keep the user's unsaved form inputs visible.
- Missing market data shows "waiting for market data" hints instead of inferring price-relative status.
- Stale analysis cannot produce an active A+ scanner status.
- Failed analysis rows remain visible with the scoped failure reason.
- Observation hints are hidden for closed observations unless the user opens the observation history.

## Safety Requirements

- Safety tests must continue scanning for forbidden IBKR calls.
- New code must not introduce account, position, order, buying power, or cash APIs.
- New Codex prompt tests must assert that manual observations and cash-like fields are not present.
- UI tests must assert that Watchlist has no cash, shares, allocation, buy, sell, hold, exit, add, reduce, approve, automate, or execute controls.
- Copy must describe output as analysis assistance and manual review, not financial advice or automated trading.

## Acceptance Criteria

- Watchlist can show scanner rows sorted by current opportunity status.
- A+ only filter hides non-A+ rows without losing selected-symbol state.
- Non-A+ rows show the main no-trade or rejection reason.
- Selecting a scanner row updates the existing Watchlist detail.
- User can create, edit, close, and reload a manual observation for a symbol.
- Observation form does not include shares, cash, buying power, allocation, account, or order fields.
- Observation hints update when current price or latest analysis changes.
- Codex prompt remains limited to supplied market data and PriceAction excerpts.
- Backtest remains independent and keeps its fixed `100` share historical simulation assumption.
- Settings remains independent and does not show scanner detail or observation forms.

## Test Coverage

Backend:

- Domain validation for valid and invalid observations.
- Storage CRUD for observations, including persistence across reload.
- Service methods for listing, saving, and closing observations without IBKR calls.
- Observation hint derivation for long and short observations.
- Agent prompt tests for A+ scanner wording and absence of manual observation, cash, buying power, account, and position fields.
- Safety tests for forbidden IBKR APIs.

Frontend:

- Scanner sort and A+ only filtering.
- Scanner row selection updates Watchlist detail.
- Observation form validation and persistence through mocked backend.
- Observation hints for entry zone, stop, target, stale analysis, and failed/no-data states.
- Absence of cash, shares, allocation, account, and order controls.
- Section isolation remains intact: Backtest and Settings do not render scanner observation forms.

## Implementation Notes

- Keep scanner derivation mostly frontend-side at first because all required fields already exist in `SymbolState`.
- Add backend support only for observation persistence and deterministic hints if the hints are easier to test in Go.
- Do not change Backtest in this feature except for tests that confirm it remains independent.
- Keep generated Wails bindings stable by regenerating only after domain or service method changes.

## Spec Review

- Placeholder scan: no placeholders remain.
- Internal consistency: scanner uses Codex market-only output; observations are local deterministic context.
- Scope check: this is one focused Watchlist feature with local persistence, not a trading execution system.
- Ambiguity check: cash, buying power, share quantity, and AI position sizing are explicitly out of scope.
