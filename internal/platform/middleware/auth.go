package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/apierror"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
	platformjson "github.com/lexxcode1/yop-pms/internal/platform/json"
)

// StubAuth reads X-Property-ID and X-User-Permissions request headers and
// writes them into the request context. Real JWT-backed auth replaces only
// this middleware in a future PR — everything downstream remains unchanged.
//
// Returns 400 if X-Property-ID is missing or not a valid UUID.
func StubAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Swagger UI is unauthenticated — remove or gate on APP_ENV=dev before production.
		if strings.HasPrefix(r.URL.Path, "/swagger") {
			next.ServeHTTP(w, r)
			return
		}
		pid := r.Header.Get("X-Property-ID")
		if pid == "" {
			platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("X-Property-ID header is required"))
			return
		}
		id, err := uuid.Parse(pid)
		if err != nil {
			platformjson.WriteError(w, r, apierror.ErrBadRequest.WithMessage("X-Property-ID must be a valid UUID"))
			return
		}

		raw := r.Header.Get("X-User-Permissions")
		var perms []string
		if raw != "" {
			for _, p := range strings.Split(raw, ",") {
				if t := strings.TrimSpace(p); t != "" {
					perms = append(perms, t)
				}
			}
		}

		ctx := helpers.SetPropertyIDInCtx(r.Context(), id)
		ctx = helpers.SetPermissionsInCtx(ctx, perms)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
