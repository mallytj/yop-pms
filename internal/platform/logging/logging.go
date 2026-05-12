package logging

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const keyLogger contextKey = "logger"

// NewLogger creates a new structured logger with appropriate level for the environment.
// Debug level is used in "dev" environment, Info level otherwise.
func NewLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "dev" {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

// WithContext stores the logger in the context, returning a new context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, keyLogger, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, returns the default slog logger (never nil).
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(keyLogger).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// WithTraceID enriches the logger with OpenTelemetry trace and span IDs from the context.
// If no span is found in the context, returns the logger unchanged.
func WithTraceID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	sc := span.SpanContext()
	return logger.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}
