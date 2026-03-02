package sanmon

import (
	"fmt"
	"strings"
)

// dangerousSchemes is the list of URI schemes blocked by the browser policy.
var dangerousSchemes = []string{"javascript:", "data:", "vbscript:"}

func validateBrowser(a *Action, p *BrowserPolicy) []Violation {
	var violations []Violation

	// Gate 2: Policy rules

	// Check dangerous URI schemes on navigate
	if a.ActionType == "navigate" && p.NoDangerousSchemes {
		url := getParamString(a.Parameters, "url")
		if url == "" {
			url = a.Target
		}
		lower := strings.ToLower(url)
		for _, scheme := range dangerousSchemes {
			if strings.HasPrefix(lower, scheme) {
				violations = append(violations, Violation{
					Rule:     "dangerous_scheme",
					Message:  fmt.Sprintf("URI scheme %q is forbidden", scheme),
					Path:     "parameters.url",
					Severity: SeverityError,
				})
				break
			}
		}
	}

	// Check forbidden selectors on click/fill/select
	if a.ActionType == "click" || a.ActionType == "fill" || a.ActionType == "select" {
		selector := getParamString(a.Parameters, "selector")
		if selector == "" {
			selector = a.Target
		}
		for _, forbidden := range p.ForbiddenSelectors {
			if selector == forbidden {
				violations = append(violations, Violation{
					Rule:     "forbidden_selector",
					Message:  fmt.Sprintf("selector %q is forbidden by policy", forbidden),
					Path:     "parameters.selector",
					Severity: SeverityError,
				})
			}
		}
	}

	// Check input length on fill
	if a.ActionType == "fill" {
		value := getParamString(a.Parameters, "value")
		if p.MaxInputLength > 0 && len(value) > p.MaxInputLength {
			violations = append(violations, Violation{
				Rule:     "input_too_long",
				Message:  fmt.Sprintf("input value length %d exceeds maximum %d", len(value), p.MaxInputLength),
				Path:     "parameters.value",
				Severity: SeverityError,
			})
		}
	}

	// Check fill action has required fields
	if a.ActionType == "fill" {
		if getParamString(a.Parameters, "selector") == "" {
			violations = append(violations, Violation{
				Rule: "fill_requires_selector", Message: "fill action requires a non-empty selector",
				Path: "parameters.selector", Severity: SeverityError,
			})
		}
	}

	// Check click action has selector
	if a.ActionType == "click" {
		if getParamString(a.Parameters, "selector") == "" {
			violations = append(violations, Violation{
				Rule: "click_requires_selector", Message: "click action requires a non-empty selector",
				Path: "parameters.selector", Severity: SeverityError,
			})
		}
	}

	return violations
}

func getParamString(params map[string]interface{}, key string) string {
	v, ok := params[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
