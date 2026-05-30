package sanmon

import "testing"

func agentAction(actionType, target string, params map[string]interface{}) *Action {
	if params == nil {
		params = map[string]interface{}{}
	}
	return &Action{
		ActionType: actionType,
		Target:     target,
		Parameters: params,
		Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "agent"},
		Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
	}
}

func TestAgentBenignPasses(t *testing.T) {
	p := StarterAgentPolicy()
	cases := []*Action{
		agentAction("shell_exec", "ls -la", map[string]interface{}{"command": "ls -la"}),
		agentAction("shell_exec", "git status", map[string]interface{}{"command": "git status"}),
		agentAction("file_read", "main.go", map[string]interface{}{"path": "main.go"}),
		agentAction("mcp_call", "memory.get", map[string]interface{}{"server": "memory", "tool": "get"}),
	}
	for _, a := range cases {
		v := validateAgent(a, &p)
		if len(v) != 0 {
			t.Errorf("expected %s %q to pass, got %v", a.ActionType, a.Target, v)
		}
	}
}

func TestNormalizeCommand(t *testing.T) {
	got := normalizeCommand("  rm   -rf    ~/  ")
	want := "rm -rf ~/"
	if got != want {
		t.Errorf("normalizeCommand = %q, want %q", got, want)
	}
}
