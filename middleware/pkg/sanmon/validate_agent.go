package sanmon

import (
	"path"
	"strings"
)

// validateAgent enforces the agent domain policy against a normalized
// coding-agent tool call, dispatching on ActionType.
func validateAgent(a *Action, p *AgentPolicy) []Violation {
	switch a.ActionType {
	case "shell_exec":
		return validateShellExec(a, p)
	case "file_write", "file_edit":
		return validateFileMutation(a, p)
	case "net_fetch":
		return validateNetFetch(a, p)
	case "file_read", "mcp_call":
		// Read-class actions carry no PR1 constraints (fail-open class).
		return nil
	default:
		return []Violation{{
			Rule: "unknown_agent_action", Message: "unknown agent action_type: " + a.ActionType,
			Path: "action_type", Severity: SeverityError,
		}}
	}
}

// normalizeCommand trims and collapses whitespace so pattern matching is
// resistant to trivial spacing tricks. Deeper normalization (quoting,
// base64, variable expansion) is future work.
func normalizeCommand(cmd string) string {
	return strings.Join(strings.Fields(cmd), " ")
}

// splitPipeline splits a command line into segments on |, ;, &&, ||.
// Not quote-aware (PR2 improves this); good enough for the headline cases.
func splitPipeline(cmd string) []string {
	r := strings.NewReplacer("&&", "\x00", "||", "\x00", ";", "\x00", "|", "\x00")
	var out []string
	for _, seg := range strings.Split(r.Replace(cmd), "\x00") {
		if s := strings.TrimSpace(seg); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// firstToken returns the leading command word of a segment.
func firstToken(seg string) string {
	f := strings.Fields(seg)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// commandForAction reads the shell command from parameters.command, falling
// back to the target.
func commandForAction(a *Action) string {
	if c := getParamString(a.Parameters, "command"); c != "" {
		return c
	}
	return a.Target
}

// pathMatchesAny reports whether p matches any glob (path.Match) in patterns.
func pathMatchesAny(p string, patterns []string) bool {
	base := path.Base(p)
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
	}
	return false
}

// validateShellExec, validateFileMutation, validateNetFetch are completed in
// later tasks; PR1 builds them incrementally.
func validateShellExec(a *Action, p *AgentPolicy) []Violation   { return nil }
func validateFileMutation(a *Action, p *AgentPolicy) []Violation { return nil }
func validateNetFetch(a *Action, p *AgentPolicy) []Violation     { return nil }
