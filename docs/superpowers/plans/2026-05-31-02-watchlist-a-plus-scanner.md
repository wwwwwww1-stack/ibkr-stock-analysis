# Watchlist A+ Scanner And Manual Trade Observations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-31-watchlist-a-plus-scanner-design.md`

**Goal:** Turn the existing Watchlist signals table into a cautious A+ scanner, keep no-trade reasons visible, and add local manual trade observations for review. This is V1 market-data-only assistance. It must not place, stage, approve, cancel, automate, or prepare trades, and it must not read IBKR account, cash, buying power, position, or order state for this V1 feature.

**Architecture:** Keep the scanner derived from existing per-symbol `SymbolState` and `AnalysisResult` data. Add local observation domain models and SQLite persistence. Expose only observation CRUD through Wails-bound service methods. Keep Codex V1 prompts limited to supplied market data, chart screenshots when provided, and Go-injected PriceAction excerpts. Implement scanner sorting/filtering and observation display in the Watchlist workspace without moving Backtest or Settings responsibilities.

**Compatibility:** `docs/superpowers/plans/2026-05-31-01-account-aware-a-plus-position-management.md` covers V2 account-aware behavior. Do not unwind existing account-boundary code if it already exists, but do not couple this V1 scanner plan to account snapshots, max trade amount, buying power, shares, holdings, or advisory sizing.

**Tech Stack:** Go, SQLite via `modernc.org/sqlite`, Wails v2 bindings, React, TypeScript, Vite, Vitest, Testing Library, local Codex CLI.

**Safety Boundary:** The scanner and observation panel are local analysis/review surfaces only. Do not add IBKR account reads, order reads, order IDs, order payloads, share sizing, cash inputs, buying power inputs, allocation fields, account mutation APIs, execution controls, staged-order state, or automated trade-management language.

---

### Task 1: Define Observation And Scanner Contracts

**Files:**
- Modify: `internal/domain/models.go`
- Modify: `internal/domain/models_test.go`
- Add as needed: `internal/domain/observations.go`
- Add as needed: `internal/domain/observations_test.go`
- Modify: `frontend/src/types/domain.ts`
- Add as needed: `frontend/src/state/scanner.ts`
- Add as needed: `frontend/src/state/scanner.test.ts`

- [ ] Add `TradeObservationStatus` constants: `watching`, `entered`, and `closed`.
- [ ] Add `TradeObservation` with `id`, `symbol`, `direction`, `status`, optional `observed_entry`, optional `planned_stop_loss`, optional `planned_take_profit`, `notes`, `created_at`, and `updated_at`.
- [ ] Add `ObservationHint` with a small enum-like status/type, symbol, severity, and review-only message text.
- [ ] Add validation for symbol parsing, long/short direction only, allowed status, positive finite prices, and optional long/short stop/target side checks when entry/stop/target are all present.
- [ ] Keep shares, cash, buying power, account, allocation, portfolio percentage, broker account ID, order ID, and computed sizing out of every V1 observation type.
- [ ] Add TypeScript mirrors for observation and hint types.
- [ ] Add frontend scanner row types for status `a_plus`, `pending`, `watch`, `no_trade`, `stale`, and `failed`.
- [ ] Add a pure scanner derivation helper that classifies and sorts current watchlist rows from `SymbolState`.
- [ ] Test scanner sorting order: fresh A+, pending, watch, no-trade, stale, failed.
- [ ] Test stale results never classify as active A+.
- [ ] Test failed/no-data rows remain visible with scoped errors.
- [ ] Commit: `test: define watchlist scanner observation contracts`.

### Task 2: Persist Local Trade Observations

**Files:**
- Modify: `internal/storage/settings.go`
- Modify: `internal/storage/settings_test.go`

- [ ] Add a `trade_observations` SQLite table with the columns from the spec.
- [ ] Add indexes on `(symbol, status, updated_at DESC)` and `(updated_at DESC)`.
- [ ] Add storage methods to list observations by symbol, save a new or existing observation, and close an observation by ID.
- [ ] Normalize symbols before persistence and reject invalid observation payloads before writes.
- [ ] Preserve `created_at` on updates and update `updated_at` on every save/close.
- [ ] Keep closed observations queryable for history while allowing callers to prefer active observations.
- [ ] Confirm deleting or editing the watchlist does not delete observation rows.
- [ ] Add migration-safe tests proving existing settings/history databases open after the new table is introduced.
- [ ] Add CRUD tests for create, update, close, reload from a new store instance, symbol filtering, and invalid input rejection.
- [ ] Commit: `feat: persist local trade observations`.

### Task 3: Expose Observation Methods Through The App Service

**Files:**
- Modify: `internal/app/service.go`
- Modify: `internal/app/service_test.go`
- Modify: `app.go`
- Modify: `frontend/src/api/backend.ts`
- Modify after Wails generation: `frontend/wailsjs/go/main/App.d.ts`
- Modify after Wails generation: `frontend/wailsjs/go/main/App.js`
- Modify after Wails generation: `frontend/wailsjs/go/models.ts`

- [ ] Add service methods `ListTradeObservations(symbol string)`, `SaveTradeObservation(input domain.TradeObservation)`, and `CloseTradeObservation(id int64)`.
- [ ] Add Wails-bound wrapper methods with the same behavior on `App`.
- [ ] Ensure the methods operate only on local storage and never call `market.MarketDataProvider`, `account.SnapshotProvider`, or IBKR provider methods.
- [ ] Scope errors as observation errors, for example invalid symbol, invalid direction, invalid price, or missing observation ID.
- [ ] Preserve the user's selected symbol and app state when an observation save fails.
- [ ] Add service tests proving list/save/close behavior, validation propagation, persistence across service restart, and no market/account provider calls.
- [ ] Update the frontend backend interface and Wails declarations.
- [ ] Run the Wails build or binding generation path after adding bound methods.
- [ ] Commit: `feat: expose local observation api`.

### Task 4: Harden The V1 Codex Scanner Prompt

**Files:**
- Modify: `internal/agent/client.go`
- Modify: `internal/agent/client_test.go`
- Modify as needed: `internal/domain/models_test.go`

- [ ] Add prompt wording that A+ requires good location, clear trigger, clear invalidation, target space, and multi-timeframe agreement.
- [ ] Ask Codex to keep `no_trade_reason` and `rejection_reasons` concise enough for scanner display.
- [ ] Preserve the default-to-neutral/no-trade rule unless the setup is clearly A+.
- [ ] Preserve the rule that Codex uses only supplied market data, optional supplied screenshot context, and injected PriceAction excerpts.
- [ ] Add tests proving the prompt does not include manual observations, cash, buying power, account balances, holdings, repository browsing, order state, or sizing fields for this V1 path.
- [ ] Add tests proving the strict schema still requires `setup_quality`, `no_trade_reason`, `rejection_reasons`, `invalidated_if`, and other existing output properties.
- [ ] Keep any V2 account-aware prompt tests isolated from this V1 scanner contract if V2 code is present.
- [ ] Commit: `test: harden v1 scanner prompt contract`.

### Task 5: Build The Scanner View In Watchlist

**Files:**
- Modify: `frontend/src/components/SignalList.tsx`
- Modify: `frontend/src/components/WatchlistWorkspace.tsx`
- Modify: `frontend/src/App.test.tsx`
- Add as needed: `frontend/src/components/ScannerStatusBadge.tsx`
- Add as needed: `frontend/src/components/SignalList.test.tsx`
- Modify: `frontend/src/App.css`

- [ ] Add an All/A+ only segmented filter for current watchlist scanner rows.
- [ ] Replace current-row ordering with the scanner sort order from Task 1 while keeping history records separate and still controlled by existing history filters.
- [ ] Show symbol, current price, scanner status, direction, setup quality, confidence, risk-reward, invalidation summary, and the first rejection or no-trade reason.
- [ ] Keep pending rows visible for queued/analyzing symbols.
- [ ] Keep no-trade and failed rows visible in All mode with concise reasons.
- [ ] In A+ only mode, hide non-A+ current rows without clearing the selected symbol or selected history state.
- [ ] Selecting a scanner row must continue to update the existing Watchlist detail panel.
- [ ] Do not render cash, shares, buying power, account allocation, order, approval, or automation controls in the scanner.
- [ ] Add Testing Library tests for row classification, sort order, A+ only filtering, selection, no-trade reason display, stale handling, and failed row display.
- [ ] Add UI safety tests proving scanner rows have no forbidden execution controls or labels.
- [ ] Commit: `feat: add watchlist scanner rows`.

### Task 6: Add The Manual Observation Panel

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/WatchlistWorkspace.tsx`
- Modify: `frontend/src/components/AnalysisDetail.tsx`
- Add: `frontend/src/components/ObservationPanel.tsx`
- Add: `frontend/src/components/ObservationPanel.test.tsx`
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/App.css`

- [ ] Load observations when the selected symbol changes and when an observation is saved or closed.
- [ ] Show one active editable observation per selected symbol, with access to closed/history rows if present.
- [ ] Render fields for direction, status, observed entry, planned stop, planned target, and notes.
- [ ] Infer symbol from the selected Watchlist symbol rather than allowing free-form symbol edits in the form.
- [ ] Keep unsaved form values visible when save fails.
- [ ] Add close behavior that marks an observation closed through the backend instead of deleting it.
- [ ] Hide observation editing when no symbol is selected.
- [ ] Do not show shares, cash, buying power, account value, portfolio percentage, broker account ID, order ID, position size, or any execution button.
- [ ] Add tests for load on selection, create, edit, close, validation errors, reload after save, and preserving unsaved inputs on failure.
- [ ] Add UI safety tests for forbidden form fields and forbidden execution labels.
- [ ] Commit: `feat: add manual observation panel`.

### Task 7: Derive Review-Only Observation Hints

**Files:**
- Modify as needed: `internal/domain/observations.go`
- Modify as needed: `internal/domain/observations_test.go`
- Add as needed: `frontend/src/state/observationHints.ts`
- Add as needed: `frontend/src/state/observationHints.test.ts`
- Modify: `frontend/src/components/ObservationPanel.tsx`
- Modify: `frontend/src/components/ObservationPanel.test.tsx`

- [ ] Derive hints from current price, latest analysis, and local observation fields only.
- [ ] Show when current price is inside or outside the AI entry zone.
- [ ] Show review-only hints when price is near or beyond the planned stop or planned target.
- [ ] Show a review-only hint when the latest scanner status moves from A+ to watch or no-trade for an active observation.
- [ ] Show missing-market-data and stale-analysis hints instead of inferring active price-relative status.
- [ ] Hide active hints for closed observations unless the user opens observation history.
- [ ] Ensure every hint uses manual-review copy and avoids direct action verbs such as buy, sell, hold, add, reduce, exit, approve, execute, or automate.
- [ ] Add long and short hint tests for entry-zone, stop, target, stale, failed/no-data, and closed-observation cases.
- [ ] Commit: `feat: show observation review hints`.

### Task 8: Layout And Styling Pass

**Files:**
- Modify: `frontend/src/App.css`
- Modify as needed: `frontend/src/style.css`
- Modify as needed: `frontend/src/components/SignalList.tsx`
- Modify as needed: `frontend/src/components/AnalysisDetail.tsx`
- Modify as needed: `frontend/src/components/ObservationPanel.tsx`

- [ ] Keep the Watchlist scanner dense and readable without making the whole workspace wider than the viewport.
- [ ] Allow the scanner table to scroll horizontally when columns exceed available width.
- [ ] Keep the selected-symbol detail panel scannable by grouping analysis, multi-timeframe context, and observations into clear sections.
- [ ] Avoid nested cards and preserve the existing restrained data-first style.
- [ ] Make stale, failed, no-trade, watch, and A+ statuses visually distinct without execution-oriented colors or copy.
- [ ] Verify narrow viewports stack sections without text overlap or clipped controls.
- [ ] Commit: `style: fit watchlist scanner layout`.

### Task 9: Documentation, Safety, And Verification

**Files:**
- Modify: `docs/manual-acceptance.md`
- Modify as needed: `README.md`
- Modify as needed only for fixes discovered by verification

- [ ] Update manual acceptance with the Watchlist scanner flow: save watchlist, run analysis, confirm scanner sort, filter A+ only, inspect no-trade reasons, and select rows.
- [ ] Add manual acceptance steps for creating, editing, closing, and reloading local observations.
- [ ] Document that observations are local user-entered context, not broker/account/order state.
- [ ] Document that V1 scanner and observations do not use cash, buying power, holdings, account state, or share sizing.
- [ ] Confirm Backtest remains independent and keeps its fixed `100` share historical simulation assumption.
- [ ] Run `make test-go`.
- [ ] Run `npm test --prefix frontend -- --run`.
- [ ] Run `make test-frontend`.
- [ ] Run `make build`.
- [ ] Run a backend safety source scan: `rg -n "PlaceOrder|CancelOrder|ReqOpenOrders|ReqAccountUpdates|Transmit|OrderId|OrderID|Bracket|staged order|stage order" internal app.go main.go`.
- [ ] Run a V1 observation storage/API scan proving observation code does not call account or market providers except through app state reads needed for selected symbol context.
- [ ] Run a frontend copy safety scan that excludes allowed historical Backtest labels and verifies the scanner/observation components do not render execution controls.
- [ ] If live IBKR market-data behavior changed, run or explicitly defer the paper-trading market-data flow in `docs/manual-acceptance.md`.
- [ ] Run `git status --short` and confirm only intended files changed.
- [ ] Commit final verification fixes if any.
