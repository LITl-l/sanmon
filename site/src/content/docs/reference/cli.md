---
title: CLI & API Reference
description: Reference for the sanmon CLI tool and HTTP server API.
---

## CLI: `sanmon`

The `sanmon` CLI validates actions, exports schemas, and inspects policies.

### `sanmon validate`

Validate a single action or a directory of actions.

```bash
# Validate a single file
sanmon validate --file testdata/valid/browser-navigate.json

# Validate all files in a directory
sanmon validate --dir testdata/valid/
```

**Output**: Pass/fail status with violation details for each action.

### `sanmon schema`

Export JSON Schema for a specific domain.

```bash
# Export browser domain schema
sanmon schema --domain browser

# Export all domain schemas
sanmon schema --domain api
sanmon schema --domain database
sanmon schema --domain iac
```

**Output**: JSON Schema to stdout (pipe to a file as needed).

### `sanmon policy`

Display the currently loaded policy configuration.

```bash
sanmon policy
```

**Output**: Summary of all loaded domain policies and their rules.

---

## HTTP Server: `sanmon-server`

The HTTP validation server exposes sanmon-core over HTTP.

### Start the server

```bash
sanmon-server --addr :8080 --policy policy/default-policy.json
```

Or via Just:

```bash
just serve
```

### `POST /validate`

Validate an action against loaded policies.

**Request**:

```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{
    "action_type": "navigate",
    "target": "https://example.com",
    "parameters": {"url": "https://example.com"},
    "context": {"domain": "browser", "authenticated": true, "session_id": "s1"},
    "metadata": {"timestamp": "2026-02-26T12:00:00Z", "agent_id": "a1", "request_id": "r1"}
  }'
```

**Response (pass)**:

```json
{
  "valid": true,
  "violations": []
}
```

**Response (fail)**:

```json
{
  "valid": false,
  "violations": [
    {
      "rule": "browser.url_whitelist",
      "message": "URL not in allowed patterns",
      "severity": "error"
    }
  ]
}
```

---

## Go Library: `sanmon-core`

Import the library for in-process validation (lowest latency).

```go
import "github.com/LITl-l/sanmon/middleware/pkg/sanmon"
```

### Engine interface

```go
type Engine interface {
    // Validate an action (JSON bytes) against loaded policies
    Validate(ctx context.Context, action []byte) (*Result, error)

    // Reload policies from disk
    ReloadPolicies(ctx context.Context) error

    // Export JSON Schema for a domain
    ExportJSONSchema(domain string) ([]byte, error)
}
```

### Usage

```go
engine, err := sanmon.NewEngine("policy/")
if err != nil {
    log.Fatal(err)
}

result, err := engine.Validate(ctx, actionJSON)
if err != nil {
    log.Fatal(err)
}

if !result.Valid {
    for _, v := range result.Violations {
        fmt.Printf("VIOLATION: %s — %s\n", v.Rule, v.Message)
    }
}
```

---

## gRPC API

```protobuf
service GuardrailsService {
  rpc Validate(ValidateRequest) returns (ValidateResponse);
  rpc ReloadPolicies(ReloadPoliciesRequest) returns (ReloadPoliciesResponse);
}
```

See `middleware/proto/guardrails.proto` for full message definitions.
