package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lexxcode1/yop-pms/internal/platform/logging"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// RequestLogger creates a middleware that logs HTTP requests with structured data.
// It injects a per-request logger into the context with request metadata (ID, method, path, remote IP).
// It also enriches the logger with OTel trace/span IDs if available.
func RequestLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract request ID from context (set by chi's middleware.RequestID)
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = "unknown"
			}

			// Extract client IP
			clientIP := getClientIP(r)

			// Create per-request logger with metadata
			reqLogger := baseLogger.With(
				"request_id", requestID,
				"method", r.Method,
				"path", r.RequestURI,
				"remote_ip", clientIP,
			)

			// Enrich with OTel trace/span IDs from context
			reqLogger = logging.WithTraceID(r.Context(), reqLogger)

			// Store logger in context for downstream handlers
			ctx := logging.WithContext(r.Context(), reqLogger)
			r = r.WithContext(ctx)

			// Wrap response writer to capture status and bytes written
			rw := &responseWriter{ResponseWriter: w}

			// Call next handler
			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			reqLogger.InfoContext(r.Context(), "request completed",
				slog.Int("status", rw.status),
				slog.Int("bytes_written", rw.bytesWritten),
				slog.Float64("latency_ms", duration.Seconds()*1000),
			)
		})
	}
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For header first, then X-Real-IP, then RemoteAddr.
func getClientIP(r *http.Request) string {
	// X-Forwarded-For header (used by proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP from the list
		ips := parseCSV(forwarded)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// X-Real-IP header (another proxy header)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// RemoteAddr
	if r.RemoteAddr != "" {
		// Remove port if present
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return ip
		}
		return r.RemoteAddr
	}

	return "unknown"
}

// parseCSV splits a comma-separated string and trims whitespace
func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
