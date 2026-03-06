---
title: The Three Gates
description: Understanding sanmon's three-gate verification architecture.
---

The three gates operate at different times but share a single source of truth (CUE).

## Overview

| Gate | Timing | Purpose | Technology |
|---|---|---|---|
| **第一門 (Structure)** | Generation-time | Force LLM output to conform to JSON Schema | CUE → JSON Schema → Constrained Decoding |
| **第二門 (Policy)** | Runtime | Validate action against business rules and safety policies | CUE runtime validation (Go) |
| **第三門 (Proof)** | CI-time | Prove meta-properties of the policy system | Lean 4 |

## Gate 1: Constrained Decoding (第一門)

Force LLM outputs to conform to a JSON Schema at token generation time. This is not post-hoc validation — it modifies the sampling distribution to make invalid tokens impossible.

JSON Schema is **derived from CUE**, not maintained separately:

```
policy/**/*.cue  →  cue export --out openapi  →  JSON Schema
```

### Compatible engines

- AWS Bedrock Structured Outputs
- Outlines (open source, vLLM integration)
- XGrammar (open source, grammar-based)

### Guarantees

- 100% JSON Schema conformance
- Type correctness, required fields, enum value correctness
- No hallucinated field names or invalid structures

### Limitations

- Does **not** guarantee semantic correctness (valid structure but wrong values)
- Does **not** enforce business rules or safety properties

## Gate 2: CUE Validator (第二門)

Validates the semantic content of structurally valid actions against configurable policies.

### How it works

Every AI agent action is represented as a unified JSON format:

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

### Policy composition

- **AND**: All applicable policies must pass
- **Inheritance**: Base policy + domain-specific overrides
- **Modularity**: Adding a new domain policy must not affect existing ones

### Validation result

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

### Performance target

CUE validation latency: **< 10ms** per action.

## Gate 3: Lean Prover (第三門)

Proves **meta-properties** of the policy system, not individual rule correctness.

### What Lean proves

| Property | Description |
|---|---|
| **Policy consistency** | No two rules in a domain policy set contradict each other |
| **Gate monotonicity** | Any action passing Gate 2 also passes Gate 1 |
| **Policy completeness** | Every action type has at least one applicable policy defined |
| **Composition safety** | Merging base + domain policies preserves invariants |

### What Lean does NOT prove

- Individual rule correctness (e.g., "this URL is in the whitelist") — CUE handles this at runtime
- LLM behavior — outside the formal model entirely

### Formal model

```lean
-- Action types as inductive types
inductive ActionType where
  | browser (a : BrowserAction)
  | api     (a : ApiAction)
  | database (a : DatabaseAction)
  | iac     (a : IacAction)

-- Core theorem: safe actions preserve safe states
theorem safe_step_preserves_safety
    (s : State) (a : Action)
    (hs : SafeState s) (ha : SafeAction s a) :
    SafeState (step s a)
```

Lean proofs run only in CI (not at runtime) and are checked on every policy change as a PR gate.

## Runtime Path

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

## Retry Loop

When validation fails:
1. Collect violation reasons
2. Construct re-prompt: original instruction + structured violation feedback
3. Re-submit to LLM with constrained decoding
4. Validate again
5. Repeat up to N times (configurable, default 3)
6. If all retries fail, return error to caller
