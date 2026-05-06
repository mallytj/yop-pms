package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lexxcode1/yop-pms/internal/platform/logging"
)

func TestRequestLogger_Basic(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	middleware := RequestLogger(logger)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Check that logger is in context
		ctxLogger := logging.FromContext(r.Context())
		if ctxLogger == nil {
			t.Fatal("Logger not found in context")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("X-Request-ID", "test-123")

	wrappedHandler.ServeHTTP(w, r)

	if !handlerCalled {
		t.Fatal("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "test" {
		t.Errorf("Body: got %q, want %q", w.Body.String(), "test")
	}
}

func TestRequestLogger_ResponseCapture(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	middleware := RequestLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/resource", nil)

	wrappedHandler.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestRequestLogger_ContextInjection(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	middleware := RequestLogger(logger)

	loggerFromHandler := (*slog.Logger)(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggerFromHandler = logging.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	wrappedHandler.ServeHTTP(w, r)

	if loggerFromHandler == nil {
		t.Fatal("Logger was not injected into context")
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:8080"

	ip := getClientIP(r)

	if ip != "192.168.1.1" {
		t.Errorf("IP: got %q, want %q", ip, "192.168.1.1")
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	r.RemoteAddr = "192.168.1.1:8080"

	ip := getClientIP(r)

	if ip != "10.0.0.1" {
		t.Errorf("IP: got %q, want %q", ip, "10.0.0.1")
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "172.16.0.1")
	r.RemoteAddr = "192.168.1.1:8080"

	ip := getClientIP(r)

	if ip != "172.16.0.1" {
		t.Errorf("IP: got %q, want %q", ip, "172.16.0.1")
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"10.0.0.1, 10.0.0.2", []string{"10.0.0.1", "10.0.0.2"}},
		{"10.0.0.1,10.0.0.2", []string{"10.0.0.1", "10.0.0.2"}},
		{"10.0.0.1", []string{"10.0.0.1"}},
		{"", []string{}},
		{"  ,  ", []string{}},
	}

	for _, tt := range tests {
		result := parseCSV(tt.input)

		if len(result) != len(tt.expected) {
			t.Errorf("Input %q: got %d items, want %d", tt.input, len(result), len(tt.expected))
			continue
		}

		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("Input %q: item %d got %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestResponseWriter_Wrapping(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}

	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte("error"))

	if rw.status != http.StatusBadRequest {
		t.Errorf("Status: got %d, want %d", rw.status, http.StatusBadRequest)
	}

	if rw.bytesWritten != 5 {
		t.Errorf("BytesWritten: got %d, want 5", rw.bytesWritten)
	}
}

func TestRequestLogger_ContextPreservation(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	middleware := RequestLogger(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context values are preserved
		logger := logging.FromContext(r.Context())
		if logger == nil {
			t.Error("Logger not in context")
		}

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	wrappedHandler.ServeHTTP(w, r)
}
