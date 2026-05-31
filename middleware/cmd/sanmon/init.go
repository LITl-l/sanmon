package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sanmon/middleware/pkg/sanmon"
)

// runInit implements: sanmon init <agent> [--dir <path>] [--force]
// It writes a starter policy to <dir>/.sanmon/policy.json and prints hook
// installation instructions. If the policy file already exists and --force is
// not set, it prints a message and exits without overwriting.
func runInit(args []string) {
	if len(args) == 0 {
		fatalf("usage: sanmon init <agent> [--dir <path>] [--force]")
	}
	agent := args[0]
	dir := "."
	force := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i < len(args) {
				dir = args[i]
			}
		case "--force":
			force = true
		}
	}

	if !knownAgents[agent] {
		fatalf("unknown agent %q (want: generic|claude|codex)", agent)
	}

	sanmonDir := filepath.Join(dir, ".sanmon")
	if err := os.MkdirAll(sanmonDir, 0o755); err != nil {
		fatalf("create .sanmon dir: %v", err)
	}

	policyPath := filepath.Join(sanmonDir, "policy.json")

	// Fix 4: do not clobber an existing policy unless --force is set.
	if _, err := os.Stat(policyPath); err == nil {
		// File exists.
		if !force {
			fmt.Fprintf(os.Stderr, "policy already exists at %s; re-run with --force to overwrite\n", policyPath)
			// Still print hook instructions even when skipping.
			printHookInstructions(agent, policyPath)
			return
		}
	}

	// Build and write the starter policy.
	policy := sanmon.DefaultPolicy()
	policy.Agent = sanmon.StarterAgentPolicy()

	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		fatalf("marshal policy: %v", err)
	}
	if err := os.WriteFile(policyPath, append(data, '\n'), 0o644); err != nil {
		fatalf("write policy: %v", err)
	}

	fmt.Printf("wrote starter policy → %s\n", policyPath)
	printHookInstructions(agent, policyPath)
}

func printHookInstructions(agent, policyPath string) {
	absPolicy, err := filepath.Abs(policyPath)
	if err != nil {
		absPolicy = policyPath
	}

	switch agent {
	case "claude":
		fmt.Printf(`
To enable the guard hook for Claude Code, add the following to your
.claude/settings.json (or settings.local.json):

  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "sanmon guard --agent claude --policy %s"
          }
        ]
      }
    ]
  }

`, absPolicy)

	case "codex":
		fmt.Printf(`
To enable the guard hook for Codex, add the following to your
codex configuration:

  pre_tool_use_hook: "sanmon guard --agent codex --policy %s"

`, absPolicy)

	default: // generic
		fmt.Printf(`
To use the guard in generic mode, pipe tool-call JSON through it:

  echo '{"tool":"shell_exec","command":"ls"}' | \
    sanmon guard --agent generic --policy %s

`, absPolicy)
	}
}
