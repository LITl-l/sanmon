// Base action schema — all agent actions must conform to this structure.
// Domain-specific policies in policy/domains/ extend these constraints.

package base

import "time"

#Action: {
	action_type: #ActionType
	target:      string & !=""
	parameters:  {[string]: _}
	context:     #Context
	metadata:    #Metadata
}

#ActionType:
	#BrowserActionType |
	#ApiActionType |
	#DatabaseActionType |
	#IacActionType |
	#ApprovalActionType |
	#AgentActionType

#BrowserActionType: "navigate" | "click" | "fill" | "select" | "scroll" | "wait" | "screenshot"
#ApiActionType:     "get" | "post" | "put" | "patch" | "delete"
#DatabaseActionType: "select" | "insert" | "update" | "delete" | "create_table" | "drop_table"
#IacActionType:     "create" | "modify" | "destroy" | "plan" | "apply"
#ApprovalActionType: "approve" | "reject"
#AgentActionType:   "shell_exec" | "file_write" | "file_edit" | "file_read" | "net_fetch" | "mcp_call"

#Context: {
	authenticated: bool
	session_id:    string & !=""
	previous_action?: #ActionType
	domain:        #Domain
}

#Domain: "browser" | "api" | "database" | "iac" | "approval" | "agent"

#Metadata: {
	timestamp:  time.Format(time.RFC3339)
	agent_id:   string & !=""
	request_id: string & !=""
}

// Validation result returned by the engine
#ValidationResult: {
	pass:        bool
	violations:  [...#Violation]
}

#Violation: {
	rule:    string
	message: string
	path:    string
	severity: "error" | "warning"
}
