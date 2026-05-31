package sanmon

import "testing"

func TestDefaultAgentPolicyIsPermissive(t *testing.T) {
	p := DefaultPolicy()
	if len(p.Agent.DenyCommandRules) != 0 {
		t.Errorf("default agent policy must be permissive (no deny rules), got %d", len(p.Agent.DenyCommandRules))
	}
}

func TestStarterAgentPolicyHasDenylist(t *testing.T) {
	a := StarterAgentPolicy()
	if len(a.DenyCommandRules) == 0 {
		t.Error("starter agent policy must populate DenyCommandRules")
	}
	if len(a.SecretFilePatterns) == 0 {
		t.Error("starter agent policy must populate SecretFilePatterns")
	}
	if len(a.ExternalSinkCommands) == 0 {
		t.Error("starter agent policy must populate ExternalSinkCommands")
	}
}
