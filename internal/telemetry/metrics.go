package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all metric instruments for the application
type Metrics struct {
	RequestCount    metric.Int64Counter
	RequestDuration metric.Float64Histogram
	ErrorCount      metric.Int64Counter
}

// NewMetrics creates and initializes all metric instruments
func NewMetrics() (*Metrics, error) {
	// Get global meter provider
	meter := otel.Meter("github.com/isabermoussa/personal-assistant-API")

	// Create request counter
	requestCount, err := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	// Create request duration histogram
	requestDuration, err := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	// Create error counter
	errorCount, err := meter.Int64Counter(
		"http.server.errors",
		metric.WithDescription("Total number of HTTP errors (status >= 400)"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		RequestCount:    requestCount,
		RequestDuration: requestDuration,
		ErrorCount:      errorCount,
	}, nil
}
