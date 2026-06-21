package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionIsLdflagsOverridable asserts the release build can stamp the
// version into the binary via -ldflags "-X main.version=...".
//
// A tagged release is only trustworthy if `sanmon version` reports the tag it
// was built from. That requires `version` to be a linker-overridable package
// var — the linker cannot rewrite a `const` (constants are inlined, so no
// symbol exists to set), so this test fails until `version` is a var.
func TestVersionIsLdflagsOverridable(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH (run via nix develop)")
	}
	const want = "v9.9.9-ldflags-test"
	bin := filepath.Join(t.TempDir(), "sanmon")
	out, err := exec.Command("go", "build",
		"-ldflags", "-X main.version="+want,
		"-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	got, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("run version: %v\n%s", err, got)
	}
	if !strings.Contains(string(got), want) {
		t.Fatalf("version output %q does not contain injected version %q",
			strings.TrimSpace(string(got)), want)
	}
}
