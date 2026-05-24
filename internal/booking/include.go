package booking

// Satisfies ADR-022

import (
	"net/http"
	"strings"
)

// ParseIncludeFlags parses the ?include query parameter from the request.
// Default (no param): items included, guest ID only, no folio.
// ?include=none: lightweight envelope without items.
func ParseIncludeFlags(r *http.Request) IncludeFlags {
	raw := r.URL.Query().Get("include")
	if raw == "" {
		return IncludeFlags{Items: true} // default: items included
	}

	flags := IncludeFlags{Items: true}
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "items":
			flags.Items = true
		case "guest":
			flags.Guest = true
		case "folio_summary":
			flags.FolioSummary = true
		case "none":
			flags.None = true
			flags.Items = false
		}
	}
	return flags
}
