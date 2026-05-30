package sanmon

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ActionClass classifies an action for fail-open/fail-closed handling.
type ActionClass int

const (
	ClassRead        ActionClass = iota // fail-OPEN on internal error
	ClassDestructive                    // fail-CLOSED on internal error
)

// classForActionType maps a normalized action_type to its failure class.
func classForActionType(actionType string) ActionClass {
	switch actionType {
	case "shell_exec", "file_write", "file_edit":
		return ClassDestructive
	default: // file_read, net_fetch, mcp_call, and unknown read-ish tools
		return ClassRead
	}
}

// genericPayload is sanmon's slim, agent-agnostic input contract.
type genericPayload struct {
	Tool    string `json:"tool"`
	Command string `json:"command"`
	Path    string `json:"path"`
	Content string `json:"content"`
	URL     string `json:"url"`
	Host    string `json:"host"`
	Server  string `json:"server"`
	MCPTool string `json:"mcp_tool"`
	AgentID string `json:"agent_id"`
	Session string `json:"session_id"`
	CWD     string `json:"cwd"`
}

// hookPayload is the Claude Code / Codex PreToolUse stdin shape (shared).
type hookPayload struct {
	HookEventName string          `json:"hook_event_name"`
	SessionID     string          `json:"session_id"`
	CWD           string          `json:"cwd"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

// DecodeAgentPayload maps an agent's native stdin payload onto a normalized
// agent-domain Action and returns its failure class. agent is one of
// "generic", "claude", "codex".
func DecodeAgentPayload(agent string, data []byte) (*Action, ActionClass, error) {
	switch agent {
	case "generic":
		return decodeGeneric(data)
	case "claude", "codex":
		return decodeHook(agent, data)
	default:
		return nil, ClassDestructive, fmt.Errorf("unknown agent: %q", agent)
	}
}

func decodeGeneric(data []byte) (*Action, ActionClass, error) {
	var g genericPayload
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, ClassDestructive, fmt.Errorf("invalid generic payload: %w", err)
	}
	if g.Tool == "" {
		return nil, ClassDestructive, fmt.Errorf("generic payload missing \"tool\"")
	}
	params := map[string]interface{}{}
	target := ""
	switch g.Tool {
	case "shell_exec":
		params["command"] = g.Command
		target = g.Command
	case "file_write":
		params["path"] = g.Path
		params["content"] = g.Content
		target = g.Path
	case "file_edit":
		params["path"] = g.Path
		target = g.Path
	case "file_read":
		params["path"] = g.Path
		target = g.Path
	case "net_fetch":
		params["url"] = g.URL
		params["host"] = g.Host
		target = g.URL
	case "mcp_call":
		params["server"] = g.Server
		params["tool"] = g.MCPTool
		target = g.Server + "." + g.MCPTool
	default:
		return nil, ClassDestructive, fmt.Errorf("unknown generic tool: %q", g.Tool)
	}
	return buildAction(g.Tool, target, params, g.AgentID, g.Session, "generic"), classForActionType(g.Tool), nil
}

func decodeHook(agent string, data []byte) (*Action, ActionClass, error) {
	var h hookPayload
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, ClassDestructive, fmt.Errorf("invalid %s payload: %w", agent, err)
	}
	var ti map[string]interface{}
	_ = json.Unmarshal(h.ToolInput, &ti) // best-effort; nil map is fine
	get := func(k string) string {
		if ti == nil {
			return ""
		}
		if s, ok := ti[k].(string); ok {
			return s
		}
		return ""
	}

	actionType, target := "", ""
	params := map[string]interface{}{}
	switch {
	case h.ToolName == "Bash":
		actionType, target = "shell_exec", get("command")
		params["command"] = get("command")
	case h.ToolName == "Write":
		actionType, target = "file_write", get("file_path")
		params["path"] = get("file_path")
		params["content"] = get("content")
	case h.ToolName == "Edit" || h.ToolName == "MultiEdit" || h.ToolName == "NotebookEdit" || h.ToolName == "apply_patch":
		actionType, target = "file_edit", get("file_path")
		params["path"] = get("file_path")
	case h.ToolName == "Read":
		actionType, target = "file_read", get("file_path")
		params["path"] = get("file_path")
	case h.ToolName == "WebFetch":
		actionType, target = "net_fetch", get("url")
		params["url"] = get("url")
		params["host"] = hostFromURL(get("url"))
	case strings.HasPrefix(h.ToolName, "mcp__"):
		actionType, target = "mcp_call", h.ToolName
		params["server"] = h.ToolName
	default:
		// Unknown tool: treat as read-class so we fail-open rather than block
		// benign tools we don't model yet.
		actionType, target = "file_read", h.ToolName
	}
	return buildAction(actionType, target, params, agent, h.SessionID, agent), classForActionType(actionType), nil
}

// buildAction assembles a Gate-1-valid Action, synthesizing required metadata
// the agent's hook payload does not provide.
func buildAction(actionType, target string, params map[string]interface{}, agentID, session, source string) *Action {
	if agentID == "" {
		agentID = "sanmon-guard:" + source
	}
	if session == "" {
		session = "sanmon-guard-session"
	}
	if target == "" {
		target = actionType // Gate-1 requires non-empty target
	}
	return &Action{
		ActionType: actionType,
		Target:     target,
		Parameters: params,
		Context:    ActionContext{Authenticated: true, SessionID: session, Domain: "agent"},
		Metadata: ActionMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			AgentID:   agentID,
			RequestID: fmt.Sprintf("guard-%d", time.Now().UnixNano()),
		},
	}
}

// EncodeDecision renders a ValidationResult into the agent's native decision.
func EncodeDecision(agent string, result ValidationResult) []byte {
	switch agent {
	case "claude", "codex":
		perm := "allow"
		reason := ""
		if !result.Pass {
			perm = "deny"
			reason = joinReasons(result.Violations)
		}
		out := map[string]interface{}{
			"hookSpecificOutput": map[string]interface{}{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       perm,
				"permissionDecisionReason": reason,
			},
		}
		b, _ := json.Marshal(out)
		return b
	default: // generic
		decision := "allow"
		if !result.Pass {
			decision = "deny"
		}
		out := map[string]interface{}{
			"decision":   decision,
			"reason":     joinReasons(result.Violations),
			"violations": result.Violations,
		}
		b, _ := json.Marshal(out)
		return b
	}
}

func joinReasons(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}
	parts := make([]string, 0, len(violations))
	for _, v := range violations {
		parts = append(parts, fmt.Sprintf("%s (%s)", v.Message, v.Rule))
	}
	return "sanmon: " + strings.Join(parts, "; ")
}
