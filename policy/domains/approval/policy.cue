// Approval domain policy — constraints for document-approval workflow agents
// (auto-approve / reject / escalate-to-human decisions on structured documents).
//
// This mirrors the Go types in middleware/pkg/sanmon/policy.go
// (ApprovalPolicy / ApprovalRule / ApprovalCondition) and the default rules in
// DefaultPolicy(). Until CUE→Go generation lands (follow-up), edits to the
// rule set here MUST also be made in DefaultPolicy() in policy.go.

package approval

#ApprovalAction: {
	action_type: "approve" | "reject"
	target:      string & !=""  // document ID
	parameters:  #ApprovalParams
	context: {
		domain: "approval"
		...
	}
	...
}

#ApprovalParams: {
	// The document under review; rule conditions match against its fields.
	document: {[string]: _}
	...
}

// ── Policy rules ──

#Decision: "approve" | "reject" | "manual_review"

#Operator: "lt" | "gt" | "lte" | "gte" | "eq" | "neq" | "in" | "not_in"

#Condition: {
	field:    string & !=""
	operator: #Operator
	value:    _  // number, string, or list of strings
}

#Rule: {
	name:       string & !=""
	decision:   #Decision
	conditions: [...#Condition]
}

// Rules are evaluated top-to-bottom; the first rule whose conditions all match
// decides the outcome. A `manual_review` decision blocks automation. With no
// rules configured the engine blocks every automated decision (fail-closed).
policy: {
	rules: [...#Rule] | *[
		{
			name:     "auto_approve_small_opex"
			decision: "approve"
			conditions: [
				{field: "amount", operator: "lt", value: 100000},
				{field: "category", operator: "eq", value: "operational_expenditure"},
			]
		},
		{
			name:     "reject_large_capex"
			decision: "reject"
			conditions: [
				{field: "amount", operator: "gt", value: 500000},
				{field: "category", operator: "eq", value: "capital_expenditure"},
			]
		},
		{
			name:     "manual_review_very_large"
			decision: "manual_review"
			conditions: [
				{field: "amount", operator: "gt", value: 1000000},
			]
		},
		{
			name:       "default_reject"
			decision:   "reject"
			conditions: []
		},
	]
}
