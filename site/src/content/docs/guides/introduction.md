---
title: Introduction
description: What is sanmon and why does it exist?
---

**sanmon** (三門) is a three-gate formal verification stack for AI agent actions. It provides mathematically proven safety guarantees — not probabilistic guardrails — for LLM-generated actions in dangerous domains.

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

## Tech Stack

| Component | Technology | Rationale |
|---|---|---|
| Schema + Policy | CUE | Single source of truth for structure and constraints |
| Runtime | Go | Native CUE support, low latency, library-first design |
| Proofs | Lean 4 | Modern theorem prover, active ecosystem, AI affinity |
| API | gRPC (+ optional REST) | Framework-agnostic integration |
| CI | GitHub Actions + Nix | Reproducible, automated verification |

## Differentiation

| Project | Approach | Difference from sanmon |
|---|---|---|
| Invariant Labs (Snyk) | Python DSL rule-based | No formal proofs, runtime checks only |
| AWS Bedrock Automated Reasoning | Formal logic for factual content | Content verification, not action constraints |
| Guardrails AI | Validator collection | Probabilistic detection, no mathematical guarantees |
| AWS Cedar + Lean | Authorization policy verification | Static policy, not AI agent runtime constraints |
| Smart contract verification | Coq/Lean/Isabelle on contract code | Verifies code itself, not AI-generated actions |

sanmon is unique in combining: single-source CUE definitions + constrained decoding + runtime validation + meta-level formal proofs (Lean).
