// Package telemetry provides OpenTelemetry instrumentation for metrics and tracing
package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Shutdown represents a function to cleanup telemetry resources
type Shutdown func(context.Context) error

// InitMetrics initializes the OpenTelemetry metrics provider with stdout exporter
// Returns a shutdown function that should be called on application exit
func InitMetrics(ctx context.Context) (Shutdown, error) {
	// Create stdout exporter for metrics
	exporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	// Create periodic reader that exports metrics every 30 seconds
	reader := metric.NewPeriodicReader(exporter,
		metric.WithInterval(30*time.Second))

	// Create meter provider
	provider := metric.NewMeterProvider(
		metric.WithReader(reader),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	slog.Info("OpenTelemetry metrics initialized with stdout exporter")

	// Return shutdown function
	return func(ctx context.Context) error {
		slog.Info("Shutting down metrics provider...")
		return provider.Shutdown(ctx)
	}, nil
}

// InitTracing initializes the OpenTelemetry tracing provider with stdout exporter
// Returns a shutdown function that should be called on application exit
func InitTracing(ctx context.Context) (Shutdown, error) {
	// Create stdout exporter for traces
	exporter, err := stdouttrace.New()
	if err != nil {
		return nil, err
	}

	// Create tracer provider with batch span processor
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	slog.Info("OpenTelemetry tracing initialized with stdout exporter")

	// Return shutdown function
	return func(ctx context.Context) error {
		slog.Info("Shutting down tracer provider...")
		return provider.Shutdown(ctx)
	}, nil
}

// InitTelemetry initializes both metrics and tracing
// Returns a combined shutdown function
func InitTelemetry(ctx context.Context) (Shutdown, error) {
	// Initialize metrics
	shutdownMetrics, err := InitMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize tracing
	shutdownTracing, err := InitTracing(ctx)
	if err != nil {
		// Cleanup metrics if tracing fails
		_ = shutdownMetrics(ctx)
		return nil, err
	}

	// Return combined shutdown function
	return func(ctx context.Context) error {
		// Shutdown both providers
		errMetrics := shutdownMetrics(ctx)
		errTracing := shutdownTracing(ctx)

		if errMetrics != nil {
			return errMetrics
		}
		return errTracing
	}, nil
}
