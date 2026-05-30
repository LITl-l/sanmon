package main

import (
	"fmt"
	"io"
	"os"

	"github.com/sanmon/middleware/pkg/sanmon"
)

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
	// Decision is in the JSON body; exit 0 so the agent reads stdout.
}

// guardFailClosed applies the asymmetric failure policy: destructive actions
// are blocked (exit 2, reason on stderr); read-class actions are allowed
// (emit an allow decision, exit 0).
func guardFailClosed(agent string, class sanmon.ActionClass, reason string) {
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
