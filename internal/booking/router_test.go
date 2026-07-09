package booking

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	yopMw "github.com/lexxcode1/yop-pms/internal/platform/middleware"
)

func TestRouter_RoutesRegistered(t *testing.T) {
	r := chi.NewRouter()
	r.Use(yopMw.StubAuth)
	r.Route("/reservations", Routes(testSvc, yopMw.RequireIfMatch))

	// Check that known routes return 404 (not 405) for the right methods
	tests := []struct {
		method string
		path   string
		perms  string
		want   int
	}{
		{http.MethodGet, "/reservations", "reservations:read", http.StatusOK},
		{http.MethodPost, "/reservations", "", http.StatusForbidden},                                                     // no permissions
		{http.MethodGet, "/reservations/00000000-0000-0000-0000-000000000001", "reservations:read", http.StatusNotFound}, // no such reservation
		{http.MethodGet, "/reservations/availability", "", http.StatusBadRequest},                                        // missing query params
		// Nonexistent route (bad UUID) returns 400 from ParseUUIDParam
		{http.MethodGet, "/reservations/nonexistent", "reservations:read", http.StatusBadRequest},
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, ts.URL+tt.path, nil)
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
				t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
			}
		})
	}
}
