package sanmon

import "fmt"

func validateApproval(a *Action, p *ApprovalPolicy) []Violation {
	if len(p.Rules) == 0 {
		return []Violation{{
			Rule:     "no_approval_rules",
			Message:  "no approval rules configured; all automated decisions are blocked",
			Path:     "action_type",
			Severity: SeverityError,
		}}
	}

	doc := getDocument(a.Parameters)
	if doc == nil {
		return []Violation{{
			Rule:     "missing_document",
			Message:  "approval action must include parameters.document with document attributes",
			Path:     "parameters.document",
			Severity: SeverityError,
		}}
	}

	for _, rule := range p.Rules {
		if !matchesAllConditions(doc, rule.Conditions) {
			continue
		}

		// First matching rule found
		if rule.Decision == "manual_review" {
			return []Violation{{
				Rule:     "manual_review_required",
				Message:  fmt.Sprintf("rule %q requires manual review; automated %s is not allowed", rule.Name, a.ActionType),
				Path:     "action_type",
				Severity: SeverityError,
			}}
		}

		if a.ActionType != rule.Decision {
			return []Violation{{
				Rule:    "decision_mismatch",
				Message: fmt.Sprintf("rule %q expects %q but agent proposed %q", rule.Name, rule.Decision, a.ActionType),
				Path:    "action_type",
				Severity: SeverityError,
			}}
		}

		// Decision matches the rule
		return nil
	}

	// No rule matched — fail closed
	return []Violation{{
		Rule:     "no_matching_rule",
		Message:  "no approval rule matched this document; automated decision is blocked",
		Path:     "parameters.document",
		Severity: SeverityError,
	}}
}

func getDocument(params map[string]interface{}) map[string]interface{} {
	v, ok := params["document"]
	if !ok {
		return nil
	}
	doc, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return doc
}

func matchesAllConditions(doc map[string]interface{}, conditions []ApprovalCondition) bool {
	for _, cond := range conditions {
		if !evaluateCondition(doc, cond) {
			return false
		}
	}
	return true
}

func evaluateCondition(doc map[string]interface{}, cond ApprovalCondition) bool {
	val, ok := doc[cond.Field]
	if !ok {
		return false
	}

	switch cond.Operator {
	case "eq":
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", cond.Value)
	case "neq":
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", cond.Value)
	case "lt", "gt", "lte", "gte":
		return compareNumeric(val, cond.Value, cond.Operator)
	case "in":
		return containsValue(cond.Value, val)
	case "not_in":
		return !containsValue(cond.Value, val)
	default:
		return false
	}
}

func compareNumeric(docVal, ruleVal interface{}, op string) bool {
	a := toFloat64(docVal)
	b := toFloat64(ruleVal)
	if a == nil || b == nil {
		return false
	}
	switch op {
	case "lt":
		return *a < *b
	case "gt":
		return *a > *b
	case "lte":
		return *a <= *b
	case "gte":
		return *a >= *b
	}
	return false
}

func toFloat64(v interface{}) *float64 {
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	default:
		return nil
	}
}

func containsValue(set interface{}, val interface{}) bool {
	s := fmt.Sprintf("%v", val)
	switch list := set.(type) {
	case []interface{}:
		for _, item := range list {
			if fmt.Sprintf("%v", item) == s {
				return true
			}
		}
	case []string:
		for _, item := range list {
			if item == s {
				return true
			}
		}
	}
	return false
}
