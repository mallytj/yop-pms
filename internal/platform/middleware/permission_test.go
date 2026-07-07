package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
)

func TestRequirePermission_Denied(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequirePermission("reservations:write")(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	// No permissions set → denied
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for missing permission, got %d", w.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called when permission is denied")
	}
}

func TestRequirePermission_Granted(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequirePermission("reservations:read")(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	ctx := helpers.SetPermissionsInCtx(r.Context(), []string{"reservations:read", "reservations:write"})
	wrapped.ServeHTTP(w, r.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !handlerCalled {
		t.Error("handler should be called when permission is granted")
	}
}

func TestRequirePermission_WrongPermission(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequirePermission("admin:super")(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/admin", nil)
	ctx := helpers.SetPermissionsInCtx(r.Context(), []string{"reservations:read"})
	wrapped.ServeHTTP(w, r.WithContext(ctx))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong permission, got %d", w.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called with wrong permission")
	}
}

func TestRequirePermission_NoContextPermissions(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequirePermission("reservations:read")(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)

	// Context with property ID but no permissions
	ctx := helpers.SetPropertyIDInCtx(r.Context(), uuid.New())
	wrapped.ServeHTTP(w, r.WithContext(ctx))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no permissions in context, got %d", w.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called without permissions")
	}
}
