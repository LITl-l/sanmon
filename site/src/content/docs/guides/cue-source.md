---
title: CUE as Source of Truth
description: How CUE unifies schema and policy definitions.
---

CUE is the single source of truth for both structure and policy in sanmon. All other representations — JSON Schema, validation logic, documentation — are derived from CUE definitions.

## Why CUE?

CUE was designed for configuration and validation. It combines types, values, and constraints in a single language:

- **Types and values unify** — a type *is* a constraint, a value *is* a type
- **Closed by default** — unknown fields are rejected
- **Hermetic** — no imports from outside the CUE universe, no side effects
- **Native Go support** — first-class Go library for runtime evaluation

## Schema Definition

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

## Policy Structure

```
policy/
├── base/action.cue          # Base schema (all domains)
└── domains/
    ├── browser/policy.cue    # URL whitelist, forbidden selectors, input limits
    ├── api/policy.cue        # Endpoint whitelist, method restrictions
    ├── database/policy.cue   # Read-only tables, WHERE required, DROP forbidden
    └── iac/policy.cue        # Resource whitelist, destroy forbidden, tags
```

## Derived Artifacts

From a single set of CUE files, sanmon derives:

| Artifact | Command | Purpose |
|---|---|---|
| JSON Schema | `just schema` | Constrained decoding (Gate 1) |
| Go validation | sanmon-core library | Runtime validation (Gate 2) |
| Lean model | Manual translation (future: automated) | Meta-proofs (Gate 3) |

## Advantages

1. **No drift** — structure and policy cannot diverge because they are defined together
2. **Composability** — CUE's lattice-based type system supports safe policy merging
3. **Tooling** — `cue vet`, `cue export`, `cue fmt` provide built-in validation and formatting
4. **Performance** — CUE evaluation is fast enough for runtime use (< 10ms target)
