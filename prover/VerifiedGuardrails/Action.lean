/-!
# Action Model

Formal definitions of AI agent actions and state transitions.
Corresponds to the JSON Schema action types defined in `schema/`.
-/

namespace VerifiedGuardrails

/-- Domains that actions can target -/
inductive Domain where
  | browser
  | api
  | database
  | iac
  deriving DecidableEq, Repr

/-- Browser action types -/
inductive BrowserAction where
  | navigate | click | fill | select | scroll | wait | screenshot
  deriving DecidableEq, Repr

/-- API action types -/
inductive ApiAction where
  | get | post | put | patch | delete
  deriving DecidableEq, Repr

/-- Database action types -/
inductive DatabaseAction where
  | select | insert | update | delete | createTable | dropTable
  deriving DecidableEq, Repr

/-- IaC action types -/
inductive IacAction where
  | create | modify | destroy | plan | apply
  deriving DecidableEq, Repr

/-- Unified action type across all domains -/
inductive ActionType where
  | browser (a : BrowserAction)
  | api     (a : ApiAction)
  | database (a : DatabaseAction)
  | iac     (a : IacAction)
  deriving DecidableEq, Repr

/-- Abstract state of the system -/
structure State where
  domain        : Domain
  authenticated : Bool
  resources     : List String
  deriving DecidableEq, Repr

/-- An action with its target and type -/
structure Action where
  actionType : ActionType
  target     : String
  deriving DecidableEq, Repr

/-- State transition function -/
def step (s : State) (_a : Action) : State :=
  -- Placeholder: real transitions defined per domain
  s

end VerifiedGuardrails
