package sanmon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidGoldenFiles(t *testing.T) {
	engine := NewEngine(DefaultPolicy())
	files, err := filepath.Glob("../../../testdata/valid/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no valid golden files found")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			result, err := engine.ValidateJSON(data)
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if !result.Pass {
				for _, v := range result.Violations {
					t.Errorf("unexpected violation: %s", v)
				}
			}
			if result.LatencyUs < 0 {
				t.Error("expected non-negative latency")
			}
		})
	}
}

func TestInvalidGoldenFiles(t *testing.T) {
	engine := NewEngine(DefaultPolicy())
	files, err := filepath.Glob("../../../testdata/invalid/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no invalid golden files found")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			result, err := engine.ValidateJSON(data)
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if result.Pass {
				t.Error("expected validation to fail, but it passed")
			}
			if len(result.Violations) == 0 {
				t.Error("expected at least one violation")
			}
			t.Logf("violations: %v", result.Violations)
		})
	}
}

func TestStructuralValidation(t *testing.T) {
	engine := NewEngine(DefaultPolicy())

	tests := []struct {
		name       string
		action     Action
		wantPass   bool
		wantRules  []string
	}{
		{
			name: "empty target",
			action: Action{
				ActionType: "navigate",
				Target:     "",
				Parameters: map[string]interface{}{},
				Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "browser"},
				Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
			},
			wantPass:  false,
			wantRules: []string{"non_empty_target"},
		},
		{
			name: "invalid domain",
			action: Action{
				ActionType: "navigate",
				Target:     "https://example.com",
				Parameters: map[string]interface{}{},
				Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "invalid"},
				Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
			},
			wantPass:  false,
			wantRules: []string{"valid_domain"},
		},
		{
			name: "valid browser navigate",
			action: Action{
				ActionType: "navigate",
				Target:     "https://example.com",
				Parameters: map[string]interface{}{"url": "https://example.com"},
				Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "browser"},
				Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.Validate(&tt.action)
			if result.Pass != tt.wantPass {
				t.Errorf("pass = %v, want %v; violations: %v", result.Pass, tt.wantPass, result.Violations)
			}
			if tt.wantRules != nil {
				for _, want := range tt.wantRules {
					found := false
					for _, v := range result.Violations {
						if v.Rule == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected violation rule %q not found in %v", want, result.Violations)
					}
				}
			}
		})
	}
}

func TestBrowserPolicy(t *testing.T) {
	engine := NewEngine(DefaultPolicy())

	t.Run("dangerous scheme blocked", func(t *testing.T) {
		a := &Action{
			ActionType: "navigate",
			Target:     "javascript:alert(1)",
			Parameters: map[string]interface{}{"url": "javascript:alert(1)"},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "browser"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for dangerous scheme")
		}
		assertHasViolation(t, result, "dangerous_scheme")
	})

	t.Run("forbidden selector blocked", func(t *testing.T) {
		a := &Action{
			ActionType: "click",
			Target:     "[data-testid='delete-all']",
			Parameters: map[string]interface{}{"selector": "[data-testid='delete-all']"},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "browser"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for forbidden selector")
		}
		assertHasViolation(t, result, "forbidden_selector")
	})
}

func TestDatabasePolicy(t *testing.T) {
	engine := NewEngine(DefaultPolicy())

	t.Run("drop table forbidden", func(t *testing.T) {
		a := &Action{
			ActionType: "drop_table",
			Target:     "users",
			Parameters: map[string]interface{}{},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "database"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for drop table")
		}
		assertHasViolation(t, result, "drop_table_forbidden")
	})

	t.Run("delete without where", func(t *testing.T) {
		a := &Action{
			ActionType: "delete",
			Target:     "users",
			Parameters: map[string]interface{}{},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "database"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for delete without where")
		}
		assertHasViolation(t, result, "mutation_requires_where")
	})
}

func TestIaCPolicy(t *testing.T) {
	engine := NewEngine(DefaultPolicy())

	t.Run("destroy forbidden", func(t *testing.T) {
		a := &Action{
			ActionType: "destroy",
			Target:     "aws_s3_bucket.prod",
			Parameters: map[string]interface{}{"resource_type": "aws_s3_bucket"},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "iac"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for destroy")
		}
		assertHasViolation(t, result, "destroy_forbidden")
	})

	t.Run("missing required tags", func(t *testing.T) {
		a := &Action{
			ActionType: "create",
			Target:     "aws_s3_bucket.x",
			Parameters: map[string]interface{}{"resource_type": "aws_s3_bucket"},
			Context:    ActionContext{Authenticated: true, SessionID: "s1", Domain: "iac"},
			Metadata:   ActionMetadata{Timestamp: "2026-01-01T00:00:00Z", AgentID: "a1", RequestID: "r1"},
		}
		result := engine.Validate(a)
		if result.Pass {
			t.Error("expected failure for missing tags")
		}
		assertHasViolation(t, result, "missing_required_tag")
	})
}

func assertHasViolation(t *testing.T, result ValidationResult, rule string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule {
			return
		}
	}
	t.Errorf("expected violation %q not found; violations: %v", rule, result.Violations)
}
