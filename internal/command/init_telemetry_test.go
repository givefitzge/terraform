// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/hashicorp/cli"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/hashicorp/terraform/internal/telemetrytest"
)

// initTelemetryTest sets up telemetry capture for a test and re-obtains the
// package-level tracer so it points at the freshly-installed provider.
// This is necessary because OTel's global delegation is one-shot: the proxy
// tracer obtained during package init() is permanently wired to the first real
// provider set via SetTracerProvider. After that provider is shut down in a
// previous test's cleanup, we must replace the package-level tracer variable
// with one obtained directly from the new global provider.
func initTelemetryTest(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	origTracer := tracer
	exp := telemetrytest.Init(t)
	tracer = otel.GetTracerProvider().Tracer("github.com/hashicorp/terraform/internal/command")
	t.Cleanup(func() { tracer = origTracer })
	return exp
}

func TestInitCommand_telemetry_backend(t *testing.T) {
	// Set up telemetry capture — must be before any command execution
	telemetry := initTelemetryTest(t)

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-backend"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("init failed: \n%s", done(t).All())
	}
	_ = done(t)

	// Verify "initialize backend" span was emitted
	telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		return s.Name == "initialize backend"
	})

	// Verify "install providers from config" span was emitted
	telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		return s.Name == "install providers from config"
	})
}

func TestInitCommand_telemetry_modules(t *testing.T) {
	telemetry := initTelemetryTest(t)

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("init failed: \n%s", done(t).All())
	}
	_ = done(t)

	// Verify "install modules" span was emitted with upgrade=false.
	// There are two "install modules" spans (one outer from getModules with
	// the upgrade attribute, one inner from installModules without it), so
	// we match on the presence of the upgrade attribute.
	span := telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		if s.Name != "install modules" {
			return false
		}
		for _, attr := range s.Attributes {
			if string(attr.Key) == "upgrade" {
				return true
			}
		}
		return false
	})
	attrs := telemetrytest.AttributesMap(span.Attributes)
	if got, want := attrs["upgrade"], false; got != want {
		t.Errorf("wrong upgrade attribute\ngot:  %v\nwant: %v", got, want)
	}
}

func TestInitCommand_telemetry_modules_upgrade(t *testing.T) {
	telemetry := initTelemetryTest(t)

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-get"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// Run with -upgrade flag
	args := []string{"-upgrade"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("init failed: \n%s", done(t).All())
	}
	_ = done(t)

	// Verify "install modules" span was emitted with upgrade=true.
	// Same as above, match on the span that carries the upgrade attribute.
	span := telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		if s.Name != "install modules" {
			return false
		}
		for _, attr := range s.Attributes {
			if string(attr.Key) == "upgrade" {
				return true
			}
		}
		return false
	})
	attrs := telemetrytest.AttributesMap(span.Attributes)
	if got, want := attrs["upgrade"], true; got != want {
		t.Errorf("wrong upgrade attribute\ngot:  %v\nwant: %v", got, want)
	}
}

func TestInitCommand_telemetry_providers_from_state(t *testing.T) {
	telemetry := initTelemetryTest(t)

	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-provider-download/state-file-only"), td)
	t.Chdir(td)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/random": {"1.0.0", "9.9.9"},
	})
	defer close()

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
			ProviderSource:   providerSource,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("init failed: \n%s", done(t).All())
	}
	_ = done(t)

	// Verify "install providers from config" span was emitted
	telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		return s.Name == "install providers from config"
	})

	// Verify "install providers from state" span was emitted
	telemetrytest.FindSpan(t, telemetry, func(s tracetest.SpanStub) bool {
		return s.Name == "install providers from state"
	})
}
