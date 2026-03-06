---
title: Domain Policies
description: Safety policies for browser, API, database, and IaC domains.
---

sanmon ships with policies for four agent domains. Each domain defines specific constraints that prevent dangerous actions.

## Browser (Playwright / Browser Use)

| Rule | Description |
|---|---|
| URL whitelist | Only allowed URL patterns (glob/regex) |
| Forbidden selectors | CSS selectors that must never be clicked/filled |
| Input length limit | Max characters for fill operations |
| Dangerous scheme block | No `javascript:`, `data:` URIs |
| Page transition graph | Allowed navigation sequences (future) |

### Example: valid browser action

```json
{
  "action_type": "navigate",
  "target": "https://example.com/page",
  "parameters": { "url": "https://example.com/page" },
  "context": { "domain": "browser", "authenticated": true, "session_id": "s1" },
  "metadata": { "timestamp": "2026-02-26T12:00:00Z", "agent_id": "a1", "request_id": "r1" }
}
```

### Example: blocked browser action

```json
{
  "action_type": "navigate",
  "target": "https://evil.com/phishing",
  "parameters": { "url": "https://evil.com/phishing" }
}
```

Violation: `URL 'https://evil.com/phishing' not in allowed patterns`

## API (MCP / Function Calling)

| Rule | Description |
|---|---|
| Endpoint whitelist | Only listed endpoints allowed |
| Method restrictions | Per-endpoint HTTP method limits |
| Auth requirement | Mutations require Authorization header |
| Body schema | Request body must match expected schema |
| Rate policy | Max calls per time window (future) |

## Database (SQL Agents)

| Rule | Description |
|---|---|
| Read-only tables | Listed tables cannot be modified |
| WHERE required | UPDATE/DELETE must have WHERE clause |
| DROP forbidden | DROP TABLE globally disabled by default |
| Sensitive columns | Access control for PII/secret columns |
| JOIN depth limit | Max nested JOINs (default 3) |

## IaC (Terraform / Pulumi)

| Rule | Description |
|---|---|
| Resource whitelist | Only listed resource types can be created/modified |
| Destroy forbidden | `destroy` action blocked by default |
| Open ingress block | Prevent `0.0.0.0/0` security group rules |
| Required tags | Every resource must have `owner`, `environment`, `project` |
| Plan always allowed | `plan` is a safe read-only operation |

## Adding a New Domain

To add a new domain:

1. Create `policy/domains/<name>/policy.cue` with domain-specific constraints
2. Add action type enums and validation rules
3. Add golden test cases in `testdata/valid/` and `testdata/invalid/`
4. Run `just test` to verify
5. Update the Lean model if formal proofs cover the new domain
