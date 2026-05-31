// Agent domain policy — constraints for a coding agent's own tool calls
// (shell execution, file writes/edits, network fetches, MCP calls).
//
// This is the single source of truth for the protective "starter" policy that
// `sanmon init` installs. The Go StarterAgentPolicy() mirrors these values.

package agent

#AgentAction: {
	action_type: "shell_exec" | "file_write" | "file_edit" | "file_read" | "net_fetch" | "mcp_call"
	target:      string & !=""
	parameters:  {[string]: _}
	context: {
		domain: "agent"
		...
	}
	...
}

#CommandRule: {
	pattern: string
	rule:    string
	message: string
}

// Starter (opt-in, protective) policy.
// Until CUE→Go generation lands (follow-up), edits to this starter policy
// MUST also be made in StarterAgentPolicy() in middleware/pkg/sanmon/policy.go.
policy: {
	deny_command_rules: [...#CommandRule] | *[
		{pattern: #"\brm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\b"#, rule: "destructive_delete", message: "recursive force-delete (rm -rf) is forbidden"},
		{pattern: #"\brm\s+-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*\b"#, rule: "destructive_delete", message: "recursive force-delete (rm -fr) is forbidden"},
		{pattern: #"\bchmod\s+-R\s+777\b"#, rule: "insecure_permissions", message: "chmod -R 777 is forbidden"},
		{pattern: #"\bdd\s+if="#, rule: "raw_disk_write", message: "raw disk writes via dd are forbidden"},
		{pattern: #"\bgit\s+reset\s+--hard\b"#, rule: "history_destruction", message: "git reset --hard is forbidden"},
		{pattern: #"\bmkfs\b"#, rule: "filesystem_format", message: "filesystem formatting (mkfs) is forbidden"},
	]
	protected_paths: [...string] | *["*/.ssh/*", "*/.aws/*", "*/.config/gh/*"]
	protected_branches: [...string] | *["main", "master"]
	denied_net_hosts: [...string] | *[]
	secret_file_patterns: [...string] | *[".env", "*.env", ".env.*", "*.pem", "id_rsa", "id_ed25519", "credentials", "*/.aws/credentials", "*/.ssh/*"]
	secret_content_patterns: [...string] | *[#"-----BEGIN [A-Z ]*PRIVATE KEY-----"#, #"AKIA[0-9A-Z]{16}"#]
	external_sink_commands: [...string] | *["curl", "wget", "nc", "ncat", "scp", "telnet", "ftp"]
}
