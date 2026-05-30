# Frontend Workspace UI Contract

## Layout

The first screen is a compact three-panel analysis workspace, not a landing page.

- Left panel: IBKR connection settings, watchlist editor, timeframe selector, per-symbol market status.
- Center panel: dense signal table for batch results.
- Right panel: selected symbol detail, latest error, stale-result warning, summary, price-action rationale, and invalidation.
- Bottom fixed note: analysis assistance only, not financial advice, no automated trading or order placement.

## States

- Connection: disconnected, connecting, connected, failed.
- Job: idle, queued, analyzing, complete, failed, no_data.
- Failed analysis keeps the previous valid result visible and labels it stale.
- Empty watchlist shows an operational empty state in the signal area.

## Visual Rules

- Use a restrained, data-first desktop style with readable table density.
- Use familiar form controls and segmented timeframe buttons.
- Keep cards to primary workspace panels only; do not nest cards.
- Status colors: green for complete/connected, blue for queued/analyzing, red for failed/no data, amber for stale.
- On narrow viewports, stack the three panels vertically and allow the signal table to scroll horizontally.

