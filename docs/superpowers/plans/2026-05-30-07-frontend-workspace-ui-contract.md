# Frontend Workspace UI Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-30-frontend-workspace-ui-contract.md`

**Goal:** Make Watchlist, Backtest, and Settings behave as independent desktop workspace sections. The active section owns the main content area; inactive sections must not leak Signals, selected-symbol analysis, Backtest reports, or Settings controls into the visible workspace.

**Architecture:** Promote section selection and section-owned UI state to the App-level workspace boundary. Keep Wails backend calls unchanged unless a frontend data-shape mismatch is discovered. Watchlist owns signals and analysis detail, Backtest owns run controls and completed reports, and Settings owns connection/chart/timeframe controls.

**Tech Stack:** React, TypeScript, Vite, Vitest, Testing Library, Wails JS bindings, app CSS.

**Safety Boundary:** Preserve the read-only analysis scope. Do not add order placement, cancellation, position sizing, account allocation, automated trading, or IBKR account mutation UI, backend calls, tests, docs, or prompts. Backtest share quantity remains the fixed simulation assumption of `100` shares.

---

### Task 1: Lock Section Isolation With Tests

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify as needed: `frontend/src/components/AnalysisDetail.test.tsx`

- [ ] Add or tighten tests proving Watchlist shows Watchlist controls, Signals, and selected analysis detail.
- [ ] Add tests proving Backtest hides Signals and selected analysis detail, then shows Backtest controls plus a compact "No backtest run." detail state before any run.
- [ ] Add tests proving Settings hides Signals, selected analysis detail, and Backtest report detail, then shows only connection, chart-window, timeframe, and save controls.
- [ ] Add a regression test proving Settings controls do not render analysis actions, Signals, or Backtest result panels.
- [ ] Keep existing automatic connection, watchlist editor, screenshot analysis, history, and CSV export tests passing.
- [ ] Commit: `test: lock workspace section isolation`.

### Task 2: Move Workspace Section State To App

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/ConnectionPanel.tsx`
- Modify as needed: `frontend/src/state/appState.ts`

- [ ] Introduce an App-level `activeSection` state with values `watchlist`, `backtest`, and `settings`.
- [ ] Make `activeSection` the single source of truth for the visible workspace section.
- [ ] Remove section selection state from `ConnectionPanel`, or rename/split the component so it no longer owns hidden workspace content.
- [ ] Preserve each section's in-memory UI state when switching sections.
- [ ] Keep Watchlist selection restoration logic scoped to Watchlist: current symbol selection and selected history row should return when possible.
- [ ] Stop clearing a completed Backtest report when a current signal or history row is selected in Watchlist.
- [ ] Reset Watchlist selection only for user-visible reasons such as deleting a symbol or saving a watchlist that no longer contains the selected symbol.
- [ ] Commit: `feat: make active workspace section app-owned`.

### Task 3: Refactor The Workspace Shell

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/ConnectionPanel.tsx`
- Create or modify as needed: `frontend/src/components/WorkspaceNav.tsx`
- Create or modify as needed: `frontend/src/components/WatchlistWorkspace.tsx`
- Create or modify as needed: `frontend/src/components/SettingsWorkspace.tsx`

- [ ] Render a persistent shell with product title, compact IBKR connection status, section navigation, bottom disclaimer, and transient toast errors.
- [ ] Render only the active section inside the main content area.
- [ ] Keep inactive primary content unmounted or hidden from the accessibility tree so tests cannot find it by role or label.
- [ ] Move Watchlist controls/status, `SignalList`, and `AnalysisDetail` into the Watchlist section.
- [ ] Keep Watchlist empty state inside the Watchlist section instead of as a global page state.
- [ ] Move Settings connection, chart-window, timeframe, and save controls into the Settings section.
- [ ] Keep section switches visually and structurally equivalent to changing workspaces, not opening an accordion beside globally visible panels.
- [ ] Commit: `feat: split frontend workspace sections`.

### Task 4: Make Backtest State Independent

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/BacktestSidebar.tsx`
- Create or modify as needed: `frontend/src/components/BacktestWorkspace.tsx`
- Modify: `frontend/src/types/domain.ts` only if an existing report/request shape mismatch is discovered

- [ ] Add a Backtest symbol selector that defaults on first open to the current Watchlist selection if available, otherwise the first saved watchlist symbol.
- [ ] Keep Backtest symbol, date range, session time window, slippage, commission, loading state, scoped error, last completed report, and last completed request as Backtest-owned state.
- [ ] Run Backtest with fixed `share_quantity: 100`; display it as an assumption, not as an editable position sizing control.
- [ ] Before the first successful run, show a compact "No backtest run." detail area.
- [ ] While a run is in progress, put the run controls in a loading state and keep the previous completed report visible with a running indicator when one exists.
- [ ] After a successful run, render the completed report for the exact request that was run.
- [ ] When Backtest inputs change after a completed run, keep the previous report visible only with a clear "Results from previous inputs" label and primary copy telling the user to run Backtest again.
- [ ] On run failure, show a scoped Backtest error and do not replace the previous completed report.
- [ ] Ensure selecting a current signal or history row in Watchlist does not replace, clear, or relabel the completed Backtest report.
- [ ] Commit: `feat: isolate backtest workspace state`.

### Task 5: Complete Backtest Report Detail

**Files:**
- Modify: `frontend/src/components/BacktestSidebar.tsx`
- Modify: `frontend/src/App.test.tsx`

- [ ] Show request summary: symbol, timeframe, date range, time window, share quantity assumption, slippage, and commission.
- [ ] Preserve summary metrics: closed P&L, return, trade count, win rate, max drawdown, bar count, and tested window.
- [ ] Preserve trade list details: direction, setup quality, signal time, entry/exit times, fills, stop, targets, shares, gross P&L, fees, net P&L, return, exit reason, thesis, counterargument, summary, and invalidation.
- [ ] Preserve exit legs when present.
- [ ] Preserve skipped setups and skip reason counts when present.
- [ ] Preserve open position detail when present.
- [ ] Add or update tests for request summary, previous-input labeling, run loading behavior, failed-run preservation, and no editable share quantity control.
- [ ] Commit: `feat: complete backtest report detail`.

### Task 6: Tighten Settings Behavior

**Files:**
- Modify: `frontend/src/components/SettingsWorkspace.tsx`
- Modify as needed: `frontend/src/components/ConnectionPanel.tsx`
- Modify: `frontend/src/App.test.tsx`

- [ ] Keep Settings limited to connect/disconnect, chart-window refresh/selection, timeframe selection, and save.
- [ ] Make timeframe changes affect future Watchlist analysis and future Backtest requests without implying existing analysis or completed Backtest results were recalculated.
- [ ] Keep chart-window loading and selection scoped to Settings.
- [ ] Keep Settings free of Signals, selected-symbol analysis detail, Backtest report detail, analysis actions, and screenshot analysis actions.
- [ ] Preserve IBKR defaults and the existing automatic connection behavior.
- [ ] Commit: `feat: constrain settings workspace`.

### Task 7: Layout And Visual Pass

**Files:**
- Modify: `frontend/src/App.css`
- Modify as needed: `frontend/src/style.css`

- [ ] Implement desktop shell layout: left rail for navigation/status and active section content filling the remaining window.
- [ ] Implement Watchlist as a three-area working layout: controls/status, Signals table, selected analysis detail.
- [ ] Implement Backtest as a two-area working layout: run controls and report detail.
- [ ] Implement Settings as a focused configuration layout with no analytical result panels.
- [ ] On narrow viewports, stack each active section's areas vertically.
- [ ] Allow Signals and Backtest trade tables or dense report areas to scroll horizontally without widening the whole app.
- [ ] Use restrained data-first styling, readable density, familiar form controls, segmented timeframe buttons, and the required status colors.
- [ ] Avoid nested cards and keep cards/panels limited to primary workspace panels.
- [ ] Commit: `style: apply workspace section layout`.

### Task 8: Verification And Safety Scan

**Files:**
- Modify as needed only for fixes discovered by verification

- [ ] Run `npm test --prefix frontend -- --run`.
- [ ] Run `make test-frontend`.
- [ ] Run `make build` after Wails-bound type or layout-impacting changes.
- [ ] Run a source scan confirming no forbidden order/account mutation APIs or UI copy were introduced: `rg -n "PlaceOrder|CancelOrder|ReqAccountUpdates|ReqPositions|ReqOpenOrders|position sizing|account allocation|automated trading|place order|cancel order"`.
- [ ] If a live IBKR integration path was touched, run or explicitly defer the paper-trading flow in `docs/manual-acceptance.md`.
- [ ] Run `git status --short` and confirm only intended files changed.
- [ ] Commit final verification fixes if any.
