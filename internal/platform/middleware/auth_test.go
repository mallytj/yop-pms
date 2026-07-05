package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
)

func TestStubAuth_SwaggerBypass(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/swagger/index.html", nil)
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for swagger path, got %d", w.Code)
	}
}

func TestStubAuth_MissingPropertyID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing X-Property-ID, got %d", w.Code)
	}
}

func TestStubAuth_InvalidUUID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	r.Header.Set("X-Property-ID", "not-a-uuid")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid X-Property-ID UUID, got %d", w.Code)
	}
}

func TestStubAuth_ValidPropertyID(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		pid := helpers.GetPropertyIDFromCtx(r.Context())
		if pid == uuid.Nil {
			t.Error("expected non-nil property ID in context")
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	r.Header.Set("X-Property-ID", uuid.New().String())
	wrapped.ServeHTTP(w, r)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStubAuth_PermissionsInContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		perms := helpers.GetPermissionsFromCtx(r.Context())
		if len(perms) != 2 {
			t.Errorf("expected 2 permissions, got %d", len(perms))
			return
		}
		if !helpers.HasPermission(r.Context(), "reservations:read") {
			t.Error("expected reservations:read permission")
		}
		if !helpers.HasPermission(r.Context(), "reservations:write") {
			t.Error("expected reservations:write permission")
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	r.Header.Set("X-Property-ID", uuid.New().String())
	r.Header.Set("X-User-Permissions", "reservations:read, reservations:write")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStubAuth_EmptyPermissionsHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		perms := helpers.GetPermissionsFromCtx(r.Context())
		if len(perms) != 0 {
			t.Errorf("expected 0 permissions, got %d", len(perms))
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := StubAuth(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	r.Header.Set("X-Property-ID", uuid.New().String())
	// No X-User-Permissions header
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
