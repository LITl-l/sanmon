package sanmon

import "fmt"

func validateIaC(a *Action, p *IaCPolicy) []Violation {
	var violations []Violation

	// Destroy is forbidden by default
	if a.ActionType == "destroy" && !p.AllowDestroy {
		violations = append(violations, Violation{
			Rule:     "destroy_forbidden",
			Message:  "destroy action is forbidden by policy",
			Path:     "action_type",
			Severity: SeverityError,
		})
	}

	// Check required tags on create/modify/apply
	if a.ActionType == "create" || a.ActionType == "modify" || a.ActionType == "apply" {
		tags := getParamMap(a.Parameters, "tags")
		for _, reqTag := range p.RequiredTags {
			if _, ok := tags[reqTag]; !ok {
				violations = append(violations, Violation{
					Rule:     "missing_required_tag",
					Message:  fmt.Sprintf("required tag %q is missing", reqTag),
					Path:     "parameters.tags",
					Severity: SeverityError,
				})
			}
		}
	}

	// Block open ingress (0.0.0.0/0)
	if p.BlockOpenIngress && (a.ActionType == "create" || a.ActionType == "modify") {
		if hasOpenIngress(a.Parameters) {
			violations = append(violations, Violation{
				Rule:     "open_ingress_blocked",
				Message:  "open ingress (0.0.0.0/0) is blocked by policy",
				Path:     "parameters.properties.ingress",
				Severity: SeverityError,
			})
		}
	}

	return violations
}

// hasOpenIngress checks if parameters contain a 0.0.0.0/0 CIDR in ingress rules.
func hasOpenIngress(params map[string]interface{}) bool {
	props := getParamMap(params, "properties")
	if props == nil {
		return false
	}
	ingress, ok := props["ingress"]
	if !ok {
		return false
	}
	return containsCIDR(ingress, "0.0.0.0/0")
}

func containsCIDR(v interface{}, cidr string) bool {
	switch val := v.(type) {
	case string:
		return val == cidr
	case []interface{}:
		for _, item := range val {
			if containsCIDR(item, cidr) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range val {
			if containsCIDR(item, cidr) {
				return true
			}
		}
	}
	return false
}
