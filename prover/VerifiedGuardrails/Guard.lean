/-!
# Guard Decision Combination

Models how the agent guard combines per-rule verdicts into one decision and
proves the load-bearing safety property: a `deny` always wins over `allow`.
This mirrors the Go engine, where any error-severity violation makes the whole
result fail (deny), regardless of other passing checks.

Pure Lean 4 core — no Mathlib (this project has no Mathlib dependency), so we
use `cases`, not `rcases`.
-/

namespace VerifiedGuardrails

/-- A guard decision for a single tool call. -/
inductive Decision where
  | allow
  | ask
  | deny
  deriving DecidableEq, Repr

/-- Combine two decisions: the more restrictive one wins (deny > ask > allow). -/
def Decision.combine : Decision → Decision → Decision
  | .deny, _       => .deny
  | _, .deny       => .deny
  | .ask, _        => .ask
  | _, .ask        => .ask
  | .allow, .allow => .allow

/-- Fold a list of decisions, starting from `allow`. -/
def combineAll : List Decision → Decision
  | []      => .allow
  | d :: ds => d.combine (combineAll ds)

/-- Combining any decision with a `deny` on the right yields `deny`. -/
theorem combine_deny_right (d : Decision) : d.combine Decision.deny = Decision.deny := by
  cases d <;> rfl

/-- If any verdict in the list is `deny`, the combined decision is `deny`.
    No number of `allow`/`ask` verdicts can override a single `deny`.

    Stated as `∀ … → …` and introduced after `induction` so the membership
    hypothesis does not depend on the variable being inducted on. -/
theorem deny_dominates :
    ∀ (ds : List Decision), Decision.deny ∈ ds → combineAll ds = Decision.deny := by
  intro ds
  induction ds with
  | nil => intro h; cases h
  | cons d rest ih =>
    intro h
    cases List.mem_cons.mp h with
    | inl hd => subst hd; rfl
    | inr htl =>
      have hrest : combineAll rest = Decision.deny := ih htl
      show Decision.combine d (combineAll rest) = Decision.deny
      rw [hrest]
      exact combine_deny_right d

end VerifiedGuardrails
