# Frontend Analysis Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first-screen Wails React workspace for configuring IBKR, watching symbol status, and reviewing structured AI analysis.

**Architecture:** React state is hydrated from Wails `GetState` and updated through backend events. Frontend controls call Wails-bound Go methods only.

**Tech Stack:** React, TypeScript, Vite, Vitest, CSS modules or app CSS, Wails JS runtime.

---

### Task 1: Visual Design Contract

**Files:**
- Create: `docs/superpowers/specs/2026-05-30-frontend-workspace-ui-contract.md`

- [ ] Define compact desktop three-panel layout, status colors, table density, responsive collapse, empty/error/stale states, and disclaimer placement.
- [ ] Keep the UI operational and data-dense, not a marketing landing page.
- [ ] Commit: `docs: add workspace ui contract`.

### Task 2: Frontend State Layer

**Files:**
- Create: `frontend/src/api/backend.ts`
- Create: `frontend/src/state/appState.ts`
- Create: `frontend/src/state/appState.test.ts`

- [ ] Test state hydration, settings update, symbol selection, and event merge behavior.
- [ ] Mock Wails bindings in tests.
- [ ] Commit: `feat: add frontend backend state adapter`.

### Task 3: Workspace Components

**Files:**
- Create: `frontend/src/components/ConnectionPanel.tsx`
- Create: `frontend/src/components/SignalTable.tsx`
- Create: `frontend/src/components/AnalysisDetail.tsx`
- Create: `frontend/src/components/Disclaimer.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] Test disconnected, connecting, connected, empty watchlist, queued, analyzing, complete, failed, stale result, and no data render states.
- [ ] Implement three-panel desktop workspace and responsive mobile stack.
- [ ] Use controls that map to the domain: inputs, segmented timeframe selector, status rows, table, and detail panel.
- [ ] Commit: `feat: build analysis workspace UI`.

### Task 4: Visual And Interaction QA

**Files:**
- Modify: frontend styles
- Create/Modify: README

- [ ] Run frontend tests, type check, and build.
- [ ] Start a local dev server and inspect desktop and mobile viewports.
- [ ] Verify buttons, inputs, symbol selection, stale result display, and disclaimer visibility.
- [ ] Commit: `test: verify workspace interactions`.

