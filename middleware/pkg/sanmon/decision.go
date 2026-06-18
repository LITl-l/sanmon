package sanmon

// Decision is a guard verdict for a single check. It is the Go port of the
// model in prover/VerifiedGuardrails/Guard.lean, whose `deny_dominates`
// theorem proves that a single deny overrides any number of allow/ask
// verdicts. Keeping the implementation structurally identical to the proved
// model is what makes that proof load-bearing rather than decorative.
type Decision int

const (
	Allow Decision = iota
	Ask
	Deny
)

func (d Decision) String() string {
	switch d {
	case Deny:
		return "deny"
	case Ask:
		return "ask"
	default:
		return "allow"
	}
}

// combine returns the more restrictive of two decisions (deny > ask > allow).
// Mirrors Decision.combine in Guard.lean.
func (d Decision) combine(o Decision) Decision {
	if d == Deny || o == Deny {
		return Deny
	}
	if d == Ask || o == Ask {
		return Ask
	}
	return Allow
}

// combineAll folds a list of decisions from allow. Mirrors combineAll in
// Guard.lean: by the deny_dominates theorem, the result is Deny iff any element
// is Deny.
func combineAll(ds []Decision) Decision {
	out := Allow
	for _, d := range ds {
		out = d.combine(out)
	}
	return out
}

// violationDecision maps one violation to a decision: error-severity violations
// deny, warnings ask for review.
func violationDecision(v Violation) Decision {
	if v.Severity == SeverityWarning {
		return Ask
	}
	return Deny
}

// decideViolations combines all violations into a single decision via the
// verified combinator. The engine's pass/fail is derived from this, so the
// proven "deny dominates" property governs the real decision path.
func decideViolations(vs []Violation) Decision {
	ds := make([]Decision, len(vs))
	for i, v := range vs {
		ds[i] = violationDecision(v)
	}
	return combineAll(ds)
}
