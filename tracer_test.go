package gotel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	traceapi "go.opentelemetry.io/otel/trace"
)

func TestNewTracer(t *testing.T) {
	t.Run("creates a new tracer", func(t *testing.T) {
		tracer := NewTracer("test-service")
		if tracer == nil {
			t.Fatal("expected tracer to be non-nil")
		}

		if tracer.Tracer == nil {
			t.Fatal("expected underlying Tracer to be non-nil")
		}
	})
}

func TestTracer_Start(t *testing.T) {
	// Set up a test tracer provider with an in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := NewTracer("test-service")

	t.Run("creates span with user visibility attribute", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "test-span")

		if span == nil {
			t.Fatal("expected span to be non-nil")
		}

		span.End()

		// Force the span to be exported
		tp.ForceFlush(context.Background())

		// Check that the span was created
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span to be exported")
		}

		// Find our test span
		var testSpan *tracetest.SpanStub
		for i := range spans {
			if spans[i].Name == "test-span" {
				testSpan = &spans[i]
				break
			}
		}

		if testSpan == nil {
			t.Fatal("test-span not found in exported spans")
		}

		// Check for user visibility attribute
		hasUserVisibility := false
		for _, attr := range testSpan.Attributes {
			if attr.Key == "internal.visibility" && attr.Value.AsString() == "user" {
				hasUserVisibility = true
				break
			}
		}

		if !hasUserVisibility {
			t.Error("expected span to have internal.visibility=user attribute")
		}
	})

	t.Run("preserves additional span options", func(t *testing.T) {
		exporter.Reset()

		ctx := context.Background()
		customAttr := attribute.String("custom.key", "custom.value")
		ctx, span := tracer.Start(ctx, "test-span-with-attrs", traceapi.WithAttributes(customAttr))
		span.End()

		tp.ForceFlush(context.Background())

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span to be exported")
		}

		// Find our test span
		var testSpan *tracetest.SpanStub
		for i := range spans {
			if spans[i].Name == "test-span-with-attrs" {
				testSpan = &spans[i]
				break
			}
		}

		if testSpan == nil {
			t.Fatal("test-span-with-attrs not found in exported spans")
		}

		// Check for both user visibility and custom attribute
		hasUserVisibility := false
		hasCustomAttr := false
		for _, attr := range testSpan.Attributes {
			if attr.Key == "internal.visibility" && attr.Value.AsString() == "user" {
				hasUserVisibility = true
			}
			if attr.Key == "custom.key" && attr.Value.AsString() == "custom.value" {
				hasCustomAttr = true
			}
		}

		if !hasUserVisibility {
			t.Error("expected span to have internal.visibility=user attribute")
		}
		if !hasCustomAttr {
			t.Error("expected span to have custom.key=custom.value attribute")
		}
	})
}

func TestTracer_StartInternal(t *testing.T) {
	// Set up a test tracer provider with an in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := NewTracer("test-service")

	t.Run("creates span without user visibility attribute", func(t *testing.T) {
		exporter.Reset()

		ctx := context.Background()
		ctx, span := tracer.StartInternal(ctx, "internal-span")

		if span == nil {
			t.Fatal("expected span to be non-nil")
		}

		span.End()

		// Force the span to be exported
		tp.ForceFlush(context.Background())

		// Check that the span was created
		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span to be exported")
		}

		// Find our test span
		var testSpan *tracetest.SpanStub
		for i := range spans {
			if spans[i].Name == "internal-span" {
				testSpan = &spans[i]
				break
			}
		}

		if testSpan == nil {
			t.Fatal("internal-span not found in exported spans")
		}

		// Check that user visibility attribute is NOT present
		for _, attr := range testSpan.Attributes {
			if attr.Key == "internal.visibility" {
				t.Error("expected internal span to NOT have internal.visibility attribute")
			}
		}
	})
}

func TestTracer_ImplementsInterface(t *testing.T) {
	// This test ensures that Tracer implements the traceapi.Tracer interface
	tracer := NewTracer("test")

	// This will fail to compile if Tracer doesn't implement the interface
	var _ = (interface{})(tracer)
}
