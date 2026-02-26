# sanmon (三門)

Three-gate formal verification stack for AI agent actions.

**Constrained Decoding + CUE + Lean** — not probabilistic guardrails, but mathematically proven safety guarantees.

## The Problem

AI agents (browser automation, API callers, DB operators, IaC tools) produce actions that can be dangerous. Current guardrails rely on probabilistic classifiers or pattern matching — they fail silently and offer no formal guarantees.

## The Approach

Instead of reasoning about LLM internals (impossible), sanmon constrains and verifies **LLM outputs**:

```
┌─────────────────────────────────────────────────┐
│  Gate 0: Constrained Decoding (構造的保証)        │
│  Force LLM output to valid JSON Schema.          │
│  100% structural correctness at generation time.  │
├─────────────────────────────────────────────────┤
│  Gate 1: CUE Validator (意味的検証)               │
│  Business rules, safety policies, whitelists.     │
│  Reject violations → re-prompt LLM.              │
├─────────────────────────────────────────────────┤
│  Gate 2: Lean Prover (健全性証明)                 │
│  Prove the rule set itself is consistent and      │
│  that no valid input can reach an unsafe state.   │
│  Runs offline in CI.                              │
└─────────────────────────────────────────────────┘
```

Constrained decoding bridges the probabilistic world (LLM) to the deterministic world (formal verification).

## Quick Start

```bash
# Prerequisites: Nix with flakes enabled
cd sanmon
direnv allow   # or: nix develop

# Verify toolchain
make policy-check   # Validate CUE policies
make schema         # Generate JSON Schema from TypeScript
make proto          # Generate gRPC Go code
make lean-build     # Build Lean proofs
```

## Project Structure

```
sanmon/
├── schema/            # Gate 0: Action types (TypeScript → JSON Schema)
│   └── src/
│       ├── actions.ts     # Zod type definitions
│       └── generate.ts    # JSON Schema generator
├── policy/            # Gate 1: CUE policy engine
│   ├── base/              # Base action schema
│   └── domains/           # Domain-specific policies
│       ├── browser/       # Playwright / browser automation
│       ├── api/           # MCP / function calling
│       ├── database/      # SQL / DB operations
│       └── iac/           # Infrastructure-as-code
├── prover/            # Gate 2: Lean 4 formal proofs
│   └── VerifiedGuardrails/
│       ├── Action.lean    # Action model (inductive types)
│       ├── Safety.lean    # Safety properties & theorems
│       └── Policy.lean    # CUE policy formalization
├── middleware/         # Runtime: Go gRPC validation server
│   ├── cmd/server/
│   ├── internal/
│   │   ├── validator/     # CUE runtime integration
│   │   └── retry/         # Re-prompt loop
│   ├── pkg/client/        # Go client library
│   └── proto/             # Protobuf definitions
├── tools/cue2lean/    # CUE → Lean theorem generator
├── ci/                # GitHub Actions (Lean proof CI)
└── docs/              # Specifications & architecture
```

## Tech Stack

| Component | Technology | Rationale |
|---|---|---|
| Runtime | Go | Native CUE support, low latency |
| Policy | CUE | Purpose-built for data constraints, JSON/YAML interop |
| Proofs | Lean 4 | Modern theorem prover, active ecosystem, AI affinity |
| Schema | TypeScript + Zod | JSON Schema generation, developer familiarity |
| API | gRPC + REST | Framework-agnostic integration |
| CI | GitHub Actions | Automated Lean proof checking |

## Domains

| Domain | Agent Type | Key Constraints |
|---|---|---|
| Browser | Playwright, Browser Use | URL whitelist, forbidden selectors, input limits |
| API | MCP, function calling | Endpoint whitelist, method restrictions, auth requirements |
| Database | SQL agents | Read-only tables, WHERE required, DROP forbidden |
| IaC | Terraform, Pulumi agents | Resource whitelist, destroy forbidden, tag requirements |

## License

TBD
