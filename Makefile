.PHONY: all schema proto policy-check lean-build clean

all: schema proto

# ── Phase 1: Generate JSON Schema from TypeScript ──
schema:
	cd schema && npm install && npx tsc && node dist/generate.js

# ── Phase 2: Validate CUE policies ──
policy-check:
	cue vet ./policy/base/
	cue vet ./policy/domains/browser/
	cue vet ./policy/domains/api/
	cue vet ./policy/domains/database/
	cue vet ./policy/domains/iac/

# ── Phase 4: Generate gRPC Go code ──
proto:
	buf generate

# ── Phase 3: Build Lean proofs ──
lean-build:
	cd prover && lake build

# ── Middleware ──
middleware-build:
	cd middleware && go build ./cmd/server

middleware-test:
	cd middleware && go test ./...

middleware-lint:
	cd middleware && golangci-lint run

# ── Clean ──
clean:
	rm -rf schema/dist schema/generated schema/node_modules
	rm -rf middleware/proto/guardrailsv1
	cd prover && lake clean 2>/dev/null || true
