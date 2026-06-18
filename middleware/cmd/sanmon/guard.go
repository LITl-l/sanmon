package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sanmon/middleware/pkg/sanmon"
)

// emitAudit stamps the record with the current time and writes it to the audit
// sink selected by SANMON_AUDIT:
//   - unset or "stderr": write the JSON Lines record to stderr (default)
//   - "off":             disable audit logging
//   - any other value:   treat as a file path, appended to (falls back to
//     stderr if the file cannot be opened)
//
// Audit logging never affects the decision on stdout or the exit code.
func emitAudit(rec sanmon.AuditRecord) {
	rec.Time = time.Now().UTC().Format(time.RFC3339)
	switch v := os.Getenv("SANMON_AUDIT"); v {
	case "off":
		return
	case "", "stderr":
		sanmon.WriteAudit(os.Stderr, rec)
	default:
		f, err := os.OpenFile(v, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sanmon guard: cannot open audit file, using stderr: "+err.Error())
			sanmon.WriteAudit(os.Stderr, rec)
			return
		}
		defer f.Close()
		sanmon.WriteAudit(f, rec)
	}
}

// knownAgents is the set of agents accepted by runGuard.
var knownAgents = map[string]bool{
	"generic": true,
	"claude":  true,
	"codex":   true,
}

// runGuard implements: sanmon guard --agent <name> [--policy <path>]
// It reads an agent's proposed tool call as JSON on stdin and writes that
// agent's native allow/deny decision as JSON on stdout (only). Diagnostics go
// to stderr. Destructive actions fail CLOSED (exit 2) on internal error;
// read-class actions fail OPEN.
func runGuard(args []string) {
	agent := "generic"
	policyPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i < len(args) {
				agent = args[i]
			}
		case "--policy":
			i++
			if i < len(args) {
				policyPath = args[i]
			}
		}
	}

	// Fix 2: validate --agent early, before reading stdin.
	if !knownAgents[agent] {
		fatalf("unknown agent %q (want: generic|claude|codex)", agent)
	}

	// Fix 1: if --policy was explicitly provided and the file does not exist,
	// fail-closed immediately (do NOT fall back to permissive default).
	if policyPath != "" {
		if _, err := os.Stat(policyPath); err != nil {
			if os.IsNotExist(err) {
				// A missing explicit --policy is an operator error, not a normal condition, so
				// we fail closed for ALL action classes here (not just destructive): better to
				// block than to silently fall back to the permissive default.
				// We don't have class yet; use ClassDestructive to be conservative.
				guardFailClosed(agent, sanmon.ClassDestructive, "policy file not found: "+policyPath)
			} else {
				guardFailClosed(agent, sanmon.ClassDestructive, "cannot stat policy file: "+err.Error())
			}
			return
		}
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		guardFailClosed(agent, sanmon.ClassDestructive, "cannot read stdin: "+err.Error())
		return
	}

	action, class, err := sanmon.DecodeAgentPayload(agent, data)
	if err != nil {
		guardFailClosed(agent, class, "cannot decode payload: "+err.Error())
		return
	}

	var policy *sanmon.Policy
	if policyPath != "" {
		policy, err = sanmon.LoadPolicy(policyPath)
		if err != nil {
			guardFailClosed(agent, class, "cannot load policy: "+err.Error())
			return
		}
	} else {
		policy = sanmon.DefaultPolicy()
	}

	result := sanmon.NewEngine(policy).Validate(action)
	out := sanmon.EncodeDecision(agent, result)
	fmt.Fprintln(os.Stdout, string(out))
	emitAudit(sanmon.AuditForDecision(agent, action, result))

	// Fix 3: generic denies exit 2 so exit-code-driven callers detect a block.
	// claude/codex always exit 0 (the decision rides in the JSON body).
	os.Exit(guardExitCode(agent, result.Pass))
}

// guardExitCode returns the process exit code for a completed decision.
// generic denies exit 2 (so exit-code-driven callers detect a block);
// claude/codex always exit 0 (the decision rides in the JSON body).
func guardExitCode(agent string, pass bool) int {
	if agent == "generic" && !pass {
		return 2
	}
	return 0
}

// guardFailClosed applies the asymmetric failure policy: destructive actions
// are blocked (exit 2, reason on stderr); read-class actions are allowed
// (emit an allow decision, exit 0).
func guardFailClosed(agent string, class sanmon.ActionClass, reason string) {
	emitAudit(sanmon.AuditFailClosed(agent, class, reason))
	if class == sanmon.ClassRead {
		fmt.Fprintln(os.Stdout, string(sanmon.EncodeDecision(agent, passResult())))
		return
	}
	fmt.Fprintln(os.Stderr, "sanmon guard: "+reason)
	os.Exit(2)
}

// passResult builds an allow result for fail-open paths.
func passResult() sanmon.ValidationResult {
	return sanmon.ValidationResult{Pass: true, Violations: []sanmon.Violation{}}
}
