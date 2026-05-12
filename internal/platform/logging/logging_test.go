package logging

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLogger_Dev(t *testing.T) {
	logger := NewLogger("dev")

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	// We can't directly check the level, but we can verify it's a valid logger
	if logger.Handler() == nil {
		t.Fatal("Logger handler is nil")
	}
}

func TestNewLogger_Prod(t *testing.T) {
	logger := NewLogger("prod")

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	if logger.Handler() == nil {
		t.Fatal("Logger handler is nil")
	}
}

func TestWithContext_AndFromContext(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger("dev")

	ctxWithLogger := WithContext(ctx, logger)
	retrievedLogger := FromContext(ctxWithLogger)

	if retrievedLogger != logger {
		t.Error("Retrieved logger is not the same as stored logger")
	}
}

func TestFromContext_NoLoggerStored(t *testing.T) {
	ctx := context.Background()

	logger := FromContext(ctx)

	if logger == nil {
		t.Fatal("FromContext returned nil")
	}

	// Should return the default logger
	if logger != slog.Default() {
		t.Error("Expected default logger when none stored in context")
	}
}

func TestWithTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger("dev")

	enriched := WithTraceID(ctx, logger)

	// Should return the same logger (not recording span)
	if enriched != logger {
		t.Error("WithTraceID should return original logger when no span in context")
	}
}

func TestWithTraceID_WithSpan(t *testing.T) {
	// Create a tracer provider and tracer
	tp := noop.NewTracerProvider()
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger := NewLogger("dev")
	enriched := WithTraceID(ctx, logger)

	if enriched == nil {
		t.Fatal("WithTraceID returned nil")
	}

	// The logger should be enriched (which we verify by checking it's not nil)
	// We can't easily verify the attributes added, but we can verify it works
}

func TestLoggingIntegration(t *testing.T) {
	// Create a context with a logger
	ctx := context.Background()
	logger := NewLogger("dev")
	ctx = WithContext(ctx, logger)

	// Retrieve the logger
	retrieved := FromContext(ctx)

	if retrieved == nil {
		t.Fatal("Retrieved logger is nil")
	}

	// Enrich with trace ID (should work even without a real span)
	enriched := WithTraceID(ctx, retrieved)

	if enriched == nil {
		t.Fatal("Enriched logger is nil")
	}
}
