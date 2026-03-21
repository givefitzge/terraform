// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

//lint:file-ignore U1000 Some utilities in here are intentionally unused in VCS but are for temporary use while debugging a test.

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/setup"
	"github.com/hashicorp/terraform/internal/telemetrytest"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// overwriteTestSpanTimestamps overwrites the timestamps in all of the given
// spans to be exactly the given fakeTime, as a way to avoid considering exact
// timestamps when comparing actual spans with desired spans.
//
// This function overwrites both the start and end times of the spans themselves
// and also the timestamps of any events associated with the spans.
func overwriteTestSpanTimestamps(spans tracetest.SpanStubs, fakeTime time.Time) {
	for i := range spans {
		spans[i].StartTime = fakeTime
		spans[i].EndTime = fakeTime
		for j := range spans[i].Events {
			spans[i].Events[j].Time = fakeTime
		}
	}
}

func fixedTraceID(n uint32) trace.TraceID {
	return trace.TraceID{
		0xfe, 0xed, 0xfa, 0xce,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n >> 0),
	}
}

func fixedSpanID(n uint32) trace.SpanID {
	return trace.SpanID{
		0xfa, 0xce, 0xfe, 0xed,
		uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n >> 0),
	}
}

func TestTelemetryInTests(t *testing.T) {
	ctx := context.Background()

	testResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("telemetry test"),
		semconv.ServiceVersionKey.String("1.2.3"),
	)

	telemetry := telemetrytest.Init(t,
		sdktrace.WithResource(testResource),
	)

	var parentSpanContext, childSpanContext trace.SpanContext

	tracer := otel.Tracer("test thingy")
	{
		ctx, parentSpan := tracer.Start(ctx, "parent span")
		parentSpanContext = parentSpan.SpanContext()
		{
			_, childSpan := tracer.Start(ctx, "child span")
			childSpanContext = childSpan.SpanContext()
			childSpan.AddEvent("did something totally hilarious")
			childSpan.SetStatus(codes.Error, "it went wrong")
			childSpan.End()
		}
		parentSpan.End()
	}

	gotSpans := telemetry.GetSpans()

	// The spans contain real timestamps that make them annoying to compare,
	// so we'll just replace those with fixed timestamps so we can easily
	// compare everything else.
	fakeTime := time.Now()
	overwriteTestSpanTimestamps(gotSpans, fakeTime)

	wantSpans := tracetest.SpanStubs{
		// These are ordered by the calls to Span.End above, so child should
		// always appear first. (That's a detail of this in-memory-only
		// exporter, not a general guarantee about OpenTracing.)
		{
			Name:        "child span",
			SpanContext: childSpanContext,
			Parent:      parentSpanContext,
			SpanKind:    trace.SpanKindInternal,
			StartTime:   fakeTime,
			EndTime:     fakeTime,
			Events: []sdktrace.Event{
				{
					Name: "did something totally hilarious",
					Time: fakeTime,
				},
			},
			Status: sdktrace.Status{
				Code:        codes.Error,
				Description: "it went wrong",
			},
			Resource: testResource,
			InstrumentationLibrary: instrumentation.Scope{
				Name: "test thingy",
			},
			InstrumentationScope: instrumentation.Scope{
				Name: "test thingy",
			},
		},
		{
			Name:           "parent span",
			SpanContext:    parentSpanContext,
			SpanKind:       trace.SpanKindInternal,
			StartTime:      fakeTime,
			EndTime:        fakeTime,
			ChildSpanCount: 1,
			Resource:       testResource,
			InstrumentationLibrary: instrumentation.Scope{
				Name: "test thingy",
			},
			InstrumentationScope: instrumentation.Scope{
				Name: "test thingy",
			},
		},
	}

	if diff := cmp.Diff(wantSpans, gotSpans, cmpopts.IgnoreUnexported(attribute.Set{})); diff != "" {
		t.Errorf("wrong spans\n%s", diff)
	}
}

func TestTelemetryInTestsGRPC(t *testing.T) {
	ctx := context.Background()

	testResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("TestTelemetryInTestsGRPC"),
	)
	telemetry := telemetrytest.Init(t,
		sdktrace.WithResource(testResource),
	)

	client, close := grpcClientForTesting(ctx, t, func(srv *grpc.Server) {
		server := &setupServer{
			initOthers: func(ctx context.Context, cc *setup.Handshake_Request, stopper *stopper) (*setup.ServerCapabilities, error) {
				return &setup.ServerCapabilities{}, nil
			},
		}
		setup.RegisterSetupServer(srv, server)
	})
	defer close()
	setupClient := setup.NewSetupClient(client)

	{
		ctx, span := otel.Tracer("TestTelemetryInTestsGRPC").Start(ctx, "root")
		_, err := setupClient.Handshake(ctx, &setup.Handshake_Request{
			Capabilities: &setup.ClientCapabilities{},
		})
		if err != nil {
			t.Fatal(err)
		}
		span.End()
	}

	clientSpan := telemetrytest.FindSpan(t, telemetry, func(ss tracetest.SpanStub) bool {
		return ss.SpanKind == trace.SpanKindClient
	})
	serverSpan := telemetrytest.FindSpan(t, telemetry, func(ss tracetest.SpanStub) bool {
		return ss.SpanKind == trace.SpanKindServer
	})
	t.Run("client span", func(t *testing.T) {
		span := clientSpan
		t.Logf("client span: %s", spew.Sdump(span))
		if got, want := span.Name, "terraform1.setup.Setup/Handshake"; got != want {
			t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
		}
		attrs := telemetrytest.AttributesMap(span.Attributes)
		if got, want := attrs["rpc.system"], "grpc"; got != want {
			t.Errorf("wrong rpc.system\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.service"], "terraform1.setup.Setup"; got != want {
			t.Errorf("wrong rpc.service\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.method"], "Handshake"; got != want {
			t.Errorf("wrong rpc.method\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("server span", func(t *testing.T) {
		span := serverSpan
		t.Logf("server span: %s", spew.Sdump(span))
		if got, want := span.Name, "terraform1.setup.Setup/Handshake"; got != want {
			t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := span.Parent.SpanID(), clientSpan.SpanContext.SpanID(); got != want {
			t.Errorf("server span is not a child of the client span\nclient span ID:        %s\nserver span parent ID: %s", want, got)
		}
		if got, want := serverSpan.SpanContext.TraceID(), clientSpan.SpanContext.TraceID(); got != want {
			t.Errorf("server span belongs to different trace than client span\nclient trace ID: %s\nserver trace ID: %s", want, got)
		}
		attrs := telemetrytest.AttributesMap(span.Attributes)
		if got, want := attrs["rpc.system"], "grpc"; got != want {
			t.Errorf("wrong rpc.system\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.service"], "terraform1.setup.Setup"; got != want {
			t.Errorf("wrong rpc.service\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.method"], "Handshake"; got != want {
			t.Errorf("wrong rpc.method\ngot:  %s\nwant: %s", got, want)
		}
	})
}
