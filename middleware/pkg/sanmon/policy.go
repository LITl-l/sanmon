package sanmon

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Policy is the top-level policy configuration loaded from a config file.
type Policy struct {
	Browser  BrowserPolicy  `json:"browser"`
	API      APIPolicy      `json:"api"`
	Database DatabasePolicy `json:"database"`
	IaC      IaCPolicy      `json:"iac"`
	Approval ApprovalPolicy `json:"approval"`
	Agent    AgentPolicy    `json:"agent"`
}

// ApprovalPolicy defines rules for document approval workflows.
// Rules are evaluated top-to-bottom; the first matching rule determines the expected decision.
type ApprovalPolicy struct {
	Rules []ApprovalRule `json:"rules"`
}

// ApprovalRule is a single approval policy rule.
type ApprovalRule struct {
	Name       string              `json:"name"`
	Decision   string              `json:"decision"` // "approve", "reject", "manual_review"
	Conditions []ApprovalCondition `json:"conditions"`
}

// ApprovalCondition is a predicate on a document field.
type ApprovalCondition struct {
	Field    string      `json:"field"`    // document field name (e.g., "amount", "department")
	Operator string      `json:"operator"` // "lt", "gt", "lte", "gte", "eq", "neq", "in", "not_in"
	Value    interface{} `json:"value"`    // number, string, or []string
}

// BrowserPolicy defines constraints for browser automation agents.
type BrowserPolicy struct {
	AllowedURLPatterns  []string `json:"allowed_url_patterns"`
	ForbiddenSelectors  []string `json:"forbidden_selectors"`
	MaxInputLength      int      `json:"max_input_length"`
	NoDangerousSchemes  bool     `json:"no_dangerous_schemes"`
}

// APIPolicy defines constraints for API/MCP agents.
type APIPolicy struct {
	AllowedEndpoints       []string            `json:"allowed_endpoints"`
	MethodRestrictions     map[string][]string  `json:"method_restrictions"`
	RequireAuthForMutation bool                 `json:"require_auth_for_mutations"`
}

// DatabasePolicy defines constraints for SQL/database agents.
type DatabasePolicy struct {
	ReadOnlyTables          []string `json:"read_only_tables"`
	NoDeleteTables          []string `json:"no_delete_tables"`
	SensitiveColumns        []string `json:"sensitive_columns"`
	RequireWhereForMutation bool     `json:"require_where_for_mutations"`
	AllowDropTable          bool     `json:"allow_drop_table"`
	MaxJoinDepth            int      `json:"max_join_depth"`
}

// IaCPolicy defines constraints for infrastructure-as-code agents.
type IaCPolicy struct {
	AllowedResourceTypes []string `json:"allowed_resource_types"`
	RequiredTags         []string `json:"required_tags"`
	AllowDestroy         bool     `json:"allow_destroy"`
	BlockOpenIngress     bool     `json:"block_open_ingress"`
	AllowPlan            bool     `json:"allow_plan"`
}

// CommandRule is a single named shell-command deny rule.
// Pattern is an RE2 regex matched against the normalized command.
type CommandRule struct {
	Pattern string `json:"pattern"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// AgentPolicy defines constraints for a coding agent's own tool calls
// (shell execution, file writes/edits, network fetches, MCP calls).
// Empty fields mean "no constraint" — the default policy is permissive;
// StarterAgentPolicy() is the opt-in protective set installed by `sanmon init`.
type AgentPolicy struct {
	DenyCommandRules      []CommandRule `json:"deny_command_rules"`
	ProtectedPaths        []string      `json:"protected_paths"`         // globs (path.Match) for file_write/file_edit
	ProtectedBranches     []string      `json:"protected_branches"`      // reserved for force-push refinement (PR2)
	DeniedNetHosts        []string      `json:"denied_net_hosts"`        // suffix match for net_fetch host
	SecretFilePatterns    []string      `json:"secret_file_patterns"`    // globs; reading+piping these to a sink = exfil
	SecretContentPatterns []string      `json:"secret_content_patterns"` // RE2 regex on file_write content
	ExternalSinkCommands  []string      `json:"external_sink_commands"`  // commands that send data off-host (curl, wget, ...)
}

// DefaultPolicy returns a policy with sensible defaults matching the CUE definitions.
func DefaultPolicy() *Policy {
	return &Policy{
		Browser: BrowserPolicy{
			AllowedURLPatterns: []string{"*.example.com"},
			ForbiddenSelectors: []string{
				"[data-testid='delete-all']",
				"[data-testid='admin-reset']",
				".danger-zone button",
			},
			MaxInputLength:     1000,
			NoDangerousSchemes: true,
		},
		API: APIPolicy{
			AllowedEndpoints:       []string{},
			MethodRestrictions:     map[string][]string{},
			RequireAuthForMutation: true,
		},
		Database: DatabasePolicy{
			ReadOnlyTables:          []string{},
			NoDeleteTables:          []string{},
			SensitiveColumns:        []string{},
			RequireWhereForMutation: true,
			AllowDropTable:          false,
			MaxJoinDepth:            3,
		},
		IaC: IaCPolicy{
			AllowedResourceTypes: []string{},
			RequiredTags:         []string{"owner", "environment", "project"},
			AllowDestroy:         false,
			BlockOpenIngress:     true,
			AllowPlan:            true,
		},
		Approval: ApprovalPolicy{
			Rules: []ApprovalRule{
				{
					Name:     "auto_approve_small_opex",
					Decision: "approve",
					Conditions: []ApprovalCondition{
						{Field: "amount", Operator: "lt", Value: float64(100000)},
						{Field: "category", Operator: "eq", Value: "operational_expenditure"},
					},
				},
				{
					Name:     "reject_large_capex",
					Decision: "reject",
					Conditions: []ApprovalCondition{
						{Field: "amount", Operator: "gt", Value: float64(500000)},
						{Field: "category", Operator: "eq", Value: "capital_expenditure"},
					},
				},
				{
					Name:     "manual_review_very_large",
					Decision: "manual_review",
					Conditions: []ApprovalCondition{
						{Field: "amount", Operator: "gt", Value: float64(1000000)},
					},
				},
				{
					Name:       "default_reject",
					Decision:   "reject",
					Conditions: []ApprovalCondition{},
				},
			},
		},
		Agent: AgentPolicy{}, // permissive by default; see StarterAgentPolicy()
	}
}

// StarterAgentPolicy returns the opinionated, protective agent policy that
// `sanmon init` installs. These patterns are mirrored in
// policy/domains/agent/policy.cue (the single source of truth).
//
// Until CUE→Go generation lands (follow-up), edits here MUST also be made in
// policy/domains/agent/policy.cue.
func StarterAgentPolicy() AgentPolicy {
	return AgentPolicy{
		DenyCommandRules: []CommandRule{
			{Pattern: `\brm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\b`, Rule: "destructive_delete", Message: "recursive force-delete (rm -rf) is forbidden"},
			{Pattern: `\brm\s+-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*\b`, Rule: "destructive_delete", Message: "recursive force-delete (rm -fr) is forbidden"},
			{Pattern: `\bchmod\s+-R\s+777\b`, Rule: "insecure_permissions", Message: "chmod -R 777 is forbidden"},
			{Pattern: `\bdd\s+if=`, Rule: "raw_disk_write", Message: "raw disk writes via dd are forbidden"},
			{Pattern: `\bgit\s+reset\s+--hard\b`, Rule: "history_destruction", Message: "git reset --hard is forbidden"},
			{Pattern: `\bmkfs\b`, Rule: "filesystem_format", Message: "filesystem formatting (mkfs) is forbidden"},
			{Pattern: `:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}`, Rule: "fork_bomb", Message: "fork bomb is forbidden"},
		},
		ProtectedPaths: []string{
			"*/.ssh/*", "*/.aws/*", "*/.config/gh/*",
		},
		ProtectedBranches: []string{"main", "master"},
		DeniedNetHosts:    []string{},
		SecretFilePatterns: []string{
			".env", "*.env", ".env.*", "*.pem", "id_rsa", "id_ed25519",
			"credentials", "*/.aws/credentials", "*/.ssh/*",
		},
		SecretContentPatterns: []string{
			`-----BEGIN [A-Z ]*PRIVATE KEY-----`,
			`AKIA[0-9A-Z]{16}`,
		},
		ExternalSinkCommands: []string{"curl", "wget", "nc", "ncat", "scp", "telnet", "ftp"},
	}
}

// LoadPolicy loads a policy from a JSON file. Falls back to defaults if file doesn't exist.
func LoadPolicy(path string) (*Policy, error) {
	p := DefaultPolicy()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	return p, nil
}
