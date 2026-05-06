package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/logging"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyKeyHeader = "Idempotency-Key"
	idempotencyPrefix    = "idempotency:"
	idempotencyTTL       = 24 * time.Hour
	idempotencyLockTTL   = 2 * time.Minute
	idempotencyWait      = 10 * time.Second
	idempotencyPoll      = 100 * time.Millisecond
)

const (
	idempotencyStateProcessing = "processing"
	idempotencyStateCompleted  = "completed"
)

// idempotencyResponse represents a cached response
type idempotencyResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"` // base64-encoded
}

// idempotencyRecord represents either an in-flight reservation or a completed response.
type idempotencyRecord struct {
	State       string               `json:"state"`
	Fingerprint string               `json:"fingerprint"`
	Response    *idempotencyResponse `json:"response,omitempty"`
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

			fingerprint, err := requestFingerprint(r)
			if err != nil {
				http.Error(w, `{"code":"BAD_REQUEST","message":"failed to read request body"}`, http.StatusBadRequest)
				return
			}

			redisCtx := r.Context()
			redisKey := idempotencyPrefix + idempotencyKey
			logger := logging.FromContext(r.Context())

			acquired, err := reserveIdempotencyKey(redisCtx, rdb, redisKey, fingerprint)
			if err != nil {
				// Redis error - fail open with warning
				logger.Warn("redis error during idempotency reservation, allowing request through", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if !acquired {
				handled, err := handleExistingIdempotencyKey(redisCtx, rdb, redisKey, fingerprint, w)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return
					}
					logger.Warn("redis error while waiting for idempotency response, allowing request through", "error", err)
					next.ServeHTTP(w, r)
					return
				}
				if handled {
					return
				}
			}

			// New request - capture response and cache it
			rc := newResponseCapture(w)

			// Call next handler
			next.ServeHTTP(rc, r)

			// Cache the response if status is 2xx
			if rc.status >= 200 && rc.status < 300 {
				cacheResponse(redisCtx, rdb, redisKey, fingerprint, rc)
			} else {
				rdb.Del(redisCtx, redisKey)
			}
		})
	}
}

func requestFingerprint(r *http.Request) (string, error) {
	var body []byte
	var err error

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		r.Body.Close()
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	bodyHash := sha256.Sum256(body)
	authHash := sha256.Sum256([]byte(r.Header.Get("Authorization")))

	h := sha256.New()
	h.Write([]byte(r.Method))
	h.Write([]byte{0})
	h.Write([]byte(r.URL.RequestURI()))
	h.Write([]byte{0})
	h.Write([]byte(hex.EncodeToString(authHash[:])))
	h.Write([]byte{0})
	h.Write([]byte(hex.EncodeToString(bodyHash[:])))

	return hex.EncodeToString(h.Sum(nil)), nil
}

func reserveIdempotencyKey(ctx context.Context, rdb *redis.Client, redisKey string, fingerprint string) (bool, error) {
	record := idempotencyRecord{
		State:       idempotencyStateProcessing,
		Fingerprint: fingerprint,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return false, err
	}

	return rdb.SetNX(ctx, redisKey, string(data), idempotencyLockTTL).Result()
}

func handleExistingIdempotencyKey(ctx context.Context, rdb *redis.Client, redisKey string, fingerprint string, w http.ResponseWriter) (bool, error) {
	deadline := time.NewTimer(idempotencyWait)
	defer deadline.Stop()

	ticker := time.NewTicker(idempotencyPoll)
	defer ticker.Stop()

	for {
		handled, wait, err := checkExistingIdempotencyKey(ctx, rdb, redisKey, fingerprint, w)
		if err != nil {
			return false, err
		}
		if handled || !wait {
			return handled, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadline.C:
			http.Error(w, `{"code":"CONFLICT","message":"request with this Idempotency-Key is still processing"}`, http.StatusConflict)
			return true, nil
		case <-ticker.C:
		}
	}
}

func checkExistingIdempotencyKey(ctx context.Context, rdb *redis.Client, redisKey string, fingerprint string, w http.ResponseWriter) (handled bool, wait bool, err error) {
	data, err := rdb.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		acquired, err := reserveIdempotencyKey(ctx, rdb, redisKey, fingerprint)
		if err != nil {
			return false, false, err
		}
		return false, !acquired, nil
	}
	if err != nil {
		return false, false, err
	}

	var record idempotencyRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"cached response is corrupted"}`, http.StatusInternalServerError)
		return true, false, nil
	}

	if record.Fingerprint != fingerprint {
		http.Error(w, `{"code":"CONFLICT","message":"Idempotency-Key was already used for a different request"}`, http.StatusConflict)
		return true, false, nil
	}

	switch record.State {
	case idempotencyStateCompleted:
		if record.Response == nil {
			http.Error(w, `{"code":"INTERNAL_ERROR","message":"cached response is corrupted"}`, http.StatusInternalServerError)
			return true, false, nil
		}
		replayCachedResponse(w, *record.Response)
		return true, false, nil
	case idempotencyStateProcessing:
		return false, true, nil
	default:
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"cached response is corrupted"}`, http.StatusInternalServerError)
		return true, false, nil
	}
}

// replayCachedResponse writes the cached response.
func replayCachedResponse(w http.ResponseWriter, cached idempotencyResponse) {
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
func cacheResponse(ctx context.Context, rdb *redis.Client, redisKey string, fingerprint string, rc *responseCapture) {
	record := idempotencyRecord{
		State:       idempotencyStateCompleted,
		Fingerprint: fingerprint,
		Response: &idempotencyResponse{
			Status:  rc.status,
			Headers: rc.headersCopy,
			Body:    base64.StdEncoding.EncodeToString(rc.body.Bytes()),
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		return // Silently skip caching on marshal error
	}

	rdb.Set(ctx, redisKey, string(data), idempotencyTTL)
}
