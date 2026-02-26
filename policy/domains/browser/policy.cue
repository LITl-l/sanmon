// Browser domain policy — constraints for Playwright / browser automation agents.

package browser

import "net"

#BrowserAction: {
	action_type: "navigate" | "click" | "fill" | "select" | "scroll" | "wait" | "screenshot"
	target:      string & !=""
	parameters:  {[string]: _}
	context: {
		domain: "browser"
		...
	}
	...
}

// ── Policy rules ──

// Allowed URL patterns (override in your project config)
#AllowedURLPatterns: [...string]

// Forbidden CSS selectors (e.g., admin buttons, dangerous forms)
#ForbiddenSelectors: [...string]

// Input value constraints
#MaxInputLength: int | *1000

// Navigate action: target must be a valid FQDN
#NavigateAction: #BrowserAction & {
	action_type: "navigate"
	parameters: {
		url: string & net.FQDN
	}
}

// Fill action: must have selector and value
#FillAction: #BrowserAction & {
	action_type: "fill"
	parameters: {
		selector: string & !=""
		value:    string
	}
}

// Click action: must have selector
#ClickAction: #BrowserAction & {
	action_type: "click"
	parameters: {
		selector: string & !=""
	}
}

// Example default policy
policy: {
	allowed_url_patterns: #AllowedURLPatterns | *["*.example.com"]
	forbidden_selectors:  #ForbiddenSelectors | *[
		"[data-testid='delete-all']",
		"[data-testid='admin-reset']",
		".danger-zone button",
	]
	max_input_length: #MaxInputLength

	// No navigation to data: or javascript: URIs
	no_dangerous_schemes: true
}
