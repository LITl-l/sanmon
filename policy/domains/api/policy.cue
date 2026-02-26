// API domain policy — constraints for MCP / function-calling agents.

package api

#ApiAction: {
	action_type: "get" | "post" | "put" | "patch" | "delete"
	target:      string & !=""  // endpoint URL
	parameters:  #ApiParams
	context: {
		domain: "api"
		...
	}
	...
}

#ApiParams: {
	headers?: {[string]: string}
	body?:    _
	query?:   {[string]: string}
	...
}

// ── Policy rules ──

// Allowed endpoints (whitelist)
#AllowedEndpoints: [...string]

// Allowed HTTP methods per endpoint pattern
#MethodRestrictions: {[string]: [...#HttpMethod]}
#HttpMethod: "get" | "post" | "put" | "patch" | "delete"

policy: {
	allowed_endpoints: #AllowedEndpoints | *[]

	// Default: only GET allowed unless overridden
	method_restrictions: #MethodRestrictions | *{}

	// Require Authorization header for non-GET
	require_auth_for_mutations: bool | *true
}
