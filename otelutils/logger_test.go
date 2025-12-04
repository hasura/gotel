package otelutils

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestNewJSONLogger(t *testing.T) {
	t.Run("creates logger with INFO level", func(t *testing.T) {
		logger, level, err := NewJSONLogger("INFO")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		if level != slog.LevelInfo {
			t.Errorf("expected level INFO, got %v", level)
		}
	})

	t.Run("creates logger with DEBUG level", func(t *testing.T) {
		logger, level, err := NewJSONLogger("DEBUG")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		if level != slog.LevelDebug {
			t.Errorf("expected level DEBUG, got %v", level)
		}
	})

	t.Run("creates logger with WARN level", func(t *testing.T) {
		logger, level, err := NewJSONLogger("WARN")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		if level != slog.LevelWarn {
			t.Errorf("expected level WARN, got %v", level)
		}
	})

	t.Run("creates logger with ERROR level", func(t *testing.T) {
		logger, level, err := NewJSONLogger("ERROR")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		if level != slog.LevelError {
			t.Errorf("expected level ERROR, got %v", level)
		}
	})

	t.Run("returns error for invalid level", func(t *testing.T) {
		_, _, err := NewJSONLogger("INVALID")
		if err == nil {
			t.Error("expected error for invalid log level")
		}
	})

	t.Run("logger writes JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		logger.Info("test message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("expected output to contain 'test message', got: %s", output)
		}

		if !strings.Contains(output, "key") {
			t.Errorf("expected output to contain 'key', got: %s", output)
		}

		if !strings.Contains(output, "value") {
			t.Errorf("expected output to contain 'value', got: %s", output)
		}
	})
}

func TestNewContextWithLogger(t *testing.T) {
	t.Run("adds logger to context", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		ctx := context.Background()

		newCtx := NewContextWithLogger(ctx, logger)

		if newCtx == nil {
			t.Fatal("expected non-nil context")
		}

		// Retrieve the logger from context
		value := newCtx.Value(LoggerContextKey)
		if value == nil {
			t.Fatal("expected logger to be in context")
		}

		retrievedLogger, ok := value.(*slog.Logger)
		if !ok {
			t.Fatal("expected value to be *slog.Logger")
		}

		if retrievedLogger != logger {
			t.Error("expected retrieved logger to be the same instance")
		}
	})

	t.Run("preserves parent context values", func(t *testing.T) {
		type testKey string
		key := testKey("test")
		value := "test-value"

		parentCtx := context.WithValue(context.Background(), key, value)

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))

		newCtx := NewContextWithLogger(parentCtx, logger)

		// Check that parent context value is preserved
		retrievedValue := newCtx.Value(key)
		if retrievedValue != value {
			t.Errorf("expected parent context value to be preserved, got %v", retrievedValue)
		}

		// Check that logger is also in context
		loggerValue := newCtx.Value(LoggerContextKey)
		if loggerValue == nil {
			t.Fatal("expected logger to be in context")
		}
	})
}

func TestNewHeaderLogGroupAttrs(t *testing.T) {
	t.Run("converts single value headers", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": []string{"application/json"},
			"Accept":       []string{"application/json"},
		}

		attr := NewHeaderLogGroupAttrs("headers", headers)

		if attr.Key != "headers" {
			t.Errorf("expected key 'headers', got '%s'", attr.Key)
		}

		// The attribute should be a group
		if attr.Value.Kind() != slog.KindGroup {
			t.Errorf("expected group attribute, got %v", attr.Value.Kind())
		}
	})

	t.Run("converts multi-value headers", func(t *testing.T) {
		headers := http.Header{
			"Accept-Encoding": []string{"gzip", "deflate", "br"},
		}

		attr := NewHeaderLogGroupAttrs("headers", headers)

		if attr.Key != "headers" {
			t.Errorf("expected key 'headers', got '%s'", attr.Key)
		}

		if attr.Value.Kind() != slog.KindGroup {
			t.Errorf("expected group attribute, got %v", attr.Value.Kind())
		}
	})

	t.Run("skips empty headers", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": []string{"application/json"},
			"Empty":        []string{},
		}

		attr := NewHeaderLogGroupAttrs("headers", headers)

		if attr.Key != "headers" {
			t.Errorf("expected key 'headers', got '%s'", attr.Key)
		}

		// Check that the group doesn't contain the empty header
		group := attr.Value.Group()
		for _, a := range group {
			if a.Key == "Empty" {
				t.Error("expected empty header to be skipped")
			}
		}
	})

	t.Run("handles empty header map", func(t *testing.T) {
		headers := http.Header{}

		attr := NewHeaderLogGroupAttrs("headers", headers)

		if attr.Key != "headers" {
			t.Errorf("expected key 'headers', got '%s'", attr.Key)
		}

		if attr.Value.Kind() != slog.KindGroup {
			t.Errorf("expected group attribute, got %v", attr.Value.Kind())
		}
	})
}

func TestContextKey_String(t *testing.T) {
	key := &contextKey{name: "TestKey"}

	expected := "context value TestKey"
	if key.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, key.String())
	}
}

