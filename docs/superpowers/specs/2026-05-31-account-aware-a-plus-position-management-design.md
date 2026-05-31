# Account-Aware A+ Scanner and Advisory Position Management Design

## Status

Draft for review.

## Goal

Extend the Watchlist A+ scanner into a V2 account-aware assistant. The app may read a sanitized IBKR account snapshot containing cash, buying power, and current stock positions. The user configures a maximum stock trade amount. Codex can use that bounded account context, together with market data and PriceAction excerpts, to produce advisory position-management output.

The app still must not place, cancel, stage, approve, or automate orders. Every position-management result is analysis assistance for manual review.

## Safety Boundary

Allowed in V2:

- Read cash and buying power from IBKR through read-only account APIs.
- Read current stock positions from IBKR through read-only position APIs.
- Let the user configure a maximum stock trade amount in USD.
- Pass a sanitized account snapshot and the maximum stock trade amount to Codex.
- Produce advisory sizing and position-management fields.
- Show calculations such as maximum advisory shares, estimated notional, risk dollars, and current exposure.

Forbidden:

- `PlaceOrder`, `CancelOrder`, order transmit flags, bracket order preparation, or staged orders.
- `ReqOpenOrders` or open-order-state workflows.
- Any UI button labeled or behaving like buy, sell, submit, transmit, approve order, place order, cancel order, or automate.
- Any backend method that mutates account, order, or portfolio state.
- Any claim that the app has executed or will execute a trade.
- Any Codex tool use, repository browsing, credential request, or account mutation.

## Product Behavior

### Account Snapshot

Settings gains an Account section after IBKR connection controls.

It shows:

- Snapshot status: unavailable, loading, ready, stale, failed.
- Cash available for stock trades.
- Buying power.
- Snapshot timestamp.
- Position count.
- Manual Refresh Account Snapshot action.
- Maximum stock trade amount input.

The maximum stock trade amount is a user preference, not an account value. It must be positive and finite. It is the hard notional cap for new or added stock exposure in the scanner. If the account snapshot is unavailable, scanner analysis can still run, but account-aware sizing fields must be omitted or marked unavailable.

### Watchlist Scanner

Watchlist remains the owner of the A+ scanner. The scanner still ranks symbols by A+ opportunity quality, but rows can now include account-aware columns:

- Current position quantity.
- Current position market value.
- Available advisory notional cap.
- Advisory max shares.
- Sizing status: available, blocked_by_cash, blocked_by_cap, existing_position_over_cap, unavailable.

The scanner must keep no-trade rows visible when account constraints block a setup. A strong market setup can be A+ but still account-blocked because cash or the configured cap is insufficient.

### Selected Symbol Detail

The selected detail gains an Account Context panel.

It shows:

- Current position for the selected symbol, if any.
- Average cost, market value, unrealized P&L, and position direction.
- User maximum stock trade amount.
- Available advisory notional cap.
- Advisory max shares based on the latest entry zone or current price fallback.
- Estimated risk dollars if entry and stop are available.

The detail panel must use advisory copy. Examples:

- "Advisory max shares within your configured cap."
- "Review manually: existing position already exceeds the configured stock cap."
- "Sizing unavailable until account snapshot is refreshed."

It must not show execution copy. Examples that must not appear:

- "Buy 100 shares."
- "Sell now."
- "Submit order."
- "Auto manage."

### Existing Position Management

If the account snapshot contains a current position in the selected symbol, Codex can produce position-management guidance.

Guidance may include:

- Whether the existing position is aligned with the current A+ thesis.
- Whether current price is near invalidation, stop, target, or entry zone.
- Whether adding exposure would exceed the configured maximum stock trade amount.
- Whether the position is oversized relative to the configured maximum stock trade amount.
- A manual-review action label such as watch, no new exposure, review risk, manage existing, or consider setup.

Guidance must not instruct the app to execute an order. If the guidance implies a possible add, trim, or exit, it must be phrased as a manual review item and include the reason and invalidation context.

## Sizing Model

The backend computes a deterministic sizing envelope before invoking Codex. Codex receives the envelope and may explain it, but the backend validates any returned sizing fields against the envelope.

Inputs:

- User maximum stock trade amount.
- Account cash and buying power.
- Existing position market value for the symbol.
- Latest current price.
- Latest AI entry zone, stop loss, and take-profit levels when available.

Derived fields:

- `account_notional_available_usd`: cash available for stocks, clamped to zero or above.
- `user_notional_cap_usd`: the configured maximum stock trade amount.
- `new_exposure_cap_usd`: `min(account_notional_available_usd, user_notional_cap_usd)`.
- `existing_symbol_exposure_usd`: absolute market value of the current position.
- `remaining_symbol_cap_usd`: `max(0, user_notional_cap_usd - existing_symbol_exposure_usd)`.
- `advisory_notional_cap_usd`: `min(new_exposure_cap_usd, remaining_symbol_cap_usd)`.
- `reference_entry_price`: entry-zone midpoint when available, otherwise current price.
- `advisory_max_shares`: floor of `advisory_notional_cap_usd / reference_entry_price`.
- `risk_per_share`: absolute difference between reference entry and stop loss when available.
- `estimated_risk_usd`: `risk_per_share * advisory_max_shares` when risk per share is available.

The sizing envelope must never be negative. If the reference entry price is missing or invalid, advisory max shares is unavailable.

## Agent Contract

Add account-aware fields to `AgentInput`:

```go
type AccountSnapshotContext struct {
    AvailableCashUSD       float64                    `json:"available_cash_usd"`
    BuyingPowerUSD         float64                    `json:"buying_power_usd"`
    SnapshotAt             time.Time                  `json:"snapshot_at"`
    Positions              []AccountPositionContext   `json:"positions"`
    MaxStockTradeAmountUSD float64                    `json:"max_stock_trade_amount_usd"`
    SizingEnvelope         *SizingEnvelopeContext     `json:"sizing_envelope,omitempty"`
}

type AccountPositionContext struct {
    Symbol          string  `json:"symbol"`
    Quantity        float64 `json:"quantity"`
    AverageCost     float64 `json:"average_cost"`
    MarketPrice     float64 `json:"market_price"`
    MarketValueUSD  float64 `json:"market_value_usd"`
    UnrealizedPnLUSD float64 `json:"unrealized_pnl_usd"`
}
```

Add account-aware fields to `AgentOutput`:

```go
type PositionManagementOutput struct {
    AccountAware              bool     `json:"account_aware"`
    SizingStatus              string   `json:"sizing_status"`
    AdvisoryAction            string   `json:"advisory_action"`
    AdvisoryMaxShares         *int     `json:"advisory_max_shares,omitempty"`
    AdvisoryNotionalCapUSD    *float64 `json:"advisory_notional_cap_usd,omitempty"`
    EstimatedRiskUSD          *float64 `json:"estimated_risk_usd,omitempty"`
    ExistingExposureUSD       *float64 `json:"existing_exposure_usd,omitempty"`
    ManagementNotes           []string `json:"management_notes"`
    ManualReviewRequired      bool     `json:"manual_review_required"`
}
```

Allowed `sizing_status` values:

- `available`
- `blocked_by_cash`
- `blocked_by_cap`
- `existing_position_over_cap`
- `missing_account_snapshot`
- `missing_directional_levels`
- `not_a_plus`
- `no_trade`

Allowed `advisory_action` values:

- `no_trade`
- `watch`
- `consider_setup`
- `manage_existing`
- `review_risk`

Validation rules:

- Account-aware output is optional when no account snapshot exists.
- Advisory shares must be less than or equal to the backend sizing envelope.
- Advisory notional must be less than or equal to the backend sizing envelope.
- Position-management output must set `manual_review_required` to true.
- Output text must not claim order execution or use direct execution commands.

Prompt rules:

- Codex may use supplied market data, PriceAction excerpts, sanitized account snapshot, and sizing envelope only.
- Codex must treat all sizing as advisory and manually reviewed.
- Codex must not request credentials, browse files, inspect orders, or call tools.
- Codex must not output order payloads, order IDs, transmit flags, or executable instructions.
- Codex must say no trade when the market setup is not A+ even if account capacity exists.
- Codex must say account-blocked when market setup is A+ but sizing envelope is zero.

## Backend Architecture

Add `internal/account`.

Interfaces:

```go
type SnapshotProvider interface {
    Snapshot(ctx context.Context) (domain.AccountSnapshot, error)
}
```

The IBKR adapter may use read-only account and position requests. It must immediately cancel any streaming subscriptions after snapshot completion where the IBKR API requires subscription-style reads.

The app service coordinates:

1. Connect to IBKR using existing settings.
2. Refresh account snapshot on demand and optionally after connect.
3. Store the latest sanitized snapshot in memory.
4. Persist user maximum stock trade amount in settings.
5. Build sizing envelopes per symbol before analysis.
6. Pass account context to Codex only when snapshot is ready and user cap is configured.
7. Validate Codex position-management output against backend sizing envelopes.

Account snapshots should not be persisted by default. Persist only the user's maximum stock trade amount.

## Frontend Architecture

Settings:

- Add Account section with Refresh Snapshot and maximum stock trade amount.
- Show snapshot status and timestamp.
- Show cash, buying power, and position count.
- Save maximum stock trade amount with settings.

Watchlist:

- Add account-aware scanner columns.
- Add selected-symbol Account Context panel.
- Add position-management output in the analysis detail.
- Disable account-aware fields when snapshot is stale or missing.

Backtest:

- Remains independent.
- Continues to use historical fixed-share assumptions unless a future backtest-specific spec changes it.
- Must not use live account snapshot for historical results.

## Data Flow

1. User connects to IBKR.
2. User opens Settings and refreshes account snapshot.
3. Backend reads cash, buying power, and current stock positions through the account provider.
4. User configures maximum stock trade amount.
5. User runs Watchlist analysis.
6. Backend builds market features, account context, and a sizing envelope per symbol.
7. Codex returns market analysis plus advisory position-management fields.
8. Backend validates any account-aware fields against the sizing envelope.
9. Frontend displays scanner ranking, account constraints, and manual-review guidance.
10. User decides outside the app what to do; the app never executes or stages orders.

## Error Handling

- Account snapshot failure does not block market-only analysis.
- Stale snapshot marks account-aware sizing unavailable until refresh.
- Missing max stock trade amount marks sizing unavailable.
- Invalid max stock trade amount shows a Settings-scoped validation error.
- If Codex returns advisory shares above the envelope, backend rejects the analysis output and records an agent error.
- If account position data has unsupported asset classes, ignore non-stock positions and show a scoped note.
- If cash or buying power is missing, sizing status is `missing_account_snapshot`.

## Privacy And Data Minimization

Only include fields Codex needs:

- Cash.
- Buying power.
- Current stock positions for watchlist symbols or the selected symbol.
- Maximum stock trade amount.
- Sizing envelope.

Do not include:

- Account ID unless required for debugging, and never in Codex prompts.
- Name, address, tax information, credentials, or statements.
- Open orders.
- Full account history.
- Non-watchlist positions unless the user explicitly enables portfolio-wide context in a future spec.

## Acceptance Criteria

- Settings allows the user to configure maximum stock trade amount.
- Settings can refresh a read-only account snapshot containing cash, buying power, and current stock positions.
- Watchlist analysis still works when account snapshot is unavailable.
- Account-aware analysis includes advisory sizing only when account snapshot and max stock trade amount are available.
- Advisory max shares never exceeds the backend sizing envelope.
- Existing positions are surfaced in selected-symbol detail.
- A+ market setups can be blocked by account cap or cash constraints.
- Non-A+ market setups remain no-trade even when cash is available.
- The app has no order placement, order cancellation, order staging, or automated trading controls.
- Backtest does not use live account data.

## Test Coverage

Backend:

- Account snapshot provider interface exposes only read-only methods.
- IBKR account adapter tests use mock callbacks for cash, buying power, and positions.
- Safety tests allow account snapshot APIs only inside `internal/account`.
- Safety tests continue forbidding `PlaceOrder`, `CancelOrder`, and `ReqOpenOrders`.
- Settings validation for maximum stock trade amount.
- Sizing envelope calculations for cash-limited, cap-limited, existing-position, and missing-level cases.
- Agent prompt tests include sanitized account context and exclude credentials, account IDs, open orders, and order payloads.
- Agent output validation rejects sizing above envelope.

Frontend:

- Settings renders max stock trade amount and account snapshot status.
- Watchlist renders account-aware scanner columns when snapshot is ready.
- Selected detail renders current position and advisory sizing fields.
- Account snapshot failure falls back to market-only analysis.
- UI does not render buy, sell, submit, transmit, approve order, cancel order, or automate controls.
- Section isolation remains intact.

Manual acceptance:

- TWS or IB Gateway paper mode.
- API socket enabled on `127.0.0.1:7497`.
- Read-only account snapshot refresh succeeds.
- Watchlist `NVDA, AAPL, TSLA`.
- Maximum stock trade amount configured, for example `$10,000`.
- Confirm A+ scanner can show advisory max shares and account-blocked states.
- Confirm no order placement UI, backend order method, staged order state, automated execution, or IBKR order mutation call exists.

## Relationship To Earlier Specs

This V2 spec supersedes the account/cash/sizing exclusions in `2026-05-31-watchlist-a-plus-scanner-design.md`. It does not supersede that spec's A+ scanner behavior, Watchlist ownership, or manual observation ideas.

## Spec Review

- Placeholder scan: no placeholders remain.
- Internal consistency: V2 permits read-only account snapshots and advisory sizing while preserving no-order execution.
- Scope check: this is a single V2 feature area for account-aware Watchlist analysis, not automated trading.
- Ambiguity check: order mutation, staged orders, open order workflows, and automated execution are explicitly out of scope.
