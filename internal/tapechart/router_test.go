package tapechart

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
)

// TestRouter_RequiresReservationsReadPermission guards against the missing
// permission check flagged in code review — the route was previously wide
// open to any authenticated caller regardless of granted permissions.
func TestRouter_RequiresReservationsReadPermission(t *testing.T) {
	r := chi.NewRouter()
	r.Use(yopMw.StubAuth)
	r.Route("/tape-chart", Routes(testSvc))

	ts := httptest.NewServer(r)
	defer ts.Close()

	tests := []struct {
		name  string
		perms string
		want  int
	}{
		{"missing permission is forbidden", "", http.StatusForbidden},
		{"unrelated permission is forbidden", "reservations:create", http.StatusForbidden},
		{"reservations:read is allowed", "reservations:read", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/tape-chart?from=2026-07-13&to=2026-07-20", nil)
			req.Header.Set("X-Property-ID", testPropertyID.String())
			if tt.perms != "" {
				req.Header.Set("X-User-Permissions", tt.perms)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}
