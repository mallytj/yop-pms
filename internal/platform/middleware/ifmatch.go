package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// RequireIfMatch parses the If-Match header on PATCH and POST requests and
// stores the parsed version in context. Returns 400 if the header is missing
// or not a valid positive integer.
//
// GET requests pass through without inspection.
func RequireIfMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		raw := r.Header.Get("If-Match")
		if raw == "" {
			platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("If-Match header is required for mutating requests"))
			return
		}
		trimmed := strings.Trim(raw, `"`)
		v, err := strconv.Atoi(trimmed)
		if err != nil || v < 1 {
			platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("If-Match must be a positive integer version number"))
			return
		}
		ctx := helpers.SetIfMatchVersion(r.Context(), int32(v)) //nolint:gosec
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
