// Command sanmon is the CLI for the sanmon (三門) formal verification engine.
//
// Usage:
//
//	sanmon validate <action.json>          Validate an action against policies
//	sanmon validate --dir <directory>       Validate all JSON files in directory
//	sanmon policy                           Show current policy configuration
//	sanmon schema                           Export JSON Schema for constrained decoding
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanmon/middleware/pkg/sanmon"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		runValidate(os.Args[2:])
	case "policy":
		runPolicy(os.Args[2:])
	case "schema":
		runSchema(os.Args[2:])
	case "version":
		fmt.Println("sanmon", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`sanmon (三門) — Formal Verification for AI Agent Actions

Usage:
  sanmon validate <action.json>        Validate a single action file
  sanmon validate --dir <directory>     Validate all JSON files in a directory
  sanmon policy                        Show current policy configuration
  sanmon policy --file <policy.json>   Load policy from file
  sanmon schema                        Export JSON Schema for all domains
  sanmon schema --domain <domain>      Export JSON Schema for a specific domain
  sanmon version                       Show version
  sanmon help                          Show this help

Examples:
  sanmon validate testdata/valid/browser_navigate.json
  sanmon validate --dir testdata/invalid/
  sanmon schema --domain browser`)
}

func runValidate(args []string) {
	var policyPath string
	var files []string
	isDir := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--policy":
			i++
			if i < len(args) {
				policyPath = args[i]
			}
		case "--dir":
			i++
			if i < len(args) {
				isDir = true
				matches, err := filepath.Glob(filepath.Join(args[i], "*.json"))
				if err != nil {
					fatalf("glob: %v", err)
				}
				files = append(files, matches...)
			}
		default:
			files = append(files, args[i])
		}
	}

	if len(files) == 0 {
		fatalf("no files to validate. Usage: sanmon validate <action.json>")
	}

	// Load policy
	var policy *sanmon.Policy
	if policyPath != "" {
		var err error
		policy, err = sanmon.LoadPolicy(policyPath)
		if err != nil {
			fatalf("load policy: %v", err)
		}
	} else {
		policy = sanmon.DefaultPolicy()
	}

	engine := sanmon.NewEngine(policy)

	passed, failed := 0, 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s — read error: %v\n", filepath.Base(f), err)
			failed++
			continue
		}
		result, err := engine.ValidateJSON(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s — parse error: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		if result.Pass {
			fmt.Printf("  ✓ %s — PASS (%dμs)\n", filepath.Base(f), result.LatencyUs)
			passed++
		} else {
			fmt.Printf("  ✗ %s — FAIL (%dμs)\n", filepath.Base(f), result.LatencyUs)
			for _, v := range result.Violations {
				fmt.Printf("    [%s] %s: %s\n", v.Severity, v.Rule, v.Message)
			}
			failed++
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed, %d total\n", passed, failed, passed+failed)

	if isDir {
		// In batch mode, exit 0 even if some fail (for demo purposes)
		return
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func runPolicy(args []string) {
	var policyPath string
	for i := 0; i < len(args); i++ {
		if args[i] == "--file" {
			i++
			if i < len(args) {
				policyPath = args[i]
			}
		}
	}

	var policy *sanmon.Policy
	if policyPath != "" {
		var err error
		policy, err = sanmon.LoadPolicy(policyPath)
		if err != nil {
			fatalf("load policy: %v", err)
		}
	} else {
		policy = sanmon.DefaultPolicy()
	}

	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		fatalf("marshal: %v", err)
	}
	fmt.Println(string(data))
}

func runSchema(args []string) {
	domain := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--domain" {
			i++
			if i < len(args) {
				domain = args[i]
			}
		}
	}

	schemas := generateSchemas()
	if domain != "" {
		s, ok := schemas[domain]
		if !ok {
			fatalf("unknown domain: %s (valid: %s)", domain, strings.Join(domainNames(), ", "))
		}
		printJSON(s)
		return
	}

	// Print all schemas
	printJSON(schemas)
}

func domainNames() []string {
	return []string{"browser", "api", "database", "iac", "approval"}
}

func generateSchemas() map[string]interface{} {
	schemas := map[string]interface{}{}

	// Base action schema
	baseProps := map[string]interface{}{
		"action_type": map[string]interface{}{"type": "string"},
		"target":      map[string]interface{}{"type": "string", "minLength": 1},
		"parameters":  map[string]interface{}{"type": "object"},
		"context": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"authenticated": map[string]interface{}{"type": "boolean"},
				"session_id":    map[string]interface{}{"type": "string", "minLength": 1},
				"domain":        map[string]interface{}{"type": "string", "enum": domainNames()},
			},
			"required": []string{"authenticated", "session_id", "domain"},
		},
		"metadata": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"timestamp":  map[string]interface{}{"type": "string", "format": "date-time"},
				"agent_id":   map[string]interface{}{"type": "string", "minLength": 1},
				"request_id": map[string]interface{}{"type": "string", "minLength": 1},
			},
			"required": []string{"timestamp", "agent_id", "request_id"},
		},
	}
	baseRequired := []string{"action_type", "target", "parameters", "context", "metadata"}

	// Browser
	schemas["browser"] = map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://sanmon.dev/schemas/browser-action.json",
		"title":       "Browser Action",
		"description": "Schema for browser automation agent actions (Playwright, Browser Use)",
		"type":        "object",
		"properties": mergeProps(baseProps, map[string]interface{}{
			"action_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"navigate", "click", "fill", "select", "scroll", "wait", "screenshot"},
			},
		}),
		"required": baseRequired,
	}

	// API
	schemas["api"] = map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://sanmon.dev/schemas/api-action.json",
		"title":       "API Action",
		"description": "Schema for API/MCP agent actions",
		"type":        "object",
		"properties": mergeProps(baseProps, map[string]interface{}{
			"action_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"get", "post", "put", "patch", "delete"},
			},
		}),
		"required": baseRequired,
	}

	// Database
	schemas["database"] = map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://sanmon.dev/schemas/database-action.json",
		"title":       "Database Action",
		"description": "Schema for SQL/database agent actions",
		"type":        "object",
		"properties": mergeProps(baseProps, map[string]interface{}{
			"action_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"select", "insert", "update", "delete", "create_table", "drop_table"},
			},
		}),
		"required": baseRequired,
	}

	// IaC
	schemas["iac"] = map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://sanmon.dev/schemas/iac-action.json",
		"title":       "IaC Action",
		"description": "Schema for infrastructure-as-code agent actions",
		"type":        "object",
		"properties": mergeProps(baseProps, map[string]interface{}{
			"action_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"create", "modify", "destroy", "plan", "apply"},
			},
		}),
		"required": baseRequired,
	}

	// Approval
	schemas["approval"] = map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://sanmon.dev/schemas/approval-action.json",
		"title":       "Approval Action",
		"description": "Schema for document approval workflow actions",
		"type":        "object",
		"properties": mergeProps(baseProps, map[string]interface{}{
			"action_type": map[string]interface{}{
				"type": "string",
				"enum": []string{"approve", "reject"},
			},
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"document": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id":             map[string]interface{}{"type": "string", "minLength": 1},
							"title":          map[string]interface{}{"type": "string"},
							"amount":         map[string]interface{}{"type": "number"},
							"department":     map[string]interface{}{"type": "string"},
							"category":       map[string]interface{}{"type": "string"},
							"applicant":      map[string]interface{}{"type": "string"},
							"submitted_date": map[string]interface{}{"type": "string"},
						},
						"required": []string{"id", "amount"},
					},
					"reason": map[string]interface{}{"type": "string"},
				},
				"required": []string{"document"},
			},
		}),
		"required": baseRequired,
	}

	return schemas
}

func mergeProps(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatalf("marshal: %v", err)
	}
	fmt.Println(string(data))
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "sanmon: "+format+"\n", args...)
	os.Exit(1)
}
