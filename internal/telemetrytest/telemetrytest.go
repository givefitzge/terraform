// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package telemetrytest

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Init configures OpenTelemetry to collect spans into a local in-memory
// buffer and returns the exporter that provides access to that buffer.
//
// The OpenTelemetry tracer provider is a global cross-cutting concern shared
// throughout the program, so it isn't valid to use this function in any test
// that calls t.Parallel, or in subtests of a parent test that has already
// used this function.
func Init(t *testing.T, providerOptions ...sdktrace.TracerProviderOption) *tracetest.InMemoryExporter {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	sp := sdktrace.NewSimpleSpanProcessor(exp)
	providerOptions = append(
		[]sdktrace.TracerProviderOption{
			sdktrace.WithSpanProcessor(sp),
		},
		providerOptions...,
	)
	provider := sdktrace.NewTracerProvider(providerOptions...)
	otel.SetTracerProvider(provider)

	pgtr := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(pgtr)

	t.Cleanup(func() {
		provider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetTextMapPropagator(nil)
	})

	return exp
}

// FindSpan tests each of the spans that have been reported to the given
// exporter with the given predicate function and returns the first one
// for which the predicate matches.
//
// If the predicate returns false for all spans then this function will fail
// the test using the given [testing.T].
func FindSpan(t *testing.T, exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStub {
	t.Helper()
	for _, span := range exp.GetSpans() {
		if predicate(span) {
			return span
		}
	}
	t.Fatal("no spans matched the predicate")
	return tracetest.SpanStub{}
}

// FindSpans tests each of the spans that have been reported to the given
// exporter with the given predicate function and returns only those for
// which the predicate matches.
//
// If no spans match at all then the result is a zero-length slice.
func FindSpans(exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStubs {
	var ret tracetest.SpanStubs
	for _, span := range exp.GetSpans() {
		if predicate(span) {
			ret = append(ret, span)
		}
	}
	return ret
}

// AttributesMap converts a slice of OpenTelemetry key-value attributes into
// a plain map for easier assertion in tests.
func AttributesMap(attrs []attribute.KeyValue) map[string]any {
	ret := make(map[string]any, len(attrs))
	for _, kv := range attrs {
		ret[string(kv.Key)] = kv.Value.AsInterface()
	}
	return ret
}
