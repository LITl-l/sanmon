// Package sanmon provides the core validation engine for the sanmon (三門) formal
// verification stack. It validates AI agent actions against configurable policies
// through a three-gate architecture: structural validation, policy enforcement,
// and formal proof verification.
package sanmon

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Engine is the main validation engine that orchestrates the three gates.
type Engine struct {
	mu     sync.RWMutex
	policy *Policy
}

// NewEngine creates a new validation engine with the given policy.
func NewEngine(policy *Policy) *Engine {
	if policy == nil {
		policy = DefaultPolicy()
	}
	return &Engine{policy: policy}
}

// ReloadPolicy atomically replaces the current policy.
func (e *Engine) ReloadPolicy(p *Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = p
}

// Policy returns the current policy (for inspection/export).
func (e *Engine) Policy() *Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.policy
}

// Validate validates an action against the loaded policies.
// It runs Gate 1 (structural validation) and Gate 2 (policy enforcement).
func (e *Engine) Validate(a *Action) ValidationResult {
	start := time.Now()

	e.mu.RLock()
	policy := e.policy
	e.mu.RUnlock()

	var allViolations []Violation

	// ── Gate 1: Structural validation ──
	structViolations := validateStructure(a)
	allViolations = append(allViolations, structViolations...)

	// If structural validation fails, skip policy checks
	if len(structViolations) > 0 {
		result := decide(allViolations)
		result.LatencyUs = time.Since(start).Microseconds()
		return result
	}

	// ── Gate 2: Domain policy validation ──
	policyViolations := e.validatePolicy(a, policy)
	allViolations = append(allViolations, policyViolations...)

	result := decide(allViolations)
	result.LatencyUs = time.Since(start).Microseconds()
	return result
}

// decide combines the violations through the verified combinator
// (decideViolations) and builds the result: the action passes only when the
// combined decision is Allow. Routing pass/fail through combineAll makes the
// Lean-proved "deny dominates" property govern the actual decision path.
func decide(violations []Violation) ValidationResult {
	if decideViolations(violations) == Allow {
		return pass()
	}
	return fail(violations...)
}

// ValidateJSON validates a JSON-encoded action.
func (e *Engine) ValidateJSON(data []byte) (ValidationResult, error) {
	var a Action
	if err := json.Unmarshal(data, &a); err != nil {
		return ValidationResult{}, fmt.Errorf("invalid action JSON: %w", err)
	}
	return e.Validate(&a), nil
}

func (e *Engine) validatePolicy(a *Action, p *Policy) []Violation {
	switch a.Context.Domain {
	case "browser":
		return validateBrowser(a, &p.Browser)
	case "api":
		return validateAPI(a, &p.API)
	case "database":
		return validateDatabase(a, &p.Database)
	case "iac":
		return validateIaC(a, &p.IaC)
	case "approval":
		return validateApproval(a, &p.Approval)
	case "agent":
		return validateAgent(a, &p.Agent)
	default:
		return []Violation{{
			Rule: "unknown_domain", Message: "unknown domain: " + a.Context.Domain,
			Path: "context.domain", Severity: SeverityError,
		}}
	}
}
