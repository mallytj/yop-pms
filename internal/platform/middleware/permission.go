package middleware

import (
	"net/http"

	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// RequirePermission returns a middleware that enforces a single permission on
// the request context. Permissions are set by StubAuth (or real auth in future).
// Returns 403 if the permission is absent.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !helpers.HasPermission(r.Context(), perm) {
				platformjson.WriteError(w, r, apierror.New("MISSING_PERMISSION", "missing permission: "+perm, http.StatusForbidden))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
