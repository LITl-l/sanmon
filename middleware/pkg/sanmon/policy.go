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
