# sanmon Implementation Plan

## Phase 1: Core Data Model & JSON Schema

**Goal**: Define the unified action format and generate JSON Schema for constrained decoding.

### Tasks

- [x] Define Zod schemas for all action types (`schema/src/actions.ts`)
- [x] Create JSON Schema generator (`schema/src/generate.ts`)
- [ ] Run `make schema` and validate generated JSON against Outlines/XGrammar requirements
- [ ] Add domain-specific parameter schemas (browser params, API params, etc.)
- [ ] Write unit tests for schema validation (valid/invalid action examples)
- [ ] Verify generated schema works with at least one constrained decoding engine

### Deliverable
`schema/generated/action-schema.json` — a self-contained JSON Schema usable by constrained decoding engines.

---

## Phase 2: CUE Policy Engine

**Goal**: Build the CUE validation runtime that checks structured actions against policies.

### Tasks

- [x] Define base action schema in CUE (`policy/base/action.cue`)
- [x] Define domain-specific policies (browser, API, database, IaC)
- [ ] Implement Go validation engine (`middleware/internal/validator/`)
  - [ ] Load CUE policies from filesystem
  - [ ] Accept JSON action, evaluate against applicable policies
  - [ ] Return structured ValidationResult (pass/fail + violations)
  - [ ] Support policy hot-reload (no server restart)
- [ ] Implement policy composition
  - [ ] AND composition (all policies must pass)
  - [ ] Domain routing (select policy by `context.domain`)
  - [ ] Base + override merging
- [ ] Write integration tests
  - [ ] Valid actions pass each domain policy
  - [ ] Known violations are correctly detected
  - [ ] Violation messages are actionable
- [ ] Benchmark: confirm < 10ms validation latency

### Deliverable
Go package `middleware/internal/validator` that validates JSON against CUE policies programmatically.

---

## Phase 3: Lean Formal Proofs

**Goal**: Prove that the policy rule sets are consistent and safe.

### Tasks

- [x] Define action model as Lean inductive types (`prover/VerifiedGuardrails/Action.lean`)
- [x] Define safety properties (`prover/VerifiedGuardrails/Safety.lean`)
- [x] Scaffold policy formalization (`prover/VerifiedGuardrails/Policy.lean`)
- [ ] Define concrete state transition functions per domain
  - [ ] Browser: state = (current_url, page_history, form_state)
  - [ ] API: state = (auth_state, call_history, rate_count)
  - [ ] Database: state = (table_states, transaction_state)
  - [ ] IaC: state = (resource_inventory, pending_changes)
- [ ] Prove safety invariants per domain
  - [ ] Browser: "no navigation outside whitelist reachable"
  - [ ] Database: "no unfiltered DELETE/UPDATE reachable"
  - [ ] IaC: "no destroy action reachable"
- [ ] Prove policy consistency: no two rules contradict
- [ ] Prove unreachability: forbidden states have no valid action path

### Deliverable
Lean proofs that typecheck (`lake build` succeeds) for each domain's core safety properties.

---

## Phase 4: Middleware Integration

**Goal**: Ship a gRPC server that agents can call for validation, with retry loop.

### Tasks

- [x] Define protobuf service (`middleware/proto/guardrails.proto`)
- [ ] Generate Go gRPC code (`make proto`)
- [ ] Implement gRPC server (`middleware/cmd/server/`)
  - [ ] Wire `Validate` RPC to CUE validator
  - [ ] Wire `ReloadPolicies` RPC
  - [ ] Add request logging and metrics
  - [ ] Add health check endpoint
- [ ] Implement retry loop (`middleware/internal/retry/`)
  - [ ] Accept LLM provider config
  - [ ] On validation failure: construct re-prompt with violation reasons
  - [ ] Re-submit to LLM with constrained decoding
  - [ ] Configurable max retries (default 3)
  - [ ] Return final result or aggregated errors
- [ ] Implement Go client library (`middleware/pkg/client/`)
- [ ] Add HTTP/REST gateway (optional, for non-gRPC clients)
- [ ] Write end-to-end tests: mock LLM → validate → pass/fail → retry

### Deliverable
Running gRPC server (`middleware/cmd/server`) with `Validate` and `ReloadPolicies` RPCs.

---

## Phase 5: cue2lean Theorem Generator

**Goal**: Automatically translate CUE policy changes into Lean theorem statements.

### Tasks

- [ ] Parse CUE policy files programmatically (Go CUE API)
- [ ] Map CUE constraints to Lean propositions
  - [ ] Enum constraints → finite type membership
  - [ ] String patterns → predicate functions
  - [ ] Numeric bounds → inequality propositions
  - [ ] Whitelist/blacklist → set membership
- [ ] Generate Lean `.lean` files with theorem statements
- [ ] Integrate into CI: policy change → generate theorems → `lake build`
- [ ] Handle incremental updates (only regenerate for changed policies)

### Deliverable
`tools/cue2lean` CLI that reads `policy/**/*.cue` and outputs `prover/VerifiedGuardrails/Policy/*.lean`.

---

## Phase 6: Domain Policy Templates

**Goal**: Production-ready policy templates for each domain.

### Tasks

- [ ] Browser policy template with realistic defaults
  - [ ] Common SaaS URL patterns
  - [ ] Standard dangerous selector patterns
  - [ ] PII detection in input values
- [ ] API policy template
  - [ ] REST API convention patterns
  - [ ] OAuth/Bearer token enforcement
  - [ ] Common dangerous endpoints
- [ ] Database policy template
  - [ ] Common PII column names
  - [ ] Audit table protections
  - [ ] Transaction isolation rules
- [ ] IaC policy template
  - [ ] AWS/GCP/Azure resource type catalogs
  - [ ] Security group best practices
  - [ ] Cost-control constraints (instance size limits)
- [ ] Documentation for each template: what it protects and how to customize

### Deliverable
`policy/domains/*/` with documented, configurable policy templates.

---

## Phase 7: CI/CD Pipeline

**Goal**: Automated verification on every change.

### Tasks

- [ ] GitHub Actions workflow: Lean proof check on PR
- [ ] GitHub Actions workflow: CUE policy validation on PR
- [ ] GitHub Actions workflow: Go tests + lint
- [ ] GitHub Actions workflow: cue2lean generation + proof check
- [ ] Badge: "Proofs Passing" in README

### Deliverable
`.github/workflows/` with complete CI pipeline.

---

## MVP (Milestone 0)

End-to-end demo for the browser domain:

1. JSON Schema for browser actions → constrained decoding
2. CUE policy: URL whitelist + forbidden selector list
3. Go gRPC server validates browser actions
4. Lean proof: "only whitelisted URLs reachable"
5. Integration with Playwright MCP for live demo

**Success criteria**:
- LLM cannot generate a `navigate` action to a non-whitelisted URL (structural guarantee)
- If it somehow does, CUE validator rejects it and triggers re-prompt (semantic guarantee)
- Lean proves the URL whitelist policy is consistent and safe (formal guarantee)
