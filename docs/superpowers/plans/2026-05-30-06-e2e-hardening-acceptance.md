# E2E Hardening Acceptance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the app for local development, document setup, and verify the end-to-end read-only analysis workflow and safety boundary.

**Architecture:** The final pass verifies all runtime areas together: Wails shell, Go backend, IBKR adapter, Node worker, frontend, persistence, and manual acceptance docs.

**Tech Stack:** Go tests, Node/Vitest tests, Wails build/dev, README manual acceptance.

---

### Task 1: Developer Scripts

**Files:**
- Create: `Makefile`
- Modify: `README.md`

- [ ] Add `make test`, `make test-go`, `make test-agent`, `make test-frontend`, `make build`, and `make dev`.
- [ ] Document required local tools and Wails CLI installation.
- [ ] Commit: `chore: add developer test scripts`.

### Task 2: Failure Modes

**Files:**
- Add tests across backend/frontend as needed

- [ ] Verify disconnected state blocks scheduling.
- [ ] Verify missing symbol data affects only that symbol.
- [ ] Verify agent failure shows error while keeping stale previous result.
- [ ] Verify invalid agent JSON emits `AI Parse Error`.
- [ ] Verify rate/API failure marks jobs failed and does not block the app.
- [ ] Commit: `test: cover analysis failure modes`.

### Task 3: Manual Acceptance Guide

**Files:**
- Create: `docs/manual-acceptance.md`
- Modify: `README.md`

- [ ] Document TWS or IB Gateway paper setup with API enabled.
- [ ] Document test values: `127.0.0.1:7497`, client ID `1001`, `NVDA,AAPL,TSLA`, `5m`.
- [ ] Document expected row statuses and result fields after a five-minute close.
- [ ] Document that no order placement capability exists.
- [ ] Commit: `docs: add manual acceptance guide`.

### Task 4: Final Verification

**Files:**
- Modify as needed only for fixes discovered by verification

- [ ] Run `make test`.
- [ ] Run `make build` or document exact Wails build blocker.
- [ ] Run safety source scan for forbidden order/account mutation APIs.
- [ ] Run `git status --short`.
- [ ] Commit final fixes if any.

