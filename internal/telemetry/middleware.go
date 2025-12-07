package telemetry

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// MetricsMiddleware returns an HTTP middleware that records metrics for each request
func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				written:        false,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := float64(time.Since(start).Milliseconds())

			// Prepare attributes
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.Int("http.status_code", wrapped.statusCode),
			}

			// Record request count
			metrics.RequestCount.Add(r.Context(), 1, metric.WithAttributes(attrs...))

			// Record request duration
			metrics.RequestDuration.Record(r.Context(), duration, metric.WithAttributes(attrs...))

			// Record error count if status >= 400
			if wrapped.statusCode >= 400 {
				metrics.ErrorCount.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			}
		})
	}
}

// TracingMiddleware returns an HTTP middleware that creates a span for each request
func TracingMiddleware() func(http.Handler) http.Handler {
	tracer := otel.Tracer("github.com/isabermoussa/personal-assistant-API")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start a new span
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.path", r.URL.Path),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				written:        false,
			}

			// Process request with updated context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Add status code to span
			span.SetAttributes(attribute.Int("http.status_code", wrapped.statusCode))

			// Mark span as error if status >= 400
			if wrapped.statusCode >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}
