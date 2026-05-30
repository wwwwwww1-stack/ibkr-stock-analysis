# OpenAI Agent Worker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local Node TypeScript worker that accepts one symbol analysis request and returns schema-validated JSON from OpenAI Agents SDK.

**Architecture:** The Go backend starts the worker as a local child process and communicates over NDJSON stdin/stdout. The worker has no IBKR credentials, account access, or order tools.

**Tech Stack:** Node 22, TypeScript, `@openai/agents@0.11.6`, `zod@4.4.3`, Vitest.

---

### Task 1: Worker Package

**Files:**
- Create: `agent-worker/package.json`
- Create: `agent-worker/tsconfig.json`
- Create: `agent-worker/src/schema.ts`
- Create: `agent-worker/src/schema.test.ts`

- [ ] Test valid input, invalid input, valid long output, valid neutral output, and invalid risk/confidence values.
- [ ] Use Zod schemas matching Go JSON contracts.
- [ ] Export TypeScript types from the schemas.
- [ ] Commit: `feat: add agent worker schemas`.

### Task 2: Agent Runner

**Files:**
- Create: `agent-worker/src/agent.ts`
- Create: `agent-worker/src/agent.test.ts`

- [ ] Test deterministic mock response without calling OpenAI.
- [ ] Configure default model `gpt-5.5`, overridable by `OPENAI_MODEL`.
- [ ] Use `Agent` and `run` from `@openai/agents`.
- [ ] Ensure the prompt forbids financial advice claims, order placement, and account access.
- [ ] Commit: `feat: add openai analysis agent runner`.

### Task 3: NDJSON CLI

**Files:**
- Create: `agent-worker/src/index.ts`
- Create: `agent-worker/src/protocol.ts`
- Create: `agent-worker/src/protocol.test.ts`

- [ ] Test request parsing, successful response envelope, error response envelope, and malformed JSON handling.
- [ ] Read one JSON request per line from stdin.
- [ ] Write one JSON response per line to stdout.
- [ ] Log diagnostics only to stderr.
- [ ] Commit: `feat: expose analysis worker protocol`.

### Task 4: Build Scripts

**Files:**
- Modify: `agent-worker/package.json`
- Modify: root docs or README

- [ ] Add `npm --prefix agent-worker test`, `build`, and `typecheck` scripts.
- [ ] Ensure `npm --prefix agent-worker test -- --run` passes.
- [ ] Ensure `npm --prefix agent-worker run build` passes.
- [ ] Commit: `chore: wire agent worker build`.

