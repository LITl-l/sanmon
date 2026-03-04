{
  description = "sanmon (三門) – Three-gate formal verification stack for AI agent actions";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # ── Go (runtime, CUE integration, gRPC server) ──
            go
            gopls
            golangci-lint
            delve           # Go debugger

            # ── CUE (policy definition & validation) ──
            cue

            # ── Lean 4 (formal proofs) via elan toolchain manager ──
            elan

            # ── Node.js / TypeScript (JSON Schema, action type definitions) ──
            # Node 24+ has built-in TypeScript support (--experimental-strip-types)
            nodejs
            nodePackages.typescript
            nodePackages.typescript-language-server

            # ── Protobuf / gRPC tooling ──
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            buf
            grpcurl

            # ── Build & utility ──
            just
            jq
            yq-go
            git
            treefmt
          ];

          shellHook = ''
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"

            # Lean / elan
            export ELAN_HOME="$PWD/.elan"
            export PATH="$ELAN_HOME/bin:$PATH"

            echo "──────────────────────────────────────────"
            echo " sanmon (三門) Dev Environment"
            echo "──────────────────────────────────────────"
            echo " Go:         $(go version | cut -d' ' -f3)"
            echo " CUE:        $(cue version | head -1 | awk '{print $2}')"
            echo " Node:       $(node --version)"
            echo " TypeScript: $(tsc --version | awk '{print $2}')"
            echo " Protoc:     $(protoc --version | awk '{print $2}')"
            echo " Buf:        $(buf --version 2>&1)"
            echo " Lean/elan:  $(elan --version 2>/dev/null || echo 'run: elan toolchain install leanprover/lean4:stable')"
            echo "──────────────────────────────────────────"
          '';
        };
      }
    );
}
