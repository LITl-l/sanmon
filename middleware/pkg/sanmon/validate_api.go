package sanmon

import "strings"

// mutationMethods are HTTP methods that modify state.
var mutationMethods = map[string]bool{
	"post": true, "put": true, "patch": true, "delete": true,
}

func validateAPI(a *Action, p *APIPolicy) []Violation {
	var violations []Violation

	// Require Authorization header for mutation methods
	if p.RequireAuthForMutation && mutationMethods[a.ActionType] {
		headers := getParamMap(a.Parameters, "headers")
		hasAuth := false
		for k := range headers {
			if strings.EqualFold(k, "authorization") {
				hasAuth = true
				break
			}
		}
		if !hasAuth {
			violations = append(violations, Violation{
				Rule:     "mutation_requires_auth",
				Message:  "mutation method " + a.ActionType + " requires Authorization header",
				Path:     "parameters.headers",
				Severity: SeverityError,
			})
		}
	}

	return violations
}

func getParamMap(params map[string]interface{}, key string) map[string]interface{} {
	v, ok := params[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return m
}
