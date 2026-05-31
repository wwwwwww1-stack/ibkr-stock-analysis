# Account-Aware A+ Position Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-31-account-aware-a-plus-position-management-design.md`

**Goal:** Extend the Watchlist A+ scanner with a read-only IBKR account snapshot, a user-configured maximum stock trade amount, deterministic advisory sizing envelopes, and optional Codex position-management guidance. The app remains advisory only and never places, stages, approves, cancels, or automates orders.

**Architecture:** Add a separate `internal/account` boundary for sanitized cash, buying power, and current stock positions. Keep `market.MarketDataProvider` market-data-only. Persist only the maximum stock trade amount in settings, keep the latest account snapshot in memory, compute sizing envelopes in Go before analysis, pass only sanitized account context to Codex, and validate Codex account-aware output against the backend envelope.

**Tech Stack:** Go, ibapi, Wails v2 bindings, React, TypeScript, Vite, Vitest, Testing Library, local Codex CLI.

**Safety Boundary:** Read-only account snapshot APIs are allowed only inside `internal/account`. Do not add `PlaceOrder`, `CancelOrder`, transmit flags, bracket order preparation, staged orders, `ReqOpenOrders`, order IDs, order payloads, order approval UI, buy/sell/submit/cancel controls, or automated trading behavior. Backtest must not use live account snapshots.

---

### Task 1: Lock Domain Contracts And Sizing Tests

**Files:**
- Modify: `internal/domain/models.go`
- Modify: `internal/domain/models_test.go`
- Add as needed: `internal/account/sizing.go`
- Add as needed: `internal/account/sizing_test.go`
- Modify: `frontend/src/types/domain.ts`

- [ ] Add domain models for `AccountSnapshotStatus`, `AccountSnapshotState`, `AccountSnapshot`, `AccountPosition`, `SizingEnvelope`, `AccountSnapshotContext`, `AccountPositionContext`, and `PositionManagementOutput`.
- [ ] Add `Settings.MaxStockTradeAmountUSD *float64` with JSON key `max_stock_trade_amount_usd,omitempty`; keep nil meaning not configured.
- [ ] Add TypeScript mirrors for every new account snapshot, sizing envelope, and position-management output shape.
- [ ] Add allowed constants for sizing status: `available`, `blocked_by_cash`, `blocked_by_cap`, `existing_position_over_cap`, `missing_account_snapshot`, `missing_directional_levels`, `not_a_plus`, and `no_trade`.
- [ ] Add allowed constants for advisory action: `no_trade`, `watch`, `consider_setup`, `manage_existing`, and `review_risk`.
- [ ] Add tests proving max stock trade amount accepts only positive finite numbers when provided, while nil remains valid.
- [ ] Add pure sizing tests for cash-limited, cap-limited, existing-position-over-cap, missing reference price, missing stop, non-A+ setup, and zero/negative clamp cases.
- [ ] Ensure `advisory_max_shares` is unavailable when the reference entry price is missing or invalid.
- [ ] Ensure every dollar/share field in a sizing envelope is non-negative and finite.
- [ ] Commit: `test: define account sizing domain contracts`.

### Task 2: Create The Read-Only Account Boundary

**Files:**
- Create: `internal/account/provider.go`
- Create: `internal/account/mock_provider.go`
- Create: `internal/account/provider_test.go`
- Modify: `internal/safety/safety_test.go`

- [ ] Add `SnapshotProvider` with `Snapshot(ctx context.Context) (domain.AccountSnapshot, error)`.
- [ ] Add `UnavailableProvider` for tests and local paths where account snapshots are disabled.
- [ ] Add `MockProvider` with configurable snapshots and errors for service tests.
- [ ] Keep the provider interface free of order, trade, allocation, mutation, and open-order concepts.
- [ ] Add a reflection safety test proving `account.SnapshotProvider` exposes only snapshot-read methods.
- [ ] Update source safety tests so `PlaceOrder`, `CancelOrder`, `ReqOpenOrders`, order transmit fields, and staged-order language remain forbidden everywhere.
- [ ] Update source safety tests so `ReqAccountSummary`, `CancelAccountSummary`, `ReqPositions`, and `CancelPositions` may appear only under `internal/account`.
- [ ] Keep `ReqAccountUpdates` forbidden everywhere unless a future spec explicitly allows it.
- [ ] Commit: `test: add read-only account provider boundary`.

### Task 3: Implement The IBKR Account Snapshot Adapter

**Files:**
- Create: `internal/account/ibkr_provider.go`
- Create: `internal/account/ibkr_provider_test.go`
- Modify as needed: `internal/market/ibkr_provider.go`
- Modify as needed: `internal/market/ibkr_provider_test.go`
- Modify as needed: `main.go`

- [ ] Implement an IBKR snapshot provider that uses only read-only account summary and position requests.
- [ ] Reuse the existing connected IBKR client path where possible without adding account methods to `market.MarketDataProvider`.
- [ ] If a small bridge is needed, expose only a minimal read-only client accessor from the concrete IBKR provider, not from the market-data interface.
- [ ] Read cash and buying power from account summary tags; normalize missing or non-finite values to a scoped snapshot error.
- [ ] Read current positions and include only USD stock positions with valid symbols.
- [ ] Ignore non-stock positions and include a scoped note or error detail that does not fail the whole snapshot when stock data is still usable.
- [ ] Immediately cancel account summary and position subscriptions after snapshot completion.
- [ ] Add timeout and context cancellation handling that cancels any in-flight account snapshot subscriptions.
- [ ] Add mock-callback tests for successful cash/buying power/position collection, timeout cancellation, non-stock filtering, and IBKR callback errors.
- [ ] Keep all direct IBKR account API calls inside `internal/account`.
- [ ] Commit: `feat: read sanitized ibkr account snapshots`.

### Task 4: Wire Account State Into The App Service

**Files:**
- Modify: `internal/app/service.go`
- Modify: `internal/app/service_test.go`
- Modify: `app.go`
- Modify: `frontend/src/api/backend.ts`
- Modify as needed after Wails generation: `frontend/wailsjs/go/main/App.d.ts`
- Modify as needed after Wails generation: `frontend/wailsjs/go/main/App.js`

- [ ] Extend `app.Service` to accept a separate `account.SnapshotProvider`.
- [ ] Add in-memory `AccountSnapshotState` to `domain.AppState`; do not persist account snapshots.
- [ ] Add a Wails-bound `RefreshAccountSnapshot() (domain.AppState, error)` method.
- [ ] On refresh, set account status to `loading`, call the account provider, then set `ready` or `failed` with a scoped error.
- [ ] Mark snapshots `stale` when disconnected or older than a testable stale threshold, such as five minutes.
- [ ] Preserve market-only analysis behavior when account status is unavailable, stale, failed, or max stock trade amount is nil.
- [ ] Persist only `settings.max_stock_trade_amount_usd`; verify store load/save keeps it and never stores account snapshots.
- [ ] Validate max stock trade amount in `SaveSettings` and return a Settings-scoped error for zero, negative, NaN, or infinite values.
- [ ] Update service tests for refresh success, refresh failure, stale-on-disconnect, max amount persistence, invalid max amount rejection, and market-only analysis fallback.
- [ ] Update Wails backend wrapper and frontend API interface for `refreshAccountSnapshot`.
- [ ] Run the Wails build or binding generation path after adding the bound method.
- [ ] Commit: `feat: add account snapshot app state`.

### Task 5: Build Account-Aware Agent Inputs And Output Validation

**Files:**
- Modify: `internal/domain/models.go`
- Modify: `internal/domain/models_test.go`
- Modify: `internal/analysis/queue.go`
- Modify: `internal/analysis/queue_test.go`
- Modify: `internal/app/service.go`
- Modify: `internal/app/service_test.go`

- [ ] Add `AccountSnapshotContext` to `domain.AgentInput` as optional JSON field `account_context`.
- [ ] Add optional `PositionManagementOutput` to `domain.AgentOutput` as JSON field `position_management`.
- [ ] When account snapshot and max stock trade amount are both available, build a sanitized per-symbol account context containing only cash, buying power, watchlist/selected-symbol positions, max amount, and the sizing envelope.
- [ ] Exclude account IDs, names, tax details, credentials, open orders, full account history, non-watchlist positions, and raw brokerage metadata from agent input.
- [ ] Build the sizing envelope before Codex is invoked; make it deterministic and testable without Codex.
- [ ] Update `analysis.Queue` or a domain helper to validate account-aware output against the input's sizing envelope after `Analyze` returns.
- [ ] Reject advisory shares above the envelope.
- [ ] Reject advisory notional above the envelope.
- [ ] Require `manual_review_required: true` whenever `position_management` is present.
- [ ] Reject position-management notes that contain direct execution or app-execution claims such as submit order, transmit, place order, cancel order, buy now, sell now, auto manage, or order ID.
- [ ] Mark invalid account-aware agent output as a failed analysis while preserving the previous stale result when one exists.
- [ ] Keep account-aware output optional when no valid account snapshot or max stock trade amount exists.
- [ ] Commit: `feat: validate account-aware analysis output`.

### Task 6: Update Codex Prompt And Strict Output Schema

**Files:**
- Modify: `internal/agent/client.go`
- Modify: `internal/agent/client_test.go`

- [ ] Extend the strict JSON schema with nullable `position_management`; keep every schema property listed in `required` for Codex strict-schema compatibility.
- [ ] Add schema constraints for allowed sizing status and advisory action values.
- [ ] Update prompt instructions conditionally: when no account context exists, keep the current "do not claim account access" rule.
- [ ] When account context exists, instruct Codex to use only the supplied sanitized account snapshot and sizing envelope.
- [ ] In all cases, instruct Codex not to browse files, request credentials, inspect orders, run tools, place orders, prepare orders, output order payloads, or claim execution.
- [ ] Require Codex to output `no_trade` when the market setup is not A+ even if account capacity exists.
- [ ] Require Codex to output account-blocked/manual-review guidance when a market setup is A+ but the sizing envelope is zero.
- [ ] Require all new natural-language position-management notes to use Simplified Chinese and manual-review language.
- [ ] Add prompt tests proving sanitized account context is included when present and absent when not present.
- [ ] Add prompt tests proving account IDs, credentials, open orders, order payloads, and local file paths are not present.
- [ ] Add schema tests for `position_management`, sizing status enum, advisory action enum, and manual review field.
- [ ] Commit: `feat: add account-aware codex contract`.

### Task 7: Surface Account Controls In Settings

**Files:**
- Modify: `frontend/src/types/domain.ts`
- Modify: `frontend/src/state/appState.ts`
- Modify: `frontend/src/state/appState.test.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/components/SettingsWorkspace.tsx`

- [ ] Add account snapshot state to the frontend default state and merge logic.
- [ ] Add max stock trade amount to Settings state and preserve it through local edits and saves.
- [ ] Add `Refresh Account Snapshot` handler that calls the backend and updates App state.
- [ ] Render an Account section in Settings after IBKR connection controls.
- [ ] Show snapshot status: unavailable, loading, ready, stale, or failed.
- [ ] Show cash available for stock trades, buying power, snapshot timestamp, and position count when available.
- [ ] Show scoped failure/stale copy without blocking market-only analysis.
- [ ] Add a positive finite USD input for maximum stock trade amount.
- [ ] Keep Save behavior explicit and do not refresh account snapshot implicitly when only editing the max amount.
- [ ] Add tests for rendering the account section, saving max stock trade amount, refresh loading/success/failure states, and invalid max amount errors.
- [ ] Add UI copy tests proving Settings does not render buy, sell, submit, transmit, approve order, cancel order, automate, or order controls.
- [ ] Commit: `feat: add account snapshot settings controls`.

### Task 8: Add Account-Aware Watchlist Columns And Detail Panels

**Files:**
- Modify: `frontend/src/components/SignalList.tsx`
- Modify: `frontend/src/components/AnalysisDetail.tsx`
- Modify: `frontend/src/components/AnalysisDetail.test.tsx`
- Modify: `frontend/src/components/WatchlistWorkspace.tsx`
- Modify: `frontend/src/App.test.tsx`
- Create as needed: `frontend/src/components/AccountContextPanel.tsx`
- Create as needed: `frontend/src/components/PositionManagementPanel.tsx`

- [ ] Add account-aware scanner columns for current position quantity, current position market value, available advisory notional cap, advisory max shares, and sizing status.
- [ ] Keep no-trade and account-blocked rows visible; do not filter out strong market setups that are blocked by cash or cap.
- [ ] Render account-aware cells as unavailable when the snapshot is missing, stale, failed, or max stock trade amount is not configured.
- [ ] Add a selected-symbol Account Context panel showing current position, average cost, market value, unrealized P&L, direction, user max amount, advisory notional cap, advisory max shares, and estimated risk when available.
- [ ] Add a Position Management panel for `position_management` output with advisory action, sizing status, manual-review-required state, and management notes.
- [ ] Use advisory copy only, such as "Advisory max shares within your configured cap" and "Review manually".
- [ ] Do not render execution copy such as "Buy 100 shares", "Sell now", "Submit order", or "Auto manage".
- [ ] Keep Backtest independent and unchanged, including the fixed `100` share historical simulation assumption.
- [ ] Add tests for current position rendering, advisory sizing rendering, unavailable account context, account-blocked A+ rows, non-A+ no-trade rows with available cash, and existing-position-over-cap copy.
- [ ] Add UI safety tests proving no forbidden execution controls or labels render in Watchlist detail or scanner rows.
- [ ] Commit: `feat: show advisory account context in watchlist`.

### Task 9: Layout And Styling Pass

**Files:**
- Modify: `frontend/src/App.css`
- Modify as needed: `frontend/src/style.css`
- Modify as needed: `frontend/src/components/SettingsWorkspace.tsx`
- Modify as needed: `frontend/src/components/AnalysisDetail.tsx`
- Modify as needed: `frontend/src/components/SignalList.tsx`

- [ ] Fit the Account section into the existing Settings workspace without creating nested cards.
- [ ] Keep dense scanner rows readable when account-aware columns are present.
- [ ] Allow the scanner table to scroll horizontally without widening the whole app.
- [ ] Keep the selected-symbol detail panel scannable by grouping market analysis, account context, and position-management guidance into clear sections.
- [ ] Make missing/stale account context visually distinct from valid advisory sizing without using execution-oriented colors or copy.
- [ ] Verify narrow viewport layout stacks sections without text overlap.
- [ ] Preserve the app's restrained data-first style and avoid decorative dashboard clutter.
- [ ] Commit: `style: fit account-aware scanner layout`.

### Task 10: Documentation, Acceptance, And Safety Verification

**Files:**
- Modify: `docs/manual-acceptance.md`
- Modify as needed: `README.md`
- Modify as needed only for fixes discovered by verification

- [ ] Update manual acceptance with the V2 read-only account snapshot refresh flow.
- [ ] Document that account snapshot reads cash, buying power, and stock positions only.
- [ ] Document that maximum stock trade amount is a user preference and hard advisory cap, not an order amount.
- [ ] Document that Backtest remains independent of live account data.
- [ ] Run `make test-go`.
- [ ] Run `npm test --prefix frontend -- --run`.
- [ ] Run `make test-frontend`.
- [ ] Run `make build`.
- [ ] Run a backend safety source scan: `rg -n "PlaceOrder|CancelOrder|ReqOpenOrders|ReqAccountUpdates|Transmit|OrderId|OrderID|Bracket|Place order|Cancel order" internal app.go main.go`.
- [ ] Run an account-boundary scan proving account read APIs appear only under `internal/account`: `rg -n "ReqAccountSummary|CancelAccountSummary|ReqPositions|CancelPositions" internal`.
- [ ] Run a frontend copy safety scan that does not flag allowed text such as "buying power": `rg -n "(?i)\\b(buy|sell)\\b|submit|transmit|approve order|place order|cancel order|auto manage|automate|order id|order payload" frontend/src`.
- [ ] If the IBKR account adapter was touched, run or explicitly defer the paper-trading account snapshot flow in `docs/manual-acceptance.md`.
- [ ] Run `git status --short` and confirm only intended files changed.
- [ ] Commit final verification fixes if any.
