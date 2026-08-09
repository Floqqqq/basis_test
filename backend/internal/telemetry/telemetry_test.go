package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInitDisabled(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init() shutdown = nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestHTTPHandlerUsesRoutePatternForSpanName(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Fatalf("provider.Shutdown() error = %v", err)
		}
	})

	router := chi.NewRouter()
	router.Use(RouteSpan)
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	recorderHTTP := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/42", nil)
	HTTPHandler(router, "test-service").ServeHTTP(recorderHTTP, request)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	if spans[0].Name() != "GET /api/v1/tasks/{id}" {
		t.Fatalf("span name = %q, want route pattern", spans[0].Name())
	}
}

func TestInitValidatesEnabledConfig(t *testing.T) {
	_, err := Init(context.Background(), Config{Enabled: true})
	if err == nil {
		t.Fatal("Init() error = nil, want validation error")
	}
}
