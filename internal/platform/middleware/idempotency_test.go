package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestIdempotency_PassthroughGET(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	wrappedHandler.ServeHTTP(w, r)

	if handlerCalled != 1 {
		t.Errorf("Handler called: got %d, want 1", handlerCalled)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestIdempotency_PostWithoutKey(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))

	wrappedHandler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestIdempotency_PostWithKeyNewRequest(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
	r.Header.Set("Idempotency-Key", "test-key-1")

	wrappedHandler.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusCreated)
	}

	if handlerCalled != 1 {
		t.Errorf("Handler called: got %d, want 1", handlerCalled)
	}

	if w.Body.String() != "created" {
		t.Errorf("Body: got %q, want %q", w.Body.String(), "created")
	}
}

func TestIdempotency_PostWithKeyCached(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	wrappedHandler := middleware(handler)

	// First request - should execute handler
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
	r1.Header.Set("Idempotency-Key", "test-key-2")

	wrappedHandler.ServeHTTP(w1, r1)

	if w1.Code != http.StatusCreated {
		t.Errorf("First request status: got %d, want %d", w1.Code, http.StatusCreated)
	}

	if handlerCalled != 1 {
		t.Errorf("Handler called after first request: got %d, want 1", handlerCalled)
	}

	// Second request with same key - should replay cached response
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
	r2.Header.Set("Idempotency-Key", "test-key-2")

	wrappedHandler.ServeHTTP(w2, r2)

	if w2.Code != http.StatusCreated {
		t.Errorf("Cached request status: got %d, want %d", w2.Code, http.StatusCreated)
	}

	if handlerCalled != 1 {
		t.Errorf("Handler called after cached request: got %d, want 1", handlerCalled)
	}

	if w2.Body.String() != "created" {
		t.Errorf("Cached body: got %q, want %q", w2.Body.String(), "created")
	}
}

func TestIdempotency_PatchWithoutCaching(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	wrappedHandler := middleware(handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/test", bytes.NewReader([]byte("{}")))
	r.Header.Set("Idempotency-Key", "test-key-3")

	wrappedHandler.ServeHTTP(w, r)

	// 5xx responses should not be cached
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestIdempotency_DifferentKeys(t *testing.T) {
	rdb, cleanup := newTestRedis(t)
	defer cleanup()

	middleware := Idempotency(rdb)

	handlerCalled := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrappedHandler := middleware(handler)

	// First request
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
	r1.Header.Set("Idempotency-Key", "key-1")
	wrappedHandler.ServeHTTP(w1, r1)

	// Second request with different key
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("{}")))
	r2.Header.Set("Idempotency-Key", "key-2")
	wrappedHandler.ServeHTTP(w2, r2)

	// Handler should be called twice (different keys)
	if handlerCalled != 2 {
		t.Errorf("Handler called: got %d, want 2", handlerCalled)
	}
}

func TestIdempotency_ResponseCapture_WriteHeaderTwice(t *testing.T) {
	w := httptest.NewRecorder()
	rc := newResponseCapture(w)

	rc.WriteHeader(http.StatusOK)
	rc.WriteHeader(http.StatusInternalServerError) // Should be ignored

	if rc.status != http.StatusOK {
		t.Errorf("Status: got %d, want %d", rc.status, http.StatusOK)
	}
}

func TestIdempotency_ResponseCapture_AutoStatus(t *testing.T) {
	w := httptest.NewRecorder()
	rc := newResponseCapture(w)

	rc.Write([]byte("test"))

	if rc.status != http.StatusOK {
		t.Errorf("Status: got %d, want %d", rc.status, http.StatusOK)
	}

	if rc.body.String() != "test" {
		t.Errorf("Body: got %q, want %q", rc.body.String(), "test")
	}
}
