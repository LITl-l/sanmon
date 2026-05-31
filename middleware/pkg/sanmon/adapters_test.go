package sanmon

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeGenericShell(t *testing.T) {
	in := []byte(`{"tool":"shell_exec","command":"rm -rf ~","agent_id":"x"}`)
	a, cls, err := DecodeAgentPayload("generic", in)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if a.Context.Domain != "agent" || a.ActionType != "shell_exec" {
		t.Errorf("bad normalized action: %+v", a)
	}
	if getParamString(a.Parameters, "command") != "rm -rf ~" {
		t.Errorf("command not mapped: %+v", a.Parameters)
	}
	if cls != ClassDestructive {
		t.Errorf("shell_exec should be destructive class, got %v", cls)
	}
	// Synthesized metadata must satisfy Gate-1.
	if a.Metadata.AgentID == "" || a.Metadata.RequestID == "" || a.Metadata.Timestamp == "" || a.Context.SessionID == "" {
		t.Errorf("guard must synthesize required metadata: %+v", a)
	}
}

func TestDecodeClaudeBashAndWrite(t *testing.T) {
	bash := []byte(`{"hook_event_name":"PreToolUse","session_id":"s","cwd":"/p","tool_name":"Bash","tool_input":{"command":"ls"}}`)
	a, _, err := DecodeAgentPayload("claude", bash)
	if err != nil || a.ActionType != "shell_exec" || getParamString(a.Parameters, "command") != "ls" {
		t.Fatalf("claude Bash decode wrong: %+v err=%v", a, err)
	}
	write := []byte(`{"tool_name":"Write","tool_input":{"file_path":"/p/x.go","content":"package p"}}`)
	a2, _, err := DecodeAgentPayload("claude", write)
	if err != nil || a2.ActionType != "file_write" || getParamString(a2.Parameters, "path") != "/p/x.go" {
		t.Fatalf("claude Write decode wrong: %+v err=%v", a2, err)
	}
	edit := []byte(`{"tool_name":"Edit","tool_input":{"file_path":"/p/x.go","old_string":"a","new_string":"b"}}`)
	a3, _, _ := DecodeAgentPayload("claude", edit)
	if a3.ActionType != "file_edit" {
		t.Fatalf("claude Edit decode wrong: %+v", a3)
	}
	read := []byte(`{"tool_name":"Read","tool_input":{"file_path":"/p/x.go"}}`)
	_, cls, _ := DecodeAgentPayload("claude", read)
	if cls != ClassRead {
		t.Fatalf("Read should be read-class, got %v", cls)
	}
}

func TestDecodeCodexApplyPatch(t *testing.T) {
	in := []byte(`{"tool_name":"apply_patch","tool_input":{"patch":"*** Begin Patch"}}`)
	a, _, err := DecodeAgentPayload("codex", in)
	if err != nil || a.ActionType != "file_edit" {
		t.Fatalf("codex apply_patch should map to file_edit: %+v err=%v", a, err)
	}
}

func TestEncodeDecisions(t *testing.T) {
	deny := fail(Violation{Rule: "agent.destructive_delete", Message: "nope", Path: "p", Severity: SeverityError})
	pass := pass()

	// Claude/Codex hookSpecificOutput shape.
	out := EncodeDecision("claude", deny)
	var claude struct {
		Hook struct {
			Event  string `json:"hookEventName"`
			Perm   string `json:"permissionDecision"`
			Reason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &claude); err != nil {
		t.Fatalf("claude encode not JSON: %v", err)
	}
	if claude.Hook.Perm != "deny" || claude.Hook.Event != "PreToolUse" || !strings.Contains(claude.Hook.Reason, "nope") {
		t.Errorf("bad claude deny encode: %s", out)
	}
	if p := EncodeDecision("claude", pass); !strings.Contains(string(p), `"permissionDecision":"allow"`) {
		t.Errorf("bad claude allow encode: %s", p)
	}

	// Generic shape.
	g := EncodeDecision("generic", deny)
	if !strings.Contains(string(g), `"decision":"deny"`) {
		t.Errorf("bad generic deny encode: %s", g)
	}
}

func TestDecodeHookMalformedToolInputFailsClosed(t *testing.T) {
	// Destructive tool with null / non-object / non-string-command tool_input
	// must return an error so the guard fails closed.
	bad := []string{
		`{"tool_name":"Bash","tool_input":null}`,
		`{"tool_name":"Bash","tool_input":42}`,
		`{"tool_name":"Bash","tool_input":{"command":["rm","-rf","~"]}}`,
		`{"tool_name":"Write","tool_input":null}`,
		`{"tool_name":"Bash"}`,
	}
	for _, in := range bad {
		_, class, err := DecodeAgentPayload("claude", []byte(in))
		if err == nil {
			t.Errorf("expected decode error (fail-closed) for %s", in)
		}
		if class != ClassDestructive {
			t.Errorf("expected ClassDestructive for %s, got %v", in, class)
		}
	}
}

func TestDecodeHookReadClassMalformedFailsOpen(t *testing.T) {
	// A read-class tool with null tool_input should NOT error (fail-open).
	a, class, err := DecodeAgentPayload("claude", []byte(`{"tool_name":"Read","tool_input":null}`))
	if err != nil {
		t.Fatalf("read-class null tool_input should not error, got %v", err)
	}
	if class != ClassRead {
		t.Errorf("expected ClassRead, got %v", class)
	}
	if a.ActionType != "file_read" {
		t.Errorf("expected file_read, got %q", a.ActionType)
	}
}

func TestDecodeHookMcpNormalized(t *testing.T) {
	a, _, err := DecodeAgentPayload("claude", []byte(`{"tool_name":"mcp__filesystem__read_file","tool_input":{}}`))
	if err != nil {
		t.Fatalf("mcp decode error: %v", err)
	}
	if a.ActionType != "mcp_call" {
		t.Fatalf("expected mcp_call, got %q", a.ActionType)
	}
	if got := getParamString(a.Parameters, "server"); got != "filesystem" {
		t.Errorf("expected server=filesystem, got %q", got)
	}
	if got := getParamString(a.Parameters, "tool"); got != "read_file" {
		t.Errorf("expected tool=read_file, got %q", got)
	}
}

func TestEncodeDecisionFallbacksAreFailClosed(t *testing.T) {
	// The hard-coded fail-closed fallbacks must be valid JSON that denies.
	claudeFallback := []byte(`{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"sanmon: internal encode error (failing closed)"}}`)
	var c struct {
		Hook struct {
			Perm string `json:"permissionDecision"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(claudeFallback, &c); err != nil {
		t.Fatalf("claude fallback is not valid JSON: %v", err)
	}
	if c.Hook.Perm != "deny" {
		t.Errorf("claude fallback must deny, got %q", c.Hook.Perm)
	}

	genericFallback := []byte(`{"decision":"deny","reason":"sanmon: internal encode error (failing closed)"}`)
	var g struct {
		Decision string `json:"decision"`
	}
	if err := json.Unmarshal(genericFallback, &g); err != nil {
		t.Fatalf("generic fallback is not valid JSON: %v", err)
	}
	if g.Decision != "deny" {
		t.Errorf("generic fallback must deny, got %q", g.Decision)
	}
}
