package sanmon

import "fmt"

func validateDatabase(a *Action, p *DatabasePolicy) []Violation {
	var violations []Violation

	// DROP TABLE is forbidden by default
	if a.ActionType == "drop_table" && !p.AllowDropTable {
		violations = append(violations, Violation{
			Rule:     "drop_table_forbidden",
			Message:  "DROP TABLE is forbidden by policy",
			Path:     "action_type",
			Severity: SeverityError,
		})
	}

	// WHERE clause required for UPDATE and DELETE
	if p.RequireWhereForMutation && (a.ActionType == "update" || a.ActionType == "delete") {
		where := getParamString(a.Parameters, "where_clause")
		if where == "" {
			violations = append(violations, Violation{
				Rule:     "mutation_requires_where",
				Message:  fmt.Sprintf("%s requires a WHERE clause", a.ActionType),
				Path:     "parameters.where_clause",
				Severity: SeverityError,
			})
		}
	}

	// Check read-only tables
	if a.ActionType != "select" {
		for _, t := range p.ReadOnlyTables {
			if a.Target == t {
				violations = append(violations, Violation{
					Rule:     "read_only_table",
					Message:  fmt.Sprintf("table %q is read-only; only SELECT is allowed", t),
					Path:     "target",
					Severity: SeverityError,
				})
			}
		}
	}

	// Check no-delete tables
	if a.ActionType == "delete" {
		for _, t := range p.NoDeleteTables {
			if a.Target == t {
				violations = append(violations, Violation{
					Rule:     "no_delete_table",
					Message:  fmt.Sprintf("DELETE is forbidden on table %q", t),
					Path:     "target",
					Severity: SeverityError,
				})
			}
		}
	}

	// Check sensitive columns
	if len(p.SensitiveColumns) > 0 {
		columns := getParamStringSlice(a.Parameters, "columns")
		for _, col := range columns {
			for _, sc := range p.SensitiveColumns {
				if col == sc {
					violations = append(violations, Violation{
						Rule:     "sensitive_column_access",
						Message:  fmt.Sprintf("column %q is marked as sensitive", col),
						Path:     "parameters.columns",
						Severity: SeverityWarning,
					})
				}
			}
		}
	}

	return violations
}

func getParamStringSlice(params map[string]interface{}, key string) []string {
	v, ok := params[key]
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
