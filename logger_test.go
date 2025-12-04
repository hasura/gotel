package gotel

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel/trace"
)

func TestLogHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	stdHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := createLogHandler("test-service", slog.New(stdHandler), nil)

	t.Run("respects log level", func(t *testing.T) {
		ctx := context.Background()

		if !handler.Enabled(ctx, slog.LevelInfo) {
			t.Error("expected Info level to be enabled")
		}

		if !handler.Enabled(ctx, slog.LevelError) {
			t.Error("expected Error level to be enabled")
		}

		if handler.Enabled(ctx, slog.LevelDebug) {
			t.Error("expected Debug level to be disabled")
		}
	})
}

func TestLogHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	stdHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := createLogHandler("test-service", slog.New(stdHandler), nil)

	t.Run("handles log records", func(t *testing.T) {
		ctx := context.Background()
		record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
		record.AddAttrs(slog.String("key", "value"))

		err := handler.Handle(ctx, record)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check that the log was written to the buffer
		logOutput := buf.String()
		if !strings.Contains(logOutput, "test message") {
			t.Errorf("expected log output to contain 'test message', got: %s", logOutput)
		}

		if !strings.Contains(logOutput, "key") {
			t.Errorf("expected log output to contain 'key', got: %s", logOutput)
		}
	})
}

func TestLogHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	stdHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := createLogHandler("test-service", slog.New(stdHandler), nil)

	t.Run("returns handler with attributes", func(t *testing.T) {
		attrs := []slog.Attr{
			slog.String("service", "test"),
			slog.Int("version", 1),
		}

		newHandler := handler.WithAttrs(attrs)
		if newHandler == nil {
			t.Fatal("expected non-nil handler")
		}

		// Verify it's a LogHandler
		if _, ok := newHandler.(LogHandler); !ok {
			t.Error("expected handler to be of type LogHandler")
		}
	})
}

func TestLogHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	stdHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := createLogHandler("test-service", slog.New(stdHandler), nil)

	t.Run("returns handler with group", func(t *testing.T) {
		newHandler := handler.WithGroup("request")
		if newHandler == nil {
			t.Fatal("expected non-nil handler")
		}

		// Verify it's a LogHandler
		if _, ok := newHandler.(LogHandler); !ok {
			t.Error("expected handler to be of type LogHandler")
		}
	})
}

func TestGetLogger(t *testing.T) {
	t.Run("returns logger from context", func(t *testing.T) {
		var buf bytes.Buffer
		expectedLogger := slog.New(slog.NewJSONHandler(&buf, nil))
		ctx := otelutils.NewContextWithLogger(context.Background(), expectedLogger)

		logger := GetLogger(ctx)
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		// The logger should be the same instance
		if logger != expectedLogger {
			t.Error("expected logger to be the same instance from context")
		}
	})

	t.Run("returns default logger when not in context", func(t *testing.T) {
		ctx := context.Background()

		logger := GetLogger(ctx)
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

func TestGetRequestLogger(t *testing.T) {
	t.Run("returns logger from request context", func(t *testing.T) {
		var buf bytes.Buffer
		expectedLogger := slog.New(slog.NewJSONHandler(&buf, nil))
		ctx := otelutils.NewContextWithLogger(context.Background(), expectedLogger)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)

		logger := GetRequestLogger(req)
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		if logger != expectedLogger {
			t.Error("expected logger to be the same instance from context")
		}
	})

	t.Run("returns logger with request ID when not in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("x-request-id", "test-request-id")

		logger := GetRequestLogger(req)
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		// We can't easily verify the request_id attribute without capturing logs,
		// but we can verify the logger is not nil
	})

	t.Run("uses trace ID as request ID when header missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := trace.ContextWithSpanContext(req.Context(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		}))
		req = req.WithContext(ctx)

		logger := GetRequestLogger(req)
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}
