package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sanmon/middleware/pkg/sanmon"
)

// runInit implements: sanmon init <agent> [--dir <root>]
// It writes a starter policy (protective AgentPolicy) and prints the hook
// registration the user adds to their agent so tool calls route through
// `sanmon guard`.
func runInit(args []string) {
	if len(args) == 0 {
		fatalf("usage: sanmon init <claude|codex|generic> [--dir <root>]")
	}
	agent := args[0]
	dir := "."
	for i := 1; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			i++
			dir = args[i]
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fatalf("resolve dir: %v", err)
	}
	sanmonDir := filepath.Join(absDir, ".sanmon")
	if err := os.MkdirAll(sanmonDir, 0o755); err != nil {
		fatalf("mkdir .sanmon: %v", err)
	}

	policy := sanmon.DefaultPolicy()
	policy.Agent = sanmon.StarterAgentPolicy()
	policyPath := filepath.Join(sanmonDir, "policy.json")
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		fatalf("marshal policy: %v", err)
	}
	if err := os.WriteFile(policyPath, append(data, '\n'), 0o644); err != nil {
		fatalf("write policy: %v", err)
	}

	self, _ := os.Executable()
	if self == "" {
		self = "sanmon"
	}
	fmt.Printf("Wrote starter policy: %s\n\n", policyPath)
	printHookInstructions(agent, self, policyPath)
}

func printHookInstructions(agent, self, policyPath string) {
	guardCmd := fmt.Sprintf("%s guard --agent %s --policy %s", self, agent, policyPath)
	switch agent {
	case "claude":
		fmt.Println("Add to .claude/settings.json:")
		fmt.Printf(`{
  "hooks": {
    "PreToolUse": [
      { "matcher": "*", "hooks": [ { "type": "command", "command": %q } ] }
    ]
  }
}
`, guardCmd)
	case "codex":
		fmt.Println("Add to ~/.codex/config.toml:")
		fmt.Printf(`[[hooks.PreToolUse]]
matcher = ".*"
[[hooks.PreToolUse.hooks]]
type = "command"
command = %q
`, guardCmd)
	default:
		fmt.Println("Pipe your agent's proposed tool call (JSON) into:")
		fmt.Printf("  %s\n", guardCmd)
		fmt.Println("Generic input: {\"tool\":\"shell_exec\",\"command\":\"...\"}")
	}
}
