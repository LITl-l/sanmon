package sanmon

import "testing"

// These tests mirror the Lean theorems in prover/VerifiedGuardrails/Guard.lean
// (combine_deny_right, deny_dominates) so the proven property is executable in
// the Go implementation, not only in the model.

func TestCombineDenyRight(t *testing.T) {
	for _, d := range []Decision{Allow, Ask, Deny} {
		if got := d.combine(Deny); got != Deny {
			t.Errorf("%v.combine(Deny) = %v, want Deny", d, got)
		}
	}
}

// deny_dominates: any list containing a Deny combines to Deny — no number of
// allow/ask verdicts can override a single deny.
func TestDenyDominates(t *testing.T) {
	cases := [][]Decision{
		{Deny},
		{Allow, Deny},
		{Allow, Ask, Allow, Deny, Allow},
		{Ask, Ask, Deny},
	}
	for _, ds := range cases {
		if got := combineAll(ds); got != Deny {
			t.Errorf("combineAll(%v) = %v, want Deny", ds, got)
		}
	}
}

func TestAskDominatesAllow(t *testing.T) {
	if got := combineAll([]Decision{Allow, Ask, Allow}); got != Ask {
		t.Errorf("combineAll with an Ask and no Deny = %v, want Ask", got)
	}
}

func TestEmptyAndAllAllow(t *testing.T) {
	if got := combineAll(nil); got != Allow {
		t.Errorf("combineAll(nil) = %v, want Allow", got)
	}
	if got := combineAll([]Decision{Allow, Allow}); got != Allow {
		t.Errorf("combineAll(all allow) = %v, want Allow", got)
	}
}

// TestEngineUpholdsDenyDominates is the model<->implementation bridge: the
// engine's pass/fail outcome must equal the verified combinator applied to the
// violations it produced. A command tripping several rules must deny, and the
// deny must survive regardless of the passing checks around it.
func TestEngineUpholdsDenyDominates(t *testing.T) {
	p := DefaultPolicy()
	p.Agent = StarterAgentPolicy()
	eng := NewEngine(p)

	multi := agentAction("shell_exec", "x", map[string]interface{}{
		"command": "rm -rf / && git push --force origin main",
	})
	res := eng.Validate(multi)
	if len(res.Violations) < 2 {
		t.Fatalf("expected multiple violations to exercise combination, got %v", res.Violations)
	}
	if decideViolations(res.Violations) != Deny {
		t.Errorf("combinator over engine violations = %v, want Deny", decideViolations(res.Violations))
	}
	if res.Pass {
		t.Errorf("engine allowed an action with violations %v", res.Violations)
	}

	ok := agentAction("shell_exec", "ls -la", map[string]interface{}{"command": "ls -la"})
	okRes := eng.Validate(ok)
	if (decideViolations(okRes.Violations) == Allow) != okRes.Pass {
		t.Errorf("engine Pass=%v disagrees with combinator over %v", okRes.Pass, okRes.Violations)
	}
}
