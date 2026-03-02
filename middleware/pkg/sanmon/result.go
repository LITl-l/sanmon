package sanmon

import "fmt"

// Severity represents the severity of a policy violation.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Violation represents a single policy violation.
type Violation struct {
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Path     string   `json:"path"`
	Severity Severity `json:"severity"`
}

func (v Violation) String() string {
	return fmt.Sprintf("[%s] %s: %s (at %s)", v.Severity, v.Rule, v.Message, v.Path)
}

// ValidationResult is the outcome of validating an action against policies.
type ValidationResult struct {
	Pass       bool        `json:"pass"`
	Violations []Violation `json:"violations"`
	LatencyUs  int64       `json:"latency_us,omitempty"`
}

func pass() ValidationResult {
	return ValidationResult{Pass: true, Violations: []Violation{}}
}

func fail(violations ...Violation) ValidationResult {
	return ValidationResult{Pass: false, Violations: violations}
}
