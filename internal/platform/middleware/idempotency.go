package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/logging"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyKeyHeader = "Idempotency-Key"
	idempotencyPrefix    = "idempotency:"
	idempotencyTTL       = 24 * time.Hour
)

// idempotencyResponse represents a cached response
type idempotencyResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"` // base64-encoded
}

// responseCapture wraps http.ResponseWriter to capture the response
type responseCapture struct {
	http.ResponseWriter
	status      int
	headers     map[string]string
	body        *bytes.Buffer
	written     bool
	headersCopy map[string]string
}

func newResponseCapture(w http.ResponseWriter) *responseCapture {
	return &responseCapture{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		headers:        make(map[string]string),
		headersCopy:    make(map[string]string),
	}
}

func (rc *responseCapture) WriteHeader(status int) {
	if rc.written {
		return
	}
	rc.written = true
	rc.status = status

	// Capture headers from the underlying writer for caching
	for k, v := range rc.ResponseWriter.Header() {
		if len(v) > 0 {
			rc.headersCopy[k] = v[0]
		}
	}

	rc.ResponseWriter.WriteHeader(status)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if !rc.written {
		rc.WriteHeader(http.StatusOK)
	}

	n, err := rc.body.Write(b)
	if err == nil {
		rc.ResponseWriter.Write(b)
	}

	return n, err
}

func (rc *responseCapture) Header() http.Header {
	return rc.ResponseWriter.Header()
}

// Idempotency creates middleware that enforces idempotent request handling using Redis.
// POST/PATCH requests must include an Idempotency-Key header.
// If the key exists in Redis, the cached response is replayed.
// If Redis is unavailable, the middleware fails open (allows the request through).
func Idempotency(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only enforce for POST and PATCH requests
			if r.Method != http.MethodPost && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get(idempotencyKeyHeader)

			// Missing idempotency key for POST/PATCH
			if idempotencyKey == "" {
				http.Error(w, `{"code":"BAD_REQUEST","message":"Idempotency-Key header is required"}`, http.StatusBadRequest)
				return
			}

			redisCtx := context.Background()

			// Check if we have a cached response
			cachedData, err := rdb.Get(redisCtx, idempotencyPrefix+idempotencyKey).Result()
			if err == nil {
				// Cache hit - replay response
				replayCachedResponse(w, cachedData)
				return
			} else if err != redis.Nil {
				// Redis error - fail open with warning
				logger := logging.FromContext(r.Context())
				logger.Warn("redis error during idempotency check, allowing request through", "error", err)
			}

			// New request - capture response and cache it
			rc := newResponseCapture(w)

			// Call next handler
			next.ServeHTTP(rc, r)

			// Cache the response if status is 2xx
			if rc.status >= 200 && rc.status < 300 {
				cacheResponse(redisCtx, rdb, idempotencyKey, rc)
			}
		})
	}
}

// replayCachedResponse deserializes and writes the cached response
func replayCachedResponse(w http.ResponseWriter, data string) {
	var cached idempotencyResponse

	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"cached response is corrupted"}`, http.StatusInternalServerError)
		return
	}

	// Write cached headers
	for k, v := range cached.Headers {
		w.Header().Set(k, v)
	}

	// Decode body
	bodyBytes, err := base64.StdEncoding.DecodeString(cached.Body)
	if err != nil {
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"cached body is corrupted"}`, http.StatusInternalServerError)
		return
	}

	// Write status and body
	w.WriteHeader(cached.Status)
	w.Write(bodyBytes)
}

// cacheResponse serializes and stores the response in Redis
func cacheResponse(ctx context.Context, rdb *redis.Client, key string, rc *responseCapture) {
	cached := idempotencyResponse{
		Status:  rc.status,
		Headers: rc.headersCopy,
		Body:    base64.StdEncoding.EncodeToString(rc.body.Bytes()),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return // Silently skip caching on marshal error
	}

	rdb.Set(ctx, idempotencyPrefix+key, string(data), idempotencyTTL)
}
