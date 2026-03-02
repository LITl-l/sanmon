.PHONY: all build test clean schema policy-check demo \
       lean-build proto serve

all: build

# ── Build ──
build: build-cli build-server

build-cli:
	cd middleware && go build -o ../bin/sanmon ./cmd/sanmon/

build-server:
	cd middleware && go build -o ../bin/sanmon-server ./cmd/server/

# ── Test ──
test:
	cd middleware && go test ./... -v -count=1

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
	@echo "JSON Schemas exported to schema/generated/"

# ── Validate CUE policies (requires cue CLI) ──
policy-check:
	cue vet ./policy/base/
	cue vet ./policy/domains/browser/
	cue vet ./policy/domains/api/
	cue vet ./policy/domains/database/
	cue vet ./policy/domains/iac/

# ── Proto: Generate gRPC Go code (requires buf) ──
proto:
	buf generate

# ── Lean: Build formal proofs (requires lean4) ──
lean-build:
	cd prover && lake build

# ── Clean ──
clean:
	rm -rf bin/
	rm -rf schema/dist schema/generated schema/node_modules
	rm -rf middleware/proto/guardrailsv1
	cd prover && lake clean 2>/dev/null || true
