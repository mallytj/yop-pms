package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Config struct {
	ServiceName    string `example:"yop-pms"`
	ServiceVersion string `example:"0.1.0"`
	OTLPEndpoint   string `example:"localhost:4318"`
	Environment    string `example:"dev"`
}

// Setup initializes the OpenTelemetry tracer provider with the given configuration.
// If OTLPEndpoint is empty, returns a no-op tracer provider (useful for development).
// The returned shutdown function must be called to flush traces before process exit.
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	// If no OTLP endpoint is configured, use a no-op tracer provider
	if cfg.OTLPEndpoint == "" {
		return func(ctx context.Context) error { return nil }, nil
	}

	// Create an exporter (prefer gRPC if endpoint suggests it, otherwise HTTP)
	var exporter sdktrace.SpanExporter

	// Try HTTP first (more common for dev setups)
	httpExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.OTLPEndpoint))
	if err == nil {
		exporter = httpExporter
	} else {
		// Fallback to gRPC
		grpcExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
		if err != nil {
			return nil, err
		}
		exporter = grpcExporter
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(newSampler(cfg.Environment)),
		sdktrace.WithResource(newResource(cfg.ServiceName, cfg.ServiceVersion)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

// newSampler creates an appropriate sampler based on the environment.
// In "dev" mode, always samples all traces. In other modes, uses parent-based sampling.
func newSampler(env string) sdktrace.Sampler {
	if env == "dev" {
		return sdktrace.AlwaysSample()
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))
}

// newResource creates a resource with service attributes for traces.
func newResource(serviceName, serviceVersion string) *resource.Resource {
	return resource.NewWithAttributes(
		"",
		attribute.String("service.name", serviceName),
		attribute.String("service.version", serviceVersion),
	)
}
