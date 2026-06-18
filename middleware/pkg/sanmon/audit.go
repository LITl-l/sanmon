package sanmon

import (
	"encoding/json"
	"io"
)

// AuditRecord is a structured, append-only log entry for one guard decision.
// It is the security audit trail: what was proposed, what sanmon decided, and
// why. Records are emitted as one JSON object per line (JSON Lines).
type AuditRecord struct {
	Time       string   `json:"time,omitempty"` // RFC3339; stamped by the caller
	Agent      string   `json:"agent"`
	Decision   string   `json:"decision"`              // "allow" | "deny"
	Mode       string   `json:"mode"`                  // "evaluated" | "fail_closed" | "fail_open"
	ActionType string   `json:"action_type,omitempty"`
	Target     string   `json:"target,omitempty"`
	Class      string   `json:"class,omitempty"` // "read" | "destructive"
	Rules      []string `json:"rules,omitempty"` // violated rule names, for denies
	LatencyUs  int64    `json:"latency_us,omitempty"`
	Reason     string   `json:"reason,omitempty"` // populated on fail modes
}

func decisionString(pass bool) string {
	if pass {
		return "allow"
	}
	return "deny"
}

func classString(c ActionClass) string {
	if c == ClassRead {
		return "read"
	}
	return "destructive"
}

// AuditForDecision builds a record for a normally-evaluated decision.
func AuditForDecision(agent string, a *Action, r ValidationResult) AuditRecord {
	rec := AuditRecord{
		Agent:     agent,
		Decision:  decisionString(r.Pass),
		Mode:      "evaluated",
		LatencyUs: r.LatencyUs,
	}
	if a != nil {
		rec.ActionType = a.ActionType
		rec.Target = a.Target
	}
	for _, v := range r.Violations {
		rec.Rules = append(rec.Rules, v.Rule)
	}
	return rec
}

// AuditFailClosed builds a record for an internal-error path, recording the
// asymmetric failure decision: destructive actions are denied (fail_closed),
// read-class actions are allowed (fail_open).
func AuditFailClosed(agent string, class ActionClass, reason string) AuditRecord {
	rec := AuditRecord{
		Agent:  agent,
		Class:  classString(class),
		Reason: reason,
	}
	if class == ClassRead {
		rec.Decision = "allow"
		rec.Mode = "fail_open"
	} else {
		rec.Decision = "deny"
		rec.Mode = "fail_closed"
	}
	return rec
}

// WriteAudit emits rec as a single JSON line to w. Errors are intentionally
// swallowed: audit logging must never block or crash the decision path.
func WriteAudit(w io.Writer, rec AuditRecord) {
	if w == nil {
		return
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}
