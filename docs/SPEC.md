# sanmon Specification

## 1. Core Design Philosophy

- LLM internals (weights, attention) cannot be formalized.
- Instead, formalize the **output surface**: structure it, constrain it, prove properties about the constraints.
- Constrained decoding is the bridge between the probabilistic world and the deterministic world.

## 2. Architecture

### 2.1 Runtime Path (per request, latency-critical)

```
LLM Provider (Bedrock / OpenAI / self-hosted)
  │ constrained decoding (JSON Schema)
  ▼
Structured Action (JSON)
  │
  ▼
CUE Validation Engine (Go runtime, gRPC/HTTP)
  │
  ├── PASS → Execute action
  └── FAIL → Re-prompt LLM with violation reason (up to N retries)
```

### 2.2 Offline Path (CI/CD, correctness-critical)

```
CUE Policy files (*.cue)
  │
  ▼
cue2lean: CUE → Lean Theorem Generator
  │
  ▼
Lean 4 Proof Checker
  → "This policy set is safe" (proven)
```

## 3. Layer 0: Constrained Decoding

### Purpose
Force LLM outputs to conform to a JSON Schema at token generation time. This is not post-hoc validation — it modifies the sampling distribution to make invalid tokens impossible.

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

## 4. Layer 1: CUE Validator

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

## 5. Layer 2: Lean Prover

### Purpose
Prove that the CUE policy rule sets are:
1. **Consistent**: No two rules contradict each other
2. **Safe**: No sequence of policy-compliant actions can reach a forbidden state
3. **Complete** (optional): Every intended prohibition is actually enforced

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

-- Safety invariant
theorem safe_step_preserves_safety
    (s : State) (a : Action)
    (hs : SafeState s) (ha : SafeAction s a) :
    SafeState (step s a)
```

### cue2lean Tool
- Parses CUE policy files
- Generates corresponding Lean theorem statements
- Generated theorems are checked by Lean in CI
- If a policy change breaks a proof, the CI pipeline fails

### Execution
- Lean proofs run only in CI (not at runtime)
- Proof checking on every policy change (PR gate)
- Proof artifacts cached for unchanged policies

## 6. Middleware (Runtime Server)

### API

gRPC service with two RPCs:

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
  → gRPC client call to sanmon
    → sanmon validates
      → PASS: return to agent, agent executes
      → FAIL: agent re-prompts LLM (or sanmon handles retry)
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

sanmon is unique in combining: structured action output (constrained decoding) + rule validation (CUE) + rule soundness proofs (Lean).
