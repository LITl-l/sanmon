# sanmon Implementation Plan

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
- [ ] Consolidate structural schema into CUE (eliminate separate TypeScript/Zod layer)
  - [ ] Move action type definitions from `schema/src/actions.ts` into CUE
  - [ ] Verify CUE schema can express all current Zod constraints
  - [ ] Generate JSON Schema from CUE (`cue export --out openapi`)
  - [ ] Validate generated JSON Schema works with constrained decoding engines
- [ ] Build golden test suite
  - [ ] `testdata/browser/valid/*.json` — valid browser actions
  - [ ] `testdata/browser/invalid/*.json` — browser policy violations
  - [ ] `testdata/api/valid/*.json` / `testdata/api/invalid/*.json`
  - [ ] `testdata/database/valid/*.json` / `testdata/database/invalid/*.json`
  - [ ] `testdata/iac/valid/*.json` / `testdata/iac/invalid/*.json`
  - [ ] Each invalid case annotated with expected violation rule + message
- [ ] `cue vet` passes for all valid cases, fails for all invalid cases
- [ ] Remove `schema/` TypeScript directory (no longer needed)

### Deliverable
- CUE as sole schema+policy definition
- `testdata/` golden test suite
- `just schema` generates JSON Schema from CUE

---

## Phase 2: Go Validation Library (sanmon-core)

**Goal**: Build the core Go library that loads CUE policies and validates actions in-process.

### Tasks

- [ ] Create `middleware/pkg/sanmon/` package
  - [ ] `engine.go`: `Engine` interface (Validate, ReloadPolicies, ExportJSONSchema)
  - [ ] `loader.go`: CUE policy loader (base + domain composition)
  - [ ] `validator.go`: CUE runtime evaluation against loaded policies
  - [ ] `result.go`: Structured ValidationResult (pass/fail + violations)
- [ ] Policy composition logic
  - [ ] AND composition (all policies must pass)
  - [ ] Domain routing (select policy by `context.domain`)
  - [ ] Base + domain-specific merge
- [ ] JSON Schema export from loaded CUE (`ExportJSONSchema`)
- [ ] Write unit tests using golden test suite from Phase 1
  - [ ] All valid golden cases pass validation
  - [ ] All invalid golden cases fail with expected violations
  - [ ] Violation messages are actionable
- [ ] Policy hot-reload (watch filesystem, atomic swap)
- [ ] Benchmark: confirm < 10ms validation latency
- [ ] `just test` runs full golden suite

### Deliverable
Go package `middleware/pkg/sanmon` — importable library for in-process validation.

---

## Phase 3: gRPC Server

**Goal**: Wrap sanmon-core in a gRPC server for cross-language / remote use.

### Tasks

- [x] Define protobuf service (`middleware/proto/guardrails.proto`)
- [ ] Generate Go gRPC code (`just proto`)
- [ ] Implement gRPC server (`middleware/cmd/server/`)
  - [ ] Wire `Validate` RPC to sanmon-core Engine
  - [ ] Wire `ReloadPolicies` RPC
  - [ ] Add request logging and metrics (latency, pass/fail counts)
  - [ ] Add health check endpoint
- [ ] Implement retry loop (`middleware/internal/retry/`)
  - [ ] On validation failure: construct re-prompt with violation reasons
  - [ ] Re-submit to LLM with constrained decoding
  - [ ] Configurable max retries (default 3)
  - [ ] Return final result or aggregated errors
- [ ] Add HTTP/REST gateway (optional, for non-gRPC clients)
- [ ] Write end-to-end tests: mock LLM → validate → pass/fail → retry

### Deliverable
Running gRPC server (`middleware/cmd/server`) importing `sanmon-core`.

---

## Phase 4: JSON Schema Generation Pipeline

**Goal**: Automate JSON Schema generation from CUE for constrained decoding engine integration.

### Tasks

- [ ] `just schema` target: `cue export` → `generated/action-schema.json`
- [ ] Per-domain schema export (browser-only, api-only, etc.)
- [ ] Schema versioning (embed git hash or semver)
- [ ] Validate generated schema against OpenAPI 3.0 spec
- [ ] Integration test: feed generated schema to Outlines / XGrammar
- [ ] Document schema usage for each constrained decoding engine

### Deliverable
`generated/` directory with per-domain JSON Schema files, automatically derived from CUE.

---

## Phase 5: Lean Meta-Proofs

**Goal**: Prove meta-properties of the policy system in Lean 4.

### Tasks

- [x] Define action model as Lean inductive types (`prover/VerifiedGuardrails/Action.lean`)
- [x] Define safety properties (`prover/VerifiedGuardrails/Safety.lean`)
- [ ] Prove policy consistency
  - [ ] No two rules in a domain contradict each other
  - [ ] Base + domain merge preserves all base invariants
- [ ] Prove gate monotonicity
  - [ ] Actions passing CUE validation (Gate 2) conform to JSON Schema (Gate 1)
  - [ ] Structural validity is a subset of semantic validity
- [ ] Prove policy completeness
  - [ ] Every action type has at least one applicable policy
  - [ ] No action type falls through without evaluation
- [ ] Prove composition safety
  - [ ] Adding a new domain policy cannot break existing domain policies
  - [ ] Policy merge is associative and commutative where applicable
- [ ] `lake build` succeeds for all proofs

### Deliverable
Lean proofs that typecheck for meta-properties of the policy system.

---

## Phase 6: CI/CD Pipeline

**Goal**: Automated verification on every change.

### Tasks

- [ ] GitHub Actions workflow: CUE validation + golden tests on PR
- [ ] GitHub Actions workflow: Go tests + lint + benchmark
- [ ] GitHub Actions workflow: Lean proof check on PR
- [ ] GitHub Actions workflow: JSON Schema generation + drift detection
- [ ] Nix-based CI (reproducible builds via `flake.nix`)
- [ ] Badge: "Tests Passing" + "Proofs Verified" in README

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
- LLM cannot generate a `navigate` action to a non-whitelisted URL (structural guarantee via derived JSON Schema)
- If it somehow does, CUE validator rejects it and triggers re-prompt (semantic guarantee)
- Lean proves the browser policy set is consistent and has no gaps (meta-level guarantee)
- All of the above verified by `just test` and `just lean-build`
