---
title: Quick Start
description: Get sanmon up and running in minutes.
---

## Prerequisites

- [Nix](https://nixos.org/download/) with flakes enabled
- Or manually install: Go 1.22+, CUE CLI, Lean 4 (elan), Just

## Setup

```bash
git clone https://github.com/LITl-l/sanmon.git
cd sanmon
direnv allow   # or: nix develop
```

## Verify the Toolchain

```bash
# Validate CUE policies
just policy-check

# Generate JSON Schema from CUE
just schema

# Generate gRPC Go code
just proto

# Build Lean proofs
just lean-build

# Run golden test suite
just test
```

## Run the Demo

```bash
just demo
```

This runs the full three-gate verification demo:

1. Validates all **valid** test actions (expect all to pass)
2. Validates all **invalid** test actions (expect all to fail with violation details)
3. Exports the JSON Schema for the browser domain
4. Shows the current loaded policy

## Start the HTTP Server

```bash
just serve
```

Starts a validation server on `:8080`. Send actions to validate:

```bash
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d @testdata/valid/browser-navigate.json
```

## Project Structure

```
sanmon/
├── policy/            # CUE: single source of truth (schema + policy)
│   ├── base/              # Base action schema (all domains)
│   └── domains/           # Domain-specific policies
├── testdata/          # Golden test suite (valid/invalid per domain)
├── middleware/         # Go: sanmon-core library + gRPC server
│   ├── pkg/sanmon/        # Core validation library (in-process)
│   ├── cmd/sanmon/        # CLI tool
│   └── cmd/server/        # HTTP validation server
├── prover/            # Lean 4: meta-proofs
├── schema/generated/  # Derived JSON Schema (from Go CLI)
├── site/              # Documentation site (Astro Starlight)
└── docs/              # Specifications & architecture
```

## Build Commands

| Command | Description |
|---|---|
| `just build` | Build CLI and HTTP server |
| `just test` | Run golden test suite |
| `just demo` | End-to-end verification demo |
| `just serve` | Start HTTP validation server on :8080 |
| `just policy-check` | Validate CUE policies |
| `just schema` | Export JSON Schemas from CUE |
| `just proto` | Generate gRPC Go code |
| `just lean-build` | Build Lean 4 proofs |
| `just clean` | Remove build artifacts |
