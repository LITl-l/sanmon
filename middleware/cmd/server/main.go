// Command server runs the sanmon HTTP validation server.
//
// Endpoints:
//
//	POST /v1/validate     Validate an action against policies
//	POST /v1/reload       Reload policies from disk
//	GET  /v1/policy       Show current policy
//	GET  /v1/schema       Export JSON schemas (optional ?domain=browser)
//	GET  /healthz         Health check
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sanmon/middleware/pkg/sanmon"
)

func main() {
	addr := ":8080"
	policyPath := ""

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--addr":
			i++
			if i < len(os.Args) {
				addr = os.Args[i]
			}
		case "--policy":
			i++
			if i < len(os.Args) {
				policyPath = os.Args[i]
			}
		}
	}

	var policy *sanmon.Policy
	if policyPath != "" {
		var err error
		policy, err = sanmon.LoadPolicy(policyPath)
		if err != nil {
			log.Fatalf("load policy: %v", err)
		}
		log.Printf("loaded policy from %s", policyPath)
	} else {
		policy = sanmon.DefaultPolicy()
		log.Printf("using default policy")
	}

	engine := sanmon.NewEngine(policy)
	srv := &server{engine: engine, policyPath: policyPath}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/validate", srv.handleValidate)
	mux.HandleFunc("/v1/reload", srv.handleReload)
	mux.HandleFunc("/v1/policy", srv.handlePolicy)
	mux.HandleFunc("/v1/schema", srv.handleSchema)
	mux.HandleFunc("/healthz", srv.handleHealth)

	log.Printf("sanmon server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, logMiddleware(mux)))
}

type server struct {
	engine     *sanmon.Engine
	policyPath string
}

func (s *server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		Action json.RawMessage `json:"action_json"`
		Domain string          `json:"domain,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	result, err := s.engine.ValidateJSON(req.Action)
	if err != nil {
		httpError(w, http.StatusBadRequest, "invalid action: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	if s.policyPath == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "no policy file configured; using defaults",
		})
		return
	}

	policy, err := sanmon.LoadPolicy(s.policyPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	s.engine.ReloadPolicy(policy)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "policy reloaded from " + s.policyPath,
	})
}

func (s *server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	writeJSON(w, http.StatusOK, s.engine.Policy())
}

func (s *server) handleSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	schemas := map[string]interface{}{
		"browser":  browserSchema(),
		"api":      apiSchema(),
		"database": databaseSchema(),
		"iac":      iacSchema(),
		"approval": approvalSchema(),
	}

	domain := r.URL.Query().Get("domain")
	if domain != "" {
		s, ok := schemas[domain]
		if !ok {
			httpError(w, http.StatusBadRequest, "unknown domain: "+domain)
			return
		}
		writeJSON(w, http.StatusOK, s)
		return
	}

	writeJSON(w, http.StatusOK, schemas)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// Schema generation helpers

func baseProperties() map[string]interface{} {
	return map[string]interface{}{
		"target":     map[string]interface{}{"type": "string", "minLength": 1},
		"parameters": map[string]interface{}{"type": "object"},
		"context": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"authenticated": map[string]interface{}{"type": "boolean"},
				"session_id":    map[string]interface{}{"type": "string", "minLength": 1},
				"domain":        map[string]interface{}{"type": "string", "enum": []string{"browser", "api", "database", "iac"}},
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
}

func makeSchema(id, title, desc string, actionTypes []string) map[string]interface{} {
	props := baseProperties()
	props["action_type"] = map[string]interface{}{"type": "string", "enum": actionTypes}
	return map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         fmt.Sprintf("https://sanmon.dev/schemas/%s-action.json", id),
		"title":       title,
		"description": desc,
		"type":        "object",
		"properties":  props,
		"required":    []string{"action_type", "target", "parameters", "context", "metadata"},
	}
}

func browserSchema() map[string]interface{} {
	return makeSchema("browser", "Browser Action",
		"Schema for browser automation agent actions",
		[]string{"navigate", "click", "fill", "select", "scroll", "wait", "screenshot"})
}

func apiSchema() map[string]interface{} {
	return makeSchema("api", "API Action",
		"Schema for API/MCP agent actions",
		[]string{"get", "post", "put", "patch", "delete"})
}

func databaseSchema() map[string]interface{} {
	return makeSchema("database", "Database Action",
		"Schema for SQL/database agent actions",
		[]string{"select", "insert", "update", "delete", "create_table", "drop_table"})
}

func iacSchema() map[string]interface{} {
	return makeSchema("iac", "IaC Action",
		"Schema for infrastructure-as-code agent actions",
		[]string{"create", "modify", "destroy", "plan", "apply"})
}

func approvalSchema() map[string]interface{} {
	schema := makeSchema("approval", "Approval Action",
		"Schema for document approval workflow actions",
		[]string{"approve", "reject"})
	// Add document sub-schema to parameters
	props := schema["properties"].(map[string]interface{})
	props["parameters"] = map[string]interface{}{
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
	}
	return schema
}
