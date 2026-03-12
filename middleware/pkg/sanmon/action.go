package sanmon

import "time"

// Action is the universal action structure that all agent actions must conform to.
type Action struct {
	ActionType string                 `json:"action_type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Context    ActionContext           `json:"context"`
	Metadata   ActionMetadata         `json:"metadata"`
}

// ActionContext provides execution context for the action.
type ActionContext struct {
	Authenticated  bool   `json:"authenticated"`
	SessionID      string `json:"session_id"`
	Domain         string `json:"domain"`
	PreviousAction string `json:"previous_action,omitempty"`
}

// ActionMetadata provides traceability information.
type ActionMetadata struct {
	Timestamp string `json:"timestamp"`
	AgentID   string `json:"agent_id"`
	RequestID string `json:"request_id"`
}

// ValidDomains is the set of recognized domains.
var ValidDomains = map[string]bool{
	"browser":  true,
	"api":      true,
	"database": true,
	"iac":      true,
	"approval": true,
}

// ValidActionTypes maps each domain to its allowed action types.
var ValidActionTypes = map[string]map[string]bool{
	"browser": {
		"navigate": true, "click": true, "fill": true,
		"select": true, "scroll": true, "wait": true, "screenshot": true,
	},
	"api": {
		"get": true, "post": true, "put": true, "patch": true, "delete": true,
	},
	"database": {
		"select": true, "insert": true, "update": true,
		"delete": true, "create_table": true, "drop_table": true,
	},
	"iac": {
		"create": true, "modify": true, "destroy": true, "plan": true, "apply": true,
	},
	"approval": {
		"approve": true, "reject": true,
	},
}

// validateStructure checks the base schema requirements (Gate 1).
func validateStructure(a *Action) []Violation {
	var violations []Violation

	if a.Target == "" {
		violations = append(violations, Violation{
			Rule: "non_empty_target", Message: "target must not be empty",
			Path: "target", Severity: SeverityError,
		})
	}

	if !ValidDomains[a.Context.Domain] {
		violations = append(violations, Violation{
			Rule: "valid_domain", Message: "domain must be one of: browser, api, database, iac, approval",
			Path: "context.domain", Severity: SeverityError,
		})
	}

	if allowed, ok := ValidActionTypes[a.Context.Domain]; ok {
		if !allowed[a.ActionType] {
			violations = append(violations, Violation{
				Rule: "valid_action_type", Message: "action_type is not valid for domain " + a.Context.Domain,
				Path: "action_type", Severity: SeverityError,
			})
		}
	}

	if a.Context.SessionID == "" {
		violations = append(violations, Violation{
			Rule: "non_empty_session_id", Message: "session_id must not be empty",
			Path: "context.session_id", Severity: SeverityError,
		})
	}

	if a.Metadata.AgentID == "" {
		violations = append(violations, Violation{
			Rule: "non_empty_agent_id", Message: "agent_id must not be empty",
			Path: "metadata.agent_id", Severity: SeverityError,
		})
	}

	if a.Metadata.RequestID == "" {
		violations = append(violations, Violation{
			Rule: "non_empty_request_id", Message: "request_id must not be empty",
			Path: "metadata.request_id", Severity: SeverityError,
		})
	}

	if _, err := time.Parse(time.RFC3339, a.Metadata.Timestamp); err != nil {
		violations = append(violations, Violation{
			Rule: "valid_timestamp", Message: "timestamp must be RFC3339 format",
			Path: "metadata.timestamp", Severity: SeverityError,
		})
	}

	return violations
}
