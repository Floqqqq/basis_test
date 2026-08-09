package service

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTaskServiceListCreatesChildSpan(t *testing.T) {
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

	ctx, parent := provider.Tracer("test").Start(context.Background(), "HTTP GET /api/v1/tasks")
	service := NewTaskService(&fakeTaskRepository{}, NewTaskPolicy(fakeTeamAccess{
		members: map[int64]bool{42: true},
	}), nil)
	if _, err := service.List(ctx, 42, 5, "todo", nil, 20, 0); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	parent.End()

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(spans))
	}
	if spans[0].Name() != "TaskService.List" {
		t.Fatalf("child span name = %q, want TaskService.List", spans[0].Name())
	}
	if spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
		t.Fatal("TaskService.List span is not a child of HTTP span")
	}
}
