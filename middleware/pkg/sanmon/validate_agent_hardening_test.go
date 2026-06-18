package sanmon

import "testing"

// These tests encode known bypasses of the string-based guard. They must fail
// before parser-based analysis lands and pass after.

// Class 1: obfuscation of a real command via quoting. A proper parser
// reconstructs the literal command word, so `r''m` / `"rm"` resolve to `rm`
// and the denylist matches.
func TestShellQuoteInsertionBypass(t *testing.T) {
	p := StarterAgentPolicy()
	cases := []struct{ cmd, rule string }{
		{`r''m -rf /tmp/x`, "destructive_delete"},
		{`"rm" -rf /tmp/x`, "destructive_delete"},
		{`ch""mod -R 777 /etc`, "insecure_permissions"},
		{`sudo "rm" -fr /var`, "destructive_delete"},
	}
	for _, c := range cases {
		a := agentAction("shell_exec", c.cmd, map[string]interface{}{"command": c.cmd})
		if v := validateShellExec(a, &p); !hasRule(v, c.rule) {
			t.Errorf("cmd %q: expected rule %q, got %v", c.cmd, c.rule, v)
		}
	}
}

// Class 2a: recursive-force delete with flags in any order / separated /
// long form — a regex over a single flag token cannot express this.
func TestShellSeparatedFlagDelete(t *testing.T) {
	p := StarterAgentPolicy()
	deny := []string{
		"rm -r -f /important",
		"rm -f -r /important",
		"rm --recursive --force /data",
		"rm -fr /data", // regression: bundled form still caught
	}
	for _, cmd := range deny {
		a := agentAction("shell_exec", cmd, map[string]interface{}{"command": cmd})
		if v := validateShellExec(a, &p); !hasRule(v, "destructive_delete") {
			t.Errorf("cmd %q: expected destructive_delete, got %v", cmd, v)
		}
	}
	// rm without BOTH recursive and force is allowed (avoid false positives).
	allow := []string{"rm file.txt", "rm -f file.txt", "rm -r emptydir"}
	for _, cmd := range allow {
		a := agentAction("shell_exec", cmd, map[string]interface{}{"command": cmd})
		if v := validateShellExec(a, &p); hasRule(v, "destructive_delete") {
			t.Errorf("cmd %q should be allowed, got %v", cmd, v)
		}
	}
}

// Class 2b: decode-and-execute obfuscation (e.g. base64 -d piped to a shell).
func TestShellObfuscatedExecution(t *testing.T) {
	p := StarterAgentPolicy()
	deny := []string{
		"echo cm0gLXJmIH4K | base64 -d | sh",
		"printf %s cm0= | base64 --decode | bash",
		"cat payload.b64 | base64 -d | zsh",
	}
	for _, cmd := range deny {
		a := agentAction("shell_exec", cmd, map[string]interface{}{"command": cmd})
		if v := validateShellExec(a, &p); !hasRule(v, "obfuscated_execution") {
			t.Errorf("cmd %q: expected obfuscated_execution, got %v", cmd, v)
		}
	}
	// Decoding to a file (not piped into a shell) is allowed.
	ok := "echo aGk= | base64 -d > out.txt"
	a := agentAction("shell_exec", ok, map[string]interface{}{"command": ok})
	if v := validateShellExec(a, &p); hasRule(v, "obfuscated_execution") {
		t.Errorf("cmd %q should be allowed, got %v", ok, v)
	}
}
