# Go Analysis Orchestrator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect market data, derived feature calculation, agent worker execution, validation, persistence, and Wails events into one backend orchestration layer.

**Architecture:** The orchestrator owns app state and queueing. Market providers and agent workers are replaceable interfaces so unit tests can run without IBKR or OpenAI.

**Tech Stack:** Go, Wails runtime events, Node child process worker pool.

---

### Task 1: Derived Features

**Files:**
- Create: `internal/analysis/features.go`
- Create: `internal/analysis/features_test.go`

- [ ] Test session high/low, recent swing highs/lows, ATR, volume context, and last-close range position.
- [ ] Keep feature calculation deterministic and independent of OpenAI.
- [ ] Commit: `feat: compute compact analysis features`.

### Task 2: Agent Client

**Files:**
- Create: `internal/agent/client.go`
- Create: `internal/agent/client_test.go`

- [ ] Test mock client success, invalid JSON rejection, worker error handling, and timeout cancellation.
- [ ] Implement a process client that runs `node agent-worker/dist/index.js`.
- [ ] Limit each worker process to one request at a time.
- [ ] Commit: `feat: add agent worker client`.

### Task 3: Analysis Queue

**Files:**
- Create: `internal/analysis/queue.go`
- Create: `internal/analysis/queue_test.go`

- [ ] Test max concurrency of 3.
- [ ] Test job status transitions: queued, analyzing, complete, failed.
- [ ] Test failures preserve previous valid result and mark it stale.
- [ ] Test retry/backoff for transient API/rate failures.
- [ ] Commit: `feat: add analysis job queue`.

### Task 4: App State And Wails Bindings

**Files:**
- Modify: `app.go`
- Create: `internal/app/service.go`
- Create: `internal/app/service_test.go`

- [ ] Test `GetState`, `SaveSettings`, `ConnectIBKR`, `DisconnectIBKR`, and `RunAnalysisNow` using mocks.
- [ ] Emit state update events with stable names: `connection:update`, `market:update`, `analysis:update`, `analysis:error`.
- [ ] Validate every agent result in Go before storage or frontend emission.
- [ ] Commit: `feat: orchestrate read-only analysis workflow`.

