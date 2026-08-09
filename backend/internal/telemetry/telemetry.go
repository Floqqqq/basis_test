package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Enabled          bool
	ServiceName      string
	ExporterEndpoint string
	ExporterInsecure bool
}

type ShutdownFunc func(context.Context) error

func Init(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("telemetry service name is required")
	}
	if cfg.ExporterEndpoint == "" {
		return nil, fmt.Errorf("telemetry exporter endpoint is required")
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.ExporterEndpoint),
	}
	if cfg.ExporterInsecure {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func HTTPHandler(next http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(next, serviceName)
}

func RouteSpan(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		routeContext := chi.RouteContext(r.Context())
		if routeContext == nil {
			return
		}
		route := routeContext.RoutePattern()
		if route == "" {
			return
		}

		span := trace.SpanFromContext(r.Context())
		span.SetName(r.Method + " " + route)
		span.SetAttributes(attribute.String("http.route", route))
	})
}
