# Wails Scaffold Contracts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the Wails v2 React TypeScript application shell and shared read-only domain contracts for the IBKR AI analysis assistant.

**Architecture:** The Go backend owns contracts, settings persistence, analysis validation, and Wails-bound application methods. The frontend consumes generated bindings and mirrors contract types in TypeScript without calling IBKR or OpenAI directly.

**Tech Stack:** Go 1.25, Wails v2.12.0, React, TypeScript, Vite, Vitest.

---

### Task 1: Scaffold Project

**Files:**
- Create/Modify: root Wails project files
- Create: `frontend/`
- Create: `wails.json`

- [ ] Install Wails CLI if missing: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`.
- [ ] Scaffold with React TypeScript: `wails init -n ibkr-stock-analysis -t react-ts`.
- [ ] If Wails refuses to scaffold into a non-empty directory, scaffold into a temporary directory and copy generated project files into the repo without overwriting `docs/`.
- [ ] Run `go test ./...` and `npm test --prefix frontend -- --run` after tests exist.
- [ ] Commit: `chore: scaffold wails app`.

### Task 2: Domain Contracts

**Files:**
- Create: `internal/domain/models.go`
- Create: `internal/domain/models_test.go`
- Create: `frontend/src/types/domain.ts`

- [ ] Add Go tests for timeframe parsing, watchlist normalization, and agent output validation.
- [ ] Implement `Timeframe`, `Bar`, `DerivedFeatures`, `AgentInput`, `EntryZone`, `AgentOutput`, `AnalysisResult`, `Settings`, and `AppState`.
- [ ] Support nullable numeric fields for neutral outputs with pointer fields.
- [ ] Mirror the contracts in TypeScript with matching JSON names.
- [ ] Run `go test ./internal/domain` and frontend type check.
- [ ] Commit: `feat: add read-only analysis contracts`.

### Task 3: Settings Persistence

**Files:**
- Create: `internal/storage/settings.go`
- Create: `internal/storage/settings_test.go`
- Modify: `app.go`

- [ ] Add tests for default settings, save/load round trip, corrupt JSON recovery, and history trimming.
- [ ] Persist settings/history under the user config directory in `ibkr-stock-analysis/settings.json`.
- [ ] Default IBKR host `127.0.0.1`, port `7497`, client ID `1001`, watchlist empty, timeframe `5m`.
- [ ] Ensure OpenAI API keys are never included in settings or frontend state.
- [ ] Commit: `feat: persist local analysis settings`.

### Task 4: Safety Boundary

**Files:**
- Create: `internal/safety/safety_test.go`
- Modify: app/backend files as needed

- [ ] Add tests that scan exported Go method names and source text for forbidden trading methods: `PlaceOrder`, `CancelOrder`, `Order`, `Trade`, `PositionSize`, `AccountAllocation`.
- [ ] Keep version 1 backend methods read-only: settings, state, connect, disconnect, manual analysis trigger.
- [ ] Add README safety note that the app is analysis assistance only.
- [ ] Run `go test ./...`.
- [ ] Commit: `test: enforce read-only trading boundary`.

