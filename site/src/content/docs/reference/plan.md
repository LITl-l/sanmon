---
title: Implementation Plan
description: Phased roadmap for sanmon development.
---

## Design Principles

- **CUE as single source of truth**: Structure and policy are defined once in CUE. JSON Schema and other artifacts are derived.
- **Library first**: Build a Go library (`sanmon-core`) before wrapping it in a gRPC server.
- **Test first**: Golden test suite drives development. Every phase starts with test data.
- **Lean for meta-proofs**: Lean proves properties about the policy system, not individual rules.

---

## Phase 1: CUE Schema + Policy Unification & Golden Tests

**Goal**: Establish CUE as the single source of truth for both structure and policy. Build a golden test suite that drives all subsequent development.

### Tasks

- [x] Define base action schema in CUE (`policy/base/action.cue`)
- [x] Define domain-specific policies (browser, API, database, IaC)
- [x] Consolidate structural schema into CUE (eliminate separate TypeScript/Zod layer)
- [ ] Build golden test suite (per-domain valid/invalid JSON fixtures)
- [ ] `cue vet` passes for all valid cases, fails for all invalid cases
- [x] Remove `schema/` TypeScript directory (no longer needed)

### Deliverable
CUE as sole schema+policy definition with `testdata/` golden test suite.

---

## Phase 2: Go Validation Library (sanmon-core)

**Goal**: Build the core Go library that loads CUE policies and validates actions in-process.

### Tasks

- [ ] Create `middleware/pkg/sanmon/` package (Engine, loader, validator, result types)
- [ ] Policy composition logic (AND, domain routing, base + domain merge)
- [ ] JSON Schema export from loaded CUE
- [ ] Unit tests using golden test suite
- [ ] Policy hot-reload (watch filesystem, atomic swap)
- [ ] Benchmark: confirm < 10ms validation latency

### Deliverable
Go package `middleware/pkg/sanmon` — importable library for in-process validation.

---

## Phase 3: gRPC Server

**Goal**: Wrap sanmon-core in a gRPC server for cross-language / remote use.

### Tasks

- [x] Define protobuf service (`middleware/proto/guardrails.proto`)
- [ ] Implement gRPC server with Validate and ReloadPolicies RPCs
- [ ] Implement retry loop (re-prompt LLM on validation failure)
- [ ] Add HTTP/REST gateway (optional)
- [ ] End-to-end tests

### Deliverable
Running gRPC server (`middleware/cmd/server`) importing `sanmon-core`.

---

## Phase 4: JSON Schema Generation Pipeline

**Goal**: Automate JSON Schema generation from CUE for constrained decoding engine integration.

### Tasks

- [ ] `just schema` target: `cue export` → `generated/action-schema.json`
- [ ] Per-domain schema export
- [ ] Schema versioning
- [ ] Validate against OpenAPI 3.0 spec
- [ ] Integration test with Outlines / XGrammar

### Deliverable
`generated/` directory with per-domain JSON Schema files, automatically derived from CUE.

---

## Phase 5: Lean Meta-Proofs

**Goal**: Prove meta-properties of the policy system in Lean 4.

### Tasks

- [x] Define action model as Lean inductive types
- [x] Define safety properties
- [ ] Prove policy consistency
- [ ] Prove gate monotonicity
- [ ] Prove policy completeness
- [ ] Prove composition safety
- [ ] `lake build` succeeds for all proofs

### Deliverable
Lean proofs that typecheck for meta-properties of the policy system.

---

## Phase 6: CI/CD Pipeline

**Goal**: Automated verification on every change.

### Tasks

- [ ] GitHub Actions: CUE validation + golden tests on PR
- [ ] GitHub Actions: Go tests + lint + benchmark
- [ ] GitHub Actions: Lean proof check on PR
- [ ] GitHub Actions: JSON Schema generation + drift detection
- [ ] Nix-based CI (reproducible builds via `flake.nix`)
- [ ] README badges: "Tests Passing" + "Proofs Verified"

### Deliverable
`.github/workflows/` with complete CI pipeline.

---

## MVP (Milestone 0)

End-to-end demo for the browser domain:

1. CUE defines browser action schema + URL whitelist policy
2. JSON Schema derived from CUE → constrained decoding
3. Go library validates browser actions against CUE policy
4. gRPC server wraps the library for remote use
5. Golden tests prove policy correctness for known cases
6. Lean proves browser policy is consistent and complete

**Success criteria**:
- LLM cannot generate a `navigate` action to a non-whitelisted URL
- If it somehow does, CUE validator rejects it and triggers re-prompt
- Lean proves the browser policy set is consistent and has no gaps
- All verified by `just test` and `just lean-build`
