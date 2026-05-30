package sanmon

import (
	"path"
	"regexp"
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

// pathMatchesAny reports whether filePath matches any glob in patterns.
// path.Match wildcards do not cross "/", so we also test the basename and every
// trailing path suffix — this lets a segment glob like "*/.ssh/*" match an
// absolute path such as "/home/user/.ssh/id_rsa".
func pathMatchesAny(filePath string, patterns []string) bool {
	base := path.Base(filePath)
	segs := strings.Split(filePath, "/")
	suffixes := make([]string, 0, len(segs))
	for i := range segs {
		if s := strings.Join(segs[i:], "/"); s != "" {
			suffixes = append(suffixes, s)
		}
	}
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, filePath); ok {
			return true
		}
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
		for _, suf := range suffixes {
			if ok, _ := path.Match(pat, suf); ok {
				return true
			}
		}
	}
	return false
}

func validateShellExec(a *Action, p *AgentPolicy) []Violation {
	cmd := normalizeCommand(commandForAction(a))
	if cmd == "" {
		return nil
	}
	var violations []Violation

	for _, rule := range p.DenyCommandRules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue // skip malformed policy patterns; do not crash the guard
		}
		if re.MatchString(cmd) {
			violations = append(violations, Violation{
				Rule:     "agent." + rule.Rule,
				Message:  rule.Message,
				Path:     "parameters.command",
				Severity: SeverityError,
			})
		}
	}
	return violations
}

func validateFileMutation(a *Action, p *AgentPolicy) []Violation { return nil }
func validateNetFetch(a *Action, p *AgentPolicy) []Violation     { return nil }
