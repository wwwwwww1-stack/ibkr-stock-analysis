# Frontend Workspace UI Contract

## Status

Draft for review.

## Goal

Make the desktop workspace feel like three independent product areas: Watchlist, Backtest, and Settings. The active directory section owns the main content area. Signals and selected-symbol analysis are part of Watchlist only. Backtest owns its own run controls and post-run result detail. Settings owns connection, chart-window, and timeframe configuration only.

The app remains read-only analysis assistance. Do not add order placement, order cancellation, position sizing, account allocation, automated trading, or any broker/account mutation UI.

## Workspace Structure

The first screen is an operational desktop workspace, not a landing page.

- Persistent app shell: product title, IBKR connection status, section navigation, bottom disclaimer, and transient toast errors.
- Section navigation: Watchlist, Backtest, Settings.
- Main content: rendered from the active section only.
- The inactive sections must not leave their primary content visible behind or beside the active section.
- Switching sections preserves each section's in-memory UI state during the session unless the user changes the inputs that logically invalidate that state.

## Section Ownership

### Watchlist

Watchlist is the only section that shows Signals and selected analysis details.

It contains:

- Watchlist symbol editor.
- Per-symbol market-data and analysis status.
- Manual analysis action for the selected symbol.
- Screenshot analysis action for the selected symbol.
- Scheduled analysis pause/resume action.
- Signals table with current results and history rows.
- Signals filters and CSV export.
- Selected symbol or selected history detail: latest error, stale-result warning, summary, price-action rationale, multi-timeframe context, and invalidation.

Rules:

- Selecting a current signal updates the Watchlist selected symbol detail.
- Selecting a history row updates the Watchlist detail to that historical analysis.
- Leaving Watchlist hides Signals and selected analysis detail.
- Returning to Watchlist restores the last Watchlist selection when possible.
- Watchlist empty state should appear inside the Watchlist section, not as a global page state.

### Backtest

Backtest is independent from Watchlist analysis detail.

It contains:

- Backtest symbol selector, defaulting to the current Watchlist selected symbol when first opened if available, otherwise the first saved watchlist symbol.
- Date range controls.
- Session time-window controls.
- Slippage and commission controls.
- Run Backtest action.
- Post-run result detail.

The share quantity remains a fixed simulation assumption, initially `100` shares. Do not add editable position sizing or account allocation controls.

Rules:

- Backtest must not show the Signals table.
- Backtest must not show the selected-symbol analysis detail.
- Before a run completes, the detail area shows a compact empty state such as "No backtest run."
- While a run is in progress, the run controls show loading state and the previous completed report may remain visible with a running indicator.
- After a successful run, the detail area shows the completed report for the exact request that was run.
- Changing symbol, date range, time window, slippage, or commission leaves the previous completed report visible only if it is clearly labeled as "Results from previous inputs." The primary state should tell the user to run Backtest again to update results.
- A failed run shows the scoped error in Backtest and does not replace the previous completed report.

Backtest report detail includes:

- Request summary: symbol, timeframe, date range, time window, share quantity assumption, slippage, and commission.
- Summary metrics: closed P&L, return, trade count, win rate, max drawdown, bar count, and tested window.
- Trade list: direction, setup quality, signal time, entry/exit times, fills, stop, targets, shares, gross P&L, fees, net P&L, return, exit reason, thesis, counterargument, summary, and invalidation.
- Exit legs when present.
- Skipped setups and skip reason counts when present.
- Open position detail when present.

### Settings

Settings is independent from Watchlist and Backtest result views.

It contains:

- IBKR connect/disconnect controls.
- Chart window refresh and selection for screenshot analysis.
- Timeframe selector.
- Save action for settings changes.

Rules:

- Settings must not show Signals.
- Settings must not show selected-symbol analysis detail.
- Settings must not show Backtest report detail.
- Timeframe changes affect future Watchlist analysis and future Backtest requests, but should not visually pretend to recalculate existing results.

## State Model

- `activeSection` is the single source of truth for which workspace section is visible.
- Watchlist owns `selectedSymbol`, selected history row, signal filters, and history pagination.
- Backtest owns backtest form state, loading state, scoped error, and the last completed report.
- Settings owns chart-window loading and pending settings state.
- Cross-section coupling should be explicit and minimal. For example, Backtest can initialize from the Watchlist selected symbol, but its active run request and result are Backtest state.
- Any state reset must be tied to a user-visible reason, such as deleting a symbol, changing an unsaved watchlist, or running a new backtest.

## Layout

Desktop:

- Left rail: section navigation and compact connection status.
- Active section content fills the rest of the window.
- Watchlist uses a three-area working layout: watchlist controls/status, Signals table, selected analysis detail.
- Backtest uses a two-area working layout: run controls and report detail.
- Settings uses a focused configuration layout with no analytical result panels.

Narrow viewports:

- Stack the active section's areas vertically.
- Allow Signals and Backtest trade tables to scroll horizontally.
- Keep primary actions near their relevant controls, not in a global floating area.

## Visual Rules

- Use a restrained, data-first desktop style with readable table density.
- Use familiar form controls and segmented timeframe buttons.
- Keep cards to primary workspace panels only; do not nest cards.
- Status colors: green for complete/connected, blue for queued/analyzing/running, red for failed/no data, amber for stale.
- Section switches should feel like changing workspaces, not merely opening an accordion in the left panel.

## Acceptance Criteria

- Clicking Watchlist shows Watchlist controls, Signals, and selected analysis detail.
- Clicking Backtest hides Signals and selected analysis detail, then shows Backtest controls and Backtest result detail.
- Running a backtest renders the completed report in the Backtest section.
- Clicking Settings hides Signals, selected analysis detail, and Backtest report detail, then shows configuration controls only.
- Switching away from Backtest and back preserves the last completed Backtest report. If inputs changed after that report, the report is labeled as results from previous inputs.
- Selecting a Signals history row affects only Watchlist detail and does not clear a completed Backtest report.
- Existing no-order, no-account-mutation safety rules remain visible in UI copy and enforced by available actions.

## Test Coverage

- Add frontend tests for section isolation: Watchlist-only Signals/detail, Backtest-only report detail, Settings-only configuration.
- Add frontend tests that a completed Backtest report persists across section switches.
- Add frontend tests that selecting a signal or history row does not replace or clear Backtest results.
- Add frontend tests that Settings controls do not render analysis or backtest result panels.
- Keep backend tests unchanged unless implementing this contract reveals a data-shape mismatch.
