package gotel

import (
	"context"

	"go.opentelemetry.io/otel"
	traceapi "go.opentelemetry.io/otel/trace"
)

// Tracer is the wrapper of traceapi.Tracer with user visibility on Hasura Console.
type Tracer struct {
	traceapi.Tracer
}

var _ traceapi.Tracer = &Tracer{}

// NewTracer creates a new OpenTelemetry tracer.
func NewTracer(name string, opts ...traceapi.TracerOption) *Tracer {
	return &Tracer{
		Tracer: otel.Tracer(name, opts...),
	}
}

// Start creates a span and a context.Context containing the newly-created span
// with `internal.visibility: "user"` so that it shows up in the Hasura Console.
func (t *Tracer) Start(
	ctx context.Context,
	spanName string,
	opts ...traceapi.SpanStartOption,
) (context.Context, traceapi.Span) {
	return t.Tracer.Start( //nolint:spancheck
		ctx,
		spanName,
		append(opts, traceapi.WithAttributes(UserVisibilityAttribute))...)
}

// StartInternal creates a span and a context.Context containing the newly-created span.
// It won't show up in the Hasura Console.
func (t *Tracer) StartInternal(
	ctx context.Context,
	spanName string,
	opts ...traceapi.SpanStartOption,
) (context.Context, traceapi.Span) {
	return t.Tracer.Start(ctx, spanName, opts...) //nolint:spancheck
}
