package sanmon

import (
	"path"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// shellCmd is one simple command extracted from a shell line, with its words
// reduced to their static literal value: quoting that only serves to obfuscate
// (e.g. `r''m`, `"rm"`, `ch""mod`) collapses to the real token, so detectors
// see the true command and arguments.
type shellCmd struct {
	name string   // base name of the leading word (e.g. "rm" for "/bin/rm")
	args []string // remaining words, literalized
	line string   // "name args..." reconstruction, for regex denylist scanning
}

// parseShellCommands extracts every simple command in src — including those
// inside pipelines, &&/||/; lists, subshells, and command substitutions.
// ok is false when the line cannot be parsed as shell, signalling callers to
// fall back to the raw string heuristics rather than trusting a partial parse.
func parseShellCommands(src string) (cmds []shellCmd, ok bool) {
	file, err := syntax.NewParser().Parse(strings.NewReader(src), "")
	if err != nil {
		return nil, false
	}
	syntax.Walk(file, func(n syntax.Node) bool {
		call, isCall := n.(*syntax.CallExpr)
		if !isCall || len(call.Args) == 0 {
			return true
		}
		words := make([]string, 0, len(call.Args))
		for _, w := range call.Args {
			if lit := wordLiteral(w); lit != "" {
				words = append(words, lit)
			}
		}
		if len(words) == 0 {
			return true
		}
		cmds = append(cmds, shellCmd{
			name: path.Base(words[0]),
			args: words[1:],
			line: strings.Join(words, " "),
		})
		return true
	})
	return cmds, true
}

// wordLiteral concatenates the statically-known parts of a shell word. Quoted
// and unquoted literals contribute their text; parameter/command/arithmetic
// expansions are not statically resolvable and contribute nothing.
func wordLiteral(w *syntax.Word) string {
	var b strings.Builder
	for _, part := range w.Parts {
		b.WriteString(partLiteral(part))
	}
	return b.String()
}

func partLiteral(part syntax.WordPart) string {
	switch p := part.(type) {
	case *syntax.Lit:
		return p.Value
	case *syntax.SglQuoted:
		return p.Value
	case *syntax.DblQuoted:
		var b strings.Builder
		for _, sub := range p.Parts {
			b.WriteString(partLiteral(sub))
		}
		return b.String()
	default:
		return ""
	}
}

// hasShortFlagLetter reports whether any short flag among args (e.g. "-rf",
// "-r") contains letter, OR any long flag equals one of longForms.
func hasShortFlagLetter(args []string, letters string, longForms ...string) bool {
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--"):
			for _, lf := range longForms {
				if a == lf {
					return true
				}
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			if strings.ContainsAny(a[1:], letters) {
				return true
			}
		}
	}
	return false
}

// isRecursiveForceDelete reports whether cmd is an `rm` that combines a
// recursive flag and a force flag, in any order or form (-rf, -r -f,
// --recursive --force). Regex over a single flag token cannot express this.
func isRecursiveForceDelete(cmd shellCmd) bool {
	if cmd.name != "rm" {
		return false
	}
	recursive := hasShortFlagLetter(cmd.args, "rR", "--recursive")
	force := hasShortFlagLetter(cmd.args, "f", "--force")
	return recursive && force
}

// decoderCmds are commands whose purpose, with the given flags, is to turn
// opaque/encoded input back into executable text.
func isDecoder(cmd shellCmd) bool {
	switch cmd.name {
	case "base64", "base32":
		return hasShortFlagLetter(cmd.args, "dD", "--decode")
	case "xxd":
		return hasShortFlagLetter(cmd.args, "r", "--revert")
	case "uudecode":
		return true
	case "openssl":
		return containsArg(cmd.args, "enc") && (containsArg(cmd.args, "-d") || containsArg(cmd.args, "-decrypt"))
	}
	return false
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
