---
title: Specification
description: Formal specification of the sanmon three-gate verification stack.
---

## 1. Core Design Philosophy

- LLM internals (weights, attention) cannot be formalized.
- Instead, formalize the **output surface**: structure it, constrain it, prove properties about the constraints.
- Constrained decoding is the bridge between the probabilistic world and the deterministic world.
- **Single source of truth**: CUE defines both structure and policy. All other representations (JSON Schema, validation logic) are derived from CUE.

## 2. Architecture

### 2.1 The Three Gates (三門)

The three gates operate at different times but share a single source of truth (CUE):

| Gate | Timing | Purpose | Technology |
|---|---|---|---|
| **第一門 (Structure)** | Generation-time | Force LLM output to conform to JSON Schema | CUE → JSON Schema → Constrained Decoding |
| **第二門 (Policy)** | Runtime | Validate action against business rules and safety policies | CUE runtime validation (Go) |
| **第三門 (Proof)** | CI-time | Prove meta-properties of the policy system | Lean 4 |

### 2.2 Runtime Path (per request, latency-critical)

```
LLM Provider (Bedrock / OpenAI / self-hosted)
  │ constrained decoding (JSON Schema derived from CUE)
  ▼
Structured Action (JSON)
  │
  ▼
sanmon-core (Go library, in-process)
  │ CUE runtime validation
  │
  ├── PASS → Execute action
  └── FAIL → Re-prompt LLM with violation reason (up to N retries)
```

### 2.3 Offline Path (CI/CD, correctness-critical)

```
CUE Policy files (*.cue)
  │
  ├─► cue export --out jsonschema → JSON Schema (第一門 artifact)
  │
  └─► Lean 4 meta-proofs
        → "Policy composition is consistent" (proven)
        → "Gate monotonicity holds" (proven)
        → "All action types have policies defined" (proven)
```

## 3. Gate 1: Constrained Decoding (第一門)

### Purpose
Force LLM outputs to conform to a JSON Schema at token generation time. This is not post-hoc validation — it modifies the sampling distribution to make invalid tokens impossible.

### Source
JSON Schema is **derived from CUE**, not maintained separately. CUE is the single source of truth for both structure and semantics.

```
policy/**/*.cue  →  cue export --out openapi  →  JSON Schema
```

### Technologies
- AWS Bedrock Structured Outputs
- Outlines (open source, vLLM integration)
- XGrammar (open source, grammar-based)

### Guarantees
- 100% JSON Schema conformance
- Type correctness, required fields, enum value correctness
- No hallucinated field names or invalid structures

### What it does NOT guarantee
- Semantic correctness (valid structure but wrong values)
- Business rule compliance
- Safety properties

## 4. Gate 2: CUE Validator (第二門)

### Purpose
Validate the semantic content of structurally valid actions against configurable policies.

### Schema: Unified Action Format

Every AI agent action is represented as:

```json
{
  "action_type": "<domain-specific enum>",
  "target": "<URL | selector | table | resource>",
  "parameters": { ... },
  "context": {
    "authenticated": true,
    "session_id": "...",
    "domain": "browser"
  },
  "metadata": {
    "timestamp": "2026-02-26T12:00:00Z",
    "agent_id": "...",
    "request_id": "..."
  }
}
```

### CUE as Single Source of Truth

CUE defines both structural schema and semantic policies in one place:

```cue
// Structure (feeds into JSON Schema generation)
#Action: {
    action_type: #BrowserActionType | #ApiActionType | ...
    target:      string
    parameters:  {...}
    context:     #Context
    metadata:    #Metadata
}

// Policy (enforced at runtime)
#BrowserPolicy: {
    url_whitelist: [...string]
    forbidden_selectors: [...string]
    max_input_length: int | *1000
}
```

### Policy Structure

```
policy/
├── base/action.cue          # Base schema (all domains)
└── domains/
    ├── browser/policy.cue    # URL whitelist, forbidden selectors, input limits
    ├── api/policy.cue        # Endpoint whitelist, method restrictions
    ├── database/policy.cue   # Read-only tables, WHERE required, DROP forbidden
    └── iac/policy.cue        # Resource whitelist, destroy forbidden, tags
```

### Policy Composition
- **AND**: All applicable policies must pass
- **Inheritance**: Base policy + domain-specific overrides
- **Modularity**: Adding a new domain policy must not affect existing ones

### Validation Result

```json
{
  "pass": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URL 'https://evil.com' not in allowed patterns",
      "path": "parameters.url",
      "severity": "error"
    }
  ]
}
```

### Performance Target
- CUE validation latency: < 10ms per action

## 5. Gate 3: Lean Prover (第三門)

### Purpose
Prove **meta-properties** of the policy system, not individual rule correctness.

### What Lean proves

| Property | Description |
|---|---|
| **Policy consistency** | No two rules in a domain policy set contradict each other |
| **Gate monotonicity** | Any action passing Gate 2 (CUE) also passes Gate 1 (JSON Schema) |
| **Policy completeness** | Every action type has at least one applicable policy defined |
| **Composition safety** | Merging base + domain policies preserves invariants |

### What Lean does NOT prove
- Individual rule correctness (e.g., "this URL is in the whitelist") — CUE handles this at runtime
- LLM behavior — outside the formal model entirely

### Formal Model

```lean
-- Action types as inductive types
inductive ActionType where
  | browser (a : BrowserAction)
  | api     (a : ApiAction)
  | database (a : DatabaseAction)
  | iac     (a : IacAction)

-- State transition
def step (s : State) (a : Action) : State

-- Core theorem: safe actions preserve safe states
theorem safe_step_preserves_safety
    (s : State) (a : Action)
    (hs : SafeState s) (ha : SafeAction s a) :
    SafeState (step s a)
```

### Execution
- Lean proofs run only in CI (not at runtime)
- Proof checking on every policy change (PR gate)
- Proof artifacts cached for unchanged policies

## 6. Runtime: sanmon-core (Go Library)

### Architecture

```
sanmon-core (Go library)              ← in-process, <10ms
  ├── CUE loader + policy compositor
  ├── Validator (CUE runtime evaluation)
  ├── JSON Schema exporter
  └── Structured violation reporter

sanmon-server (thin gRPC wrapper)     ← for cross-language / remote use
  └── imports sanmon-core

sanmon-sdk (future)                   ← language-specific clients
  └── gRPC client wrappers
```

### Library API (Go)

```go
// Core validation
type Engine interface {
    Validate(ctx context.Context, action []byte) (*Result, error)
    ReloadPolicies(ctx context.Context) error
    ExportJSONSchema(domain string) ([]byte, error)
}
```

### gRPC API

```protobuf
service GuardrailsService {
  rpc Validate(ValidateRequest) returns (ValidateResponse);
  rpc ReloadPolicies(ReloadPoliciesRequest) returns (ReloadPoliciesResponse);
}
```

### Retry Loop

When validation fails:
1. Collect violation reasons
2. Construct re-prompt: original instruction + structured violation feedback
3. Re-submit to LLM with constrained decoding
4. Validate again
5. Repeat up to N times (configurable, default 3)
6. If all retries fail, return error to caller

### Integration Pattern

```
Agent Framework (any)
  → sanmon-core.Validate(action)       # in-process (preferred)
  → or gRPC client call to sanmon-server  # remote
    → PASS: agent executes action
    → FAIL: agent re-prompts LLM
```

## 7. Domain-Specific Policies

### 7.1 Browser (Playwright / Browser Use)

| Rule | Description |
|---|---|
| URL whitelist | Only allowed URL patterns (glob/regex) |
| Forbidden selectors | CSS selectors that must never be clicked/filled |
| Input length limit | Max characters for fill operations |
| Dangerous scheme block | No `javascript:`, `data:` URIs |
| Page transition graph | Allowed navigation sequences (future) |

### 7.2 API (MCP / Function Calling)

| Rule | Description |
|---|---|
| Endpoint whitelist | Only listed endpoints allowed |
| Method restrictions | Per-endpoint HTTP method limits |
| Auth requirement | Mutations require Authorization header |
| Body schema | Request body must match expected schema |
| Rate policy | Max calls per time window (future) |

### 7.3 Database

| Rule | Description |
|---|---|
| Read-only tables | Listed tables cannot be modified |
| WHERE required | UPDATE/DELETE must have WHERE clause |
| DROP forbidden | DROP TABLE globally disabled by default |
| Sensitive columns | Access control for PII/secret columns |
| JOIN depth limit | Max nested JOINs (default 3) |

### 7.4 IaC (Terraform / Pulumi)

| Rule | Description |
|---|---|
| Resource whitelist | Only listed resource types can be created/modified |
| Destroy forbidden | destroy action blocked by default |
| Open ingress block | Prevent 0.0.0.0/0 security group rules |
| Required tags | Every resource must have owner, environment, project |
| Plan always allowed | plan is a safe read-only operation |

## 8. Differentiation

| Project | Approach | Difference from sanmon |
|---|---|---|
| Invariant Labs (Snyk) | Python DSL rule-based | No formal proofs, runtime checks only |
| AWS Bedrock Automated Reasoning | Formal logic for factual content | Content verification, not action constraints |
| Guardrails AI | Validator collection | Probabilistic detection, no mathematical guarantees |
| AWS Cedar + Lean | Authorization policy verification | Static policy, not AI agent runtime constraints |
| Smart contract verification | Coq/Lean/Isabelle on contract code | Verifies code itself, not AI-generated actions |

sanmon is unique in combining: single-source CUE definitions + constrained decoding + runtime validation + meta-level formal proofs (Lean).
