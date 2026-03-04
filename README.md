# sanmon (三門)

Three-gate formal verification stack for AI agent actions.

**CUE + Go + Lean** — not probabilistic guardrails, but mathematically proven safety guarantees.

## The Problem

AI agents (browser automation, API callers, DB operators, IaC tools) produce actions that can be dangerous. Current guardrails rely on probabilistic classifiers or pattern matching — they fail silently and offer no formal guarantees.

## The Approach

Instead of reasoning about LLM internals (impossible), sanmon constrains and verifies **LLM outputs** through three gates, all derived from a **single source of truth** (CUE):

```
┌─────────────────────────────────────────────────┐
│  第一門: Constrained Decoding (構造的保証)        │
│  JSON Schema derived from CUE.                   │
│  100% structural correctness at generation time.  │
├─────────────────────────────────────────────────┤
│  第二門: CUE Validator (意味的検証)               │
│  Business rules, safety policies, whitelists.     │
│  Reject violations → re-prompt LLM.              │
├─────────────────────────────────────────────────┤
│  第三門: Lean Prover (健全性証明)                 │
│  Prove the policy system is consistent,           │
│  complete, and compositionally safe.              │
│  Runs offline in CI.                              │
└─────────────────────────────────────────────────┘
```

CUE is the single source of truth: structure and policy defined once, JSON Schema derived automatically. Constrained decoding bridges the probabilistic world (LLM) to the deterministic world (formal verification).

## Quick Start

```bash
# Prerequisites: Nix with flakes enabled
cd sanmon
direnv allow   # or: nix develop

# Verify toolchain
just policy-check   # Validate CUE policies
just schema         # Generate JSON Schema from CUE
just proto          # Generate gRPC Go code
just lean-build     # Build Lean proofs
just test           # Run golden test suite
```

## Project Structure

```
sanmon/
├── policy/            # CUE: single source of truth (schema + policy)
│   ├── base/              # Base action schema (all domains)
│   └── domains/           # Domain-specific policies
│       ├── browser/       # Playwright / browser automation
│       ├── api/           # MCP / function calling
│       ├── database/      # SQL / DB operations
│       └── iac/           # Infrastructure-as-code
├── testdata/          # Golden test suite (valid/invalid per domain)
├── middleware/         # Go: sanmon-core library + gRPC server
│   ├── pkg/sanmon/        # Core validation library (in-process)
│   ├── cmd/server/        # Thin gRPC wrapper
│   ├── internal/retry/    # Re-prompt loop
│   └── proto/             # Protobuf definitions
├── prover/            # Lean 4: meta-proofs
│   └── VerifiedGuardrails/
│       ├── Action.lean    # Action model (inductive types)
│       ├── Safety.lean    # Safety properties & theorems
│       └── Policy.lean    # Meta-property proofs
├── generated/         # Derived artifacts (JSON Schema, gRPC code)
└── docs/              # Specifications & architecture
```

## Tech Stack

| Component | Technology | Rationale |
|---|---|---|
| Schema + Policy | CUE | Single source of truth for structure and constraints |
| Runtime | Go | Native CUE support, low latency, library-first design |
| Proofs | Lean 4 | Modern theorem prover, active ecosystem, AI affinity |
| API | gRPC (+ optional REST) | Framework-agnostic integration |
| CI | GitHub Actions + Nix | Reproducible, automated verification |

## The Three Gates

| Gate | Timing | What it does | Technology |
|---|---|---|---|
| **第一門** (Structure) | Generation-time | Force valid JSON structure via constrained decoding | CUE → JSON Schema |
| **第二門** (Policy) | Runtime | Validate business rules and safety policies | CUE runtime (Go) |
| **第三門** (Proof) | CI-time | Prove policy system meta-properties | Lean 4 |

## Domains

| Domain | Agent Type | Key Constraints |
|---|---|---|
| Browser | Playwright, Browser Use | URL whitelist, forbidden selectors, input limits |
| API | MCP, function calling | Endpoint whitelist, method restrictions, auth requirements |
| Database | SQL agents | Read-only tables, WHERE required, DROP forbidden |
| IaC | Terraform, Pulumi agents | Resource whitelist, destroy forbidden, tag requirements |

## License

TBD
