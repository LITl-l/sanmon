# sanmon Implementation Plan

## Design Principles

- **CUE as single source of truth**: Structure and policy are defined once in CUE. JSON Schema and other artifacts are derived.
- **Library first**: Build a Go library (`sanmon-core`) before wrapping it in a gRPC server.
- **Test first**: Golden test suite drives development. Every phase starts with test data.
- **Lean for meta-proofs**: Lean proves properties about the policy system, not individual rules.

---

## Status (2026-06)

The validation engine is shipped and demo-ready; some original architecture choices changed in implementation. Checkboxes below reflect this. Notable divergences from the original plan:

- **Schema generation is done by the Go CLI (`sanmon schema`), not `cue export`.** The CUE files under `policy/` mirror the Go types and are validated by `cue vet`. Drift is now guarded: a CI step regenerates and diffs `schema/generated/`, and `TestStarterAgentPolicyMatchesCUE` decodes the agent CUE policy and diffs it against `StarterAgentPolicy()` so Go↔CUE divergence fails CI. Generating one from the other *automatically* (CUE → Go) remains the headline follow-up.
- **The server is `net/http` JSON, not gRPC.** `proto/guardrails.proto` exists but is unused; gRPC is deferred.
- **Domains shipped:** browser, api, database, iac, approval, **agent** (the universal pre-execution guard — `sanmon guard` / `sanmon init`), beyond the original four.
- **Shell analysis uses a real parser** (`mvdan.cc/sh`): commands are extracted across pipelines/lists/subshells and literalized, defeating quote-insertion obfuscation; structural detectors catch recursive-force deletes and decode-and-execute chains. Runtime value expansion (`$VAR`, `$(...)`) is still not simulated.
- **No filesystem hot-reload or latency benchmark yet**; `Engine.ReloadPolicy` does atomic in-memory swap.
- **CI:** Go build/vet/test + CUE vet + schema-drift guard run in `.github/workflows/ci.yml`. Lean proof CI is still pending.

---

## Phase 1: CUE Schema + Policy Unification & Golden Tests

**Goal**: Establish CUE as the single source of truth for both structure and policy. Build a golden test suite that drives all subsequent development.

### Tasks

- [x] Define base action schema in CUE (`policy/base/action.cue`)
- [x] Define domain-specific policies (browser, API, database, IaC)
- [x] Consolidate structural schema into CUE (eliminate separate TypeScript/Zod layer)
  - [x] Move action type definitions from `schema/src/actions.ts` into CUE
  - [x] Verify CUE schema can express all current Zod constraints
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
- [x] Remove `schema/` TypeScript directory (no longer needed)

### Deliverable
- CUE as sole schema+policy definition
- `testdata/` golden test suite
- `just schema` generates JSON Schema from CUE

---

## Phase 2: Go Validation Library (sanmon-core)

**Goal**: Build the core Go library that loads CUE policies and validates actions in-process.

### Tasks

- [x] Create `middleware/pkg/sanmon/` package
  - [x] `engine.go`: `Engine` (Validate, ValidateJSON, ReloadPolicy)
  - [x] `result.go`: Structured ValidationResult (pass/fail + violations)
  - [x] Per-domain validators (`validate_<domain>.go`) — chosen over a single CUE-runtime evaluator
- [x] Policy composition logic
  - [x] Domain routing (`validatePolicy` selects policy by `context.domain`)
  - [x] Base + domain-specific validation
- [ ] JSON Schema export — implemented via Go (`sanmon schema`), not from loaded CUE (`ExportJSONSchema`); CUE-generative export is the follow-up
- [x] Write unit tests using golden test suite
  - [x] All valid golden cases pass validation
  - [x] All invalid golden cases fail with expected violations
  - [x] Violation messages are actionable
- [ ] Policy hot-reload — `ReloadPolicy` does atomic in-memory swap; filesystem watch not yet implemented
- [ ] Benchmark: confirm < 10ms validation latency
- [x] `just test` runs full golden suite

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
- [x] HTTP/REST server (`cmd/server`, net/http) — shipped as the primary transport instead of gRPC
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

- [x] GitHub Actions workflow: Go vet + build + test on PR (`.github/workflows/ci.yml`)
- [x] GitHub Actions workflow: CUE validation (`cue vet`) on PR
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
