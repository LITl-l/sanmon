import Lake
open Lake DSL

package sanmon where
  leanOptions := #[
    ⟨`autoImplicit, false⟩
  ]

@[default_target]
lean_lib VerifiedGuardrails where
  srcDir := "."
  roots := #[`VerifiedGuardrails]
