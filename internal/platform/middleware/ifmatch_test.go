package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/helpers"
)

func TestRequireIfMatch_GETPassthrough(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/reservations", nil)
	wrapped.ServeHTTP(w, r)

	if !handlerCalled {
		t.Error("expected GET to pass through without If-Match")
	}
}

func TestRequireIfMatch_MissingHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_NonInteger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", "abc")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_Negative(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", "-1")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_Zero(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", "0")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for zero If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_ValidVersion(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := helpers.GetIfMatchVersion(r.Context())
		if v != 5 {
			t.Errorf("expected version 5 in context, got %d", v)
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", "5")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireIfMatch_QuotedVersion(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := helpers.GetIfMatchVersion(r.Context())
		if v != 3 {
			t.Errorf("expected version 3 in context, got %d", v)
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", `"3"`)
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for quoted If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_LargeVersion(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := helpers.GetIfMatchVersion(r.Context())
		if v != 2147483647 {
			t.Errorf("expected max int32 version in context, got %d", v)
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/reservations/123", nil)
	r.Header.Set("If-Match", strconv.Itoa(2147483647))
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for large If-Match, got %d", w.Code)
	}
}

func TestRequireIfMatch_POST(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := helpers.GetIfMatchVersion(r.Context())
		if v != 1 {
			t.Errorf("expected version 1 in context, got %d", v)
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequireIfMatch(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/reservations/123/confirm", nil)
	r.Header.Set("If-Match", "1")
	wrapped.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for POST with If-Match, got %d", w.Code)
	}
}
