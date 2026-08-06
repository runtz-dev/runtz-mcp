// Package telemetry wires the OpenTelemetry SDK into the MCP server: OTLP
// traces and metrics over HTTP, plus Go runtime metrics.
//
// It is configured with the standard OTEL_* environment variables the
// OpenTelemetry SDK already defines, so the Helm chart points the server at a
// collector without a code change and the names match every other
// OpenTelemetry deployment an operator runs.
//
//	OTEL_EXPORTER_OTLP_ENDPOINT  Collector base URL, e.g.
//	                             http://signoz-otel-collector.signoz.svc.cluster.local:4318
//	                             An http:// scheme sends in plaintext; https:// uses TLS.
//	OTEL_SERVICE_NAME            Overrides the service name passed to Setup.
//	OTEL_RESOURCE_ATTRIBUTES     Extra resource attributes, e.g.
//	                             deployment.environment=dev,service.namespace=runtz
//	OTEL_SDK_DISABLED            "true" turns the whole SDK off.
//
// With no endpoint set, Setup is a no-op: the global providers stay the SDK's
// no-op ones, so the instrumented handler keeps serving with no exporter
// attached and no network traffic. That is the default, which matters here —
// this binary is documented as running anywhere the standard library does, and
// nobody self-hosting it should need a collector to start it.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Service identifies the process being instrumented. Name and Version are
// defaults: OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES still win, so a
// deployment can relabel a process without rebuilding it.
type Service struct {
	Name    string
	Version string
}

// ShutdownFunc flushes and closes whatever Setup started. It is always safe to
// call, including when telemetry is disabled, and callers should give it its
// own context: the one that triggered shutdown is usually already cancelled.
type ShutdownFunc func(context.Context) error

// Setup installs the global tracer provider, meter provider and propagators,
// and starts collecting Go runtime metrics. It returns the shutdown function
// that flushes buffered spans and metrics on the way out.
//
// Failing to reach the collector is not an error here: the exporters connect
// lazily and retry in the background, so a collector that is down never keeps
// the server from answering MCP requests.
func Setup(ctx context.Context, service Service, logf func(string, ...any)) (ShutdownFunc, error) {
	if !enabled() {
		return func(context.Context) error { return nil }, nil
	}

	// Export failures are the SDK's problem, not the request's. Log them and
	// keep serving rather than letting them surface at the call site.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logf("opentelemetry error: %v", err)
	}))

	res, err := newResource(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		// The tracer provider is already global at this point; tear it back
		// down so a half-initialised pipeline never outlives this error.
		return nil, errors.Join(
			fmt.Errorf("create otlp metric exporter: %w", err),
			tracerProvider.Shutdown(ctx),
		)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	// W3C tracecontext plus baggage: what the collector and every other
	// OpenTelemetry SDK speak by default, so a request that arrives already
	// traced stays on the same trace instead of starting a new one.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if err := runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
		logf("failed to start go runtime metrics: %v", err)
	}

	logf("opentelemetry enabled: service=%s endpoint=%s", service.Name, endpoint())

	return func(ctx context.Context) error {
		return errors.Join(
			tracerProvider.Shutdown(ctx),
			meterProvider.Shutdown(ctx),
		)
	}, nil
}

// newResource describes this process to the collector. Order matters: the
// explicit attributes go in first so that resource.WithFromEnv — which reads
// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES — can override them.
func newResource(ctx context.Context, service Service) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(service.Name)}
	if service.Version != "" {
		attrs = append(attrs, semconv.ServiceVersion(service.Version))
	}

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithProcessRuntimeDescription(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithFromEnv(),
	)
}

// enabled reports whether an exporter should be built at all. It mirrors the
// SDK's own rules so that this server turns on and off exactly like any other
// OpenTelemetry process.
func enabled() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OTEL_SDK_DISABLED")), "true") {
		return false
	}

	return endpoint() != ""
}

// endpoint returns the collector URL in effect, preferring the signal-specific
// variable the way the SDK does.
func endpoint() string {
	if value := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")); value != "" {
		return value
	}

	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
}
