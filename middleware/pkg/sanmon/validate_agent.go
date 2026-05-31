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

// rceShells are interpreters that, when fed piped remote content, execute it.
var rceShells = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true}

// segmentReadsSecret reports whether a pipeline segment reads a secret file.
func segmentReadsSecret(seg string, secretPatterns []string) bool {
	for _, tok := range strings.Fields(seg) {
		if pathMatchesAny(tok, secretPatterns) {
			return true
		}
	}
	return false
}

// segmentIsExternalSink reports whether a segment sends data off-host.
func segmentIsExternalSink(seg string, sinks []string) bool {
	cmd := firstToken(seg)
	for _, s := range sinks {
		if cmd == s {
			return true
		}
	}
	return strings.Contains(seg, "http://") || strings.Contains(seg, "https://")
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

	segments := splitPipeline(cmd)
	readsSecret, hasExternalSink, hasRemoteFetch, pipesToShell := false, false, false, false
	for _, seg := range segments {
		if segmentReadsSecret(seg, p.SecretFilePatterns) {
			readsSecret = true
		}
		if segmentIsExternalSink(seg, p.ExternalSinkCommands) {
			hasExternalSink = true
		}
		ft := firstToken(seg)
		if ft == "curl" || ft == "wget" {
			hasRemoteFetch = true
		}
		if rceShells[ft] {
			pipesToShell = true
		}
	}
	if readsSecret && hasExternalSink {
		violations = append(violations, Violation{
			Rule:     "agent.secret_exfiltration",
			Message:  "reads a secret file and pipes it to an external host",
			Path:     "parameters.command",
			Severity: SeverityError,
		})
	}
	// PR1 intentional heuristic: fires whenever a remote-fetch command and a shell
	// interpreter appear anywhere in the same pipeline, including &&-separated
	// segments (e.g. `curl ... -o data.json && bash build.sh`). Requiring a
	// literal `|` between the fetch and shell invocation is deferred to PR2.
	if hasRemoteFetch && pipesToShell && len(segments) >= 2 {
		violations = append(violations, Violation{
			Rule:     "agent.remote_code_execution",
			Message:  "pipes remotely-fetched content into a shell interpreter",
			Path:     "parameters.command",
			Severity: SeverityError,
		})
	}

	if isGitForcePush(cmd) {
		violations = append(violations, Violation{
			Rule:     "agent.force_push",
			Message:  "force-pushing can overwrite remote history",
			Path:     "parameters.command",
			Severity: SeverityError,
		})
	}
	return violations
}

// gitShortForceFlagRe matches a short -f flag (possibly bundled, e.g. -uf) as a
// standalone argument. Compiled once at package init, not per call.
var gitShortForceFlagRe = regexp.MustCompile(`(^|\s)-[a-zA-Z]*f([a-zA-Z]*)?(\s|$)`)

// gitForceLongRe matches the dangerous --force / --force-with-lease flags as
// whole words, so the SAFE --force-if-includes flag is not matched.
var gitForceLongRe = regexp.MustCompile(`--force(-with-lease)?(\s|$)`)

// isGitForcePush reports whether the command force-pushes with git. The flag
// check is scoped to the pipeline segment that actually contains "git" and
// "push", so an unrelated -f in another subcommand (e.g. `grep -f x && git
// push`) does not trip it.
func isGitForcePush(cmd string) bool {
	for _, seg := range splitPipeline(cmd) {
		if !strings.Contains(seg, "git") || !strings.Contains(seg, "push") {
			continue
		}
		if gitForceLongRe.MatchString(seg) || gitShortForceFlagRe.MatchString(seg) {
			return true
		}
	}
	return false
}

func validateFileMutation(a *Action, p *AgentPolicy) []Violation {
	filePath := getParamString(a.Parameters, "path")
	if filePath == "" {
		filePath = a.Target
	}
	var violations []Violation

	if pathMatchesAny(filePath, p.ProtectedPaths) {
		violations = append(violations, Violation{
			Rule:     "agent.protected_path_write",
			Message:  "writing to a protected path is forbidden: " + filePath,
			Path:     "parameters.path",
			Severity: SeverityError,
		})
	}

	content := getParamString(a.Parameters, "content")
	if content != "" {
		for _, pat := range p.SecretContentPatterns {
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			if re.MatchString(content) {
				violations = append(violations, Violation{
					Rule:     "agent.secret_in_write",
					Message:  "file content appears to contain a secret/credential",
					Path:     "parameters.content",
					Severity: SeverityError,
				})
				break
			}
		}
	}
	return violations
}

func validateNetFetch(a *Action, p *AgentPolicy) []Violation {
	host := getParamString(a.Parameters, "host")
	if host == "" {
		host = hostFromURL(getParamString(a.Parameters, "url"))
	}
	if host == "" {
		host = hostFromURL(a.Target)
	}
	for _, denied := range p.DeniedNetHosts {
		if host == denied || strings.HasSuffix(host, "."+denied) {
			return []Violation{{
				Rule:     "agent.denied_net_host",
				Message:  "network access to host is forbidden: " + host,
				Path:     "parameters.host",
				Severity: SeverityError,
			}}
		}
	}
	return nil
}

// hostFromURL extracts the host from a URL string without failing on junk.
func hostFromURL(raw string) string {
	s := strings.ToLower(raw)
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.Index(s, ":"); i >= 0 {
		s = s[:i]
	}
	return s
}
