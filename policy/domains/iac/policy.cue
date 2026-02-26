// IaC domain policy — constraints for infrastructure-as-code agents.

package iac

#IacAction: {
	action_type: "create" | "modify" | "destroy" | "plan" | "apply"
	target:      string & !=""  // resource ID or type
	parameters:  #IacParams
	context: {
		domain: "iac"
		...
	}
	...
}

#IacParams: {
	resource_type?: string
	properties?:    _
	tags?:          {[string]: string}
	...
}

// ── Policy rules ──

// Resource types that can be created or modified
#AllowedResourceTypes: [...string]

// Required tags on every resource
#RequiredTags: [...string]

policy: {
	allowed_resource_types: #AllowedResourceTypes | *[]
	required_tags:          #RequiredTags | *["owner", "environment", "project"]

	// Destroy is forbidden by default
	allow_destroy: bool | *false

	// Prevent opening 0.0.0.0/0 ingress
	block_open_ingress: bool | *true

	// plan is always allowed
	allow_plan: bool | *true
}
