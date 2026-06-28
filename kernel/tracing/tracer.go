// Package tracing defines the distributed tracing port.
//
// v1 ships a NoopTracer (zero overhead). A future v2 will provide an
// OTelTracer backed by go.opentelemetry.io/otel with OTLP/Jaeger export.
package tracing

import "context"

// Tracer is the tracing port. Inject it into use cases and repositories
// to instrument cross-cutting operations.
type Tracer interface {
	Start(ctx context.Context, spanName string) (context.Context, Span)
}

// Span represents an active trace span.
type Span interface {
	End()
	SetError(err error)
	SetAttribute(key string, value any)
}

// NoopTracer is a Tracer that does nothing. Use in v1 and in tests.
type NoopTracer struct{}

// Start returns the same context and a noop span.
func (NoopTracer) Start(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End()                        {}
func (noopSpan) SetError(_ error)            {}
func (noopSpan) SetAttribute(_ string, _ any) {}
