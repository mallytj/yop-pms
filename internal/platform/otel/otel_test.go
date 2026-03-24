package otel

import (
	"context"
	"strings"
	"testing"

	gootel "go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func TestSetup_NoEndpoint_ReturnsNoopShutdown(t *testing.T) {
	shutdown, err := Setup(context.Background(), Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "",
		Environment:    "dev",
	})
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function is nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}
}

func TestSetup_WithFakeEndpoint_Succeeds(t *testing.T) {
	// Restore the global tracer provider after the test so it does not leak
	// into subsequent tests. Setup installs a real SDK provider globally.
	prev := gootel.GetTracerProvider()
	t.Cleanup(func() { gootel.SetTracerProvider(prev) })

	// The HTTP exporter is lazy — it does not connect until the first export.
	// Setup should succeed even with an unreachable endpoint.
	shutdown, err := Setup(context.Background(), Config{
		ServiceName:    "test-service",
		ServiceVersion: "0.1.0",
		OTLPEndpoint:   "localhost:4318",
		Environment:    "test",
	})
	if err != nil {
		t.Fatalf("Setup with fake endpoint returned unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	// Shutdown may error if the SDK tries to flush to an unreachable endpoint.
	// Log but do not fail — the purpose of this test is that Setup succeeds, not that flush succeeds.
	if err := shutdown(context.Background()); err != nil {
		t.Logf("shutdown with unreachable endpoint returned (expected): %v", err)
	}

	// Restore immediately after shutdown so the global is clean for other tests.
	gootel.SetTracerProvider(nooptrace.NewTracerProvider())
}

func TestNewSampler_Dev_IsAlwaysSample(t *testing.T) {
	s := newSampler("dev")
	if s.Description() != sdktrace.AlwaysSample().Description() {
		t.Errorf("dev sampler: got %q, want AlwaysSample", s.Description())
	}
}

func TestNewSampler_NonDev_IsParentBased(t *testing.T) {
	cases := []struct{ name, env string }{
		{"prod", "prod"},
		{"staging", "staging"},
		{"test", "test"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSampler(tc.env)
			if s.Description() == sdktrace.AlwaysSample().Description() {
				t.Errorf("env=%q: expected ParentBased sampler, got AlwaysSample", tc.env)
			}
			if !strings.Contains(s.Description(), "ParentBased") {
				t.Errorf("env=%q: sampler description %q does not contain ParentBased", tc.env, s.Description())
			}
		})
	}
}

func TestNewResource_HasServiceAttributes(t *testing.T) {
	r := newResource("my-service", "2.0.0")
	if r == nil {
		t.Fatal("newResource returned nil")
	}

	attrs := make(map[string]string)
	for _, kv := range r.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}

	if got := attrs["service.name"]; got != "my-service" {
		t.Errorf("service.name: got %q, want %q", got, "my-service")
	}
	if got := attrs["service.version"]; got != "2.0.0" {
		t.Errorf("service.version: got %q, want %q", got, "2.0.0")
	}
}
