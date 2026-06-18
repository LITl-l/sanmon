# sanmon (三門) — Three-gate formal verification stack

set dotenv-load := false

default: build

# ── Build ──

build: build-cli build-server

# sanmon is pure Go — pin CGO_ENABLED=0 for static, reproducible binaries
# that build without a C toolchain.
build-cli:
    cd middleware && CGO_ENABLED=0 go build -o ../bin/sanmon ./cmd/sanmon/

build-server:
    cd middleware && CGO_ENABLED=0 go build -o ../bin/sanmon-server ./cmd/server/

# ── Test ──

test:
    cd middleware && CGO_ENABLED=0 go test ./... -count=1

# ── CI gate: vet + build + test (mirrors .github/workflows/ci.yml) ──

check:
    cd middleware && CGO_ENABLED=0 go vet ./...
    cd middleware && CGO_ENABLED=0 go build ./...
    cd middleware && CGO_ENABLED=0 go test ./... -count=1

# ── Demo: run all validations against golden test data ──

demo: build-cli
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "  三門 (sanmon) — Formal Verification Demo"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo ""
    @echo "Gate 1+2: Validating VALID actions..."
    @./bin/sanmon validate --dir testdata/valid/
    @echo ""
    @echo "Gate 1+2: Validating INVALID actions (expect failures)..."
    @./bin/sanmon validate --dir testdata/invalid/
    @echo ""
    @echo "JSON Schema export (browser domain):"
    @./bin/sanmon schema --domain browser
    @echo ""
    @echo "Current policy:"
    @./bin/sanmon policy

# ── Serve: start HTTP validation server ──

serve: build-server
    ./bin/sanmon-server --addr :8080 --policy policy/default-policy.json

# ── Schema: export JSON Schemas ──

schema: build-cli
    @mkdir -p schema/generated
    @./bin/sanmon schema --domain browser  > schema/generated/browser-action.json
    @./bin/sanmon schema --domain api      > schema/generated/api-action.json
    @./bin/sanmon schema --domain database > schema/generated/database-action.json
    @./bin/sanmon schema --domain iac      > schema/generated/iac-action.json
    @./bin/sanmon schema --domain approval > schema/generated/approval-action.json
    @./bin/sanmon schema --domain agent     > schema/generated/agent-action.json
    @echo "JSON Schemas exported to schema/generated/"

# Fail if committed JSON Schemas are stale vs the Go generator (CI drift guard).
# VCS-agnostic: regenerates to a temp dir and diffs against the committed files.
schema-check: build-cli
    @tmp=$(mktemp -d); \
     for d in browser api database iac approval agent; do \
       ./bin/sanmon schema --domain "$d" > "$tmp/$d-action.json"; \
       diff -u "schema/generated/$d-action.json" "$tmp/$d-action.json" \
         || { echo "ERROR: schema/generated/$d-action.json is stale — run 'just schema' and commit." >&2; rm -rf "$tmp"; exit 1; }; \
     done; \
     rm -rf "$tmp"; echo "Generated schemas are in sync."

# ── Demo: backoffice approval mock app ──

demo-backoffice:
    @echo "バックオフィス承認デモアプリを起動中..."
    @echo "http://localhost:3000 でアクセスしてください"
    python3 -m http.server 3000 -d demo/backoffice/

# ── Validate CUE policies (requires cue CLI) ──

policy-check:
    cue vet ./policy/base/
    cue vet ./policy/domains/browser/
    cue vet ./policy/domains/api/
    cue vet ./policy/domains/database/
    cue vet ./policy/domains/iac/
    cue vet ./policy/domains/approval/
    cue vet ./policy/domains/agent/

# ── Proto: Generate gRPC Go code (requires buf) ──

proto:
    buf generate

# ── Lean: Build formal proofs (requires lean4) ──

lean-build:
    cd prover && lake build

# ── Docs: Starlight documentation site ──

docs-dev:
    cd site && bun run dev

docs-build:
    cd site && bun run build

docs-preview:
    cd site && bun run preview

docs-install:
    cd site && bun install

# ── Clean ──

clean:
    rm -rf bin/
    rm -rf schema/generated
    rm -rf middleware/proto/guardrailsv1
    cd prover && lake clean 2>/dev/null || true
