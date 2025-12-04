package gotel

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestGetDefault(t *testing.T) {
	t.Run("returns value when not empty", func(t *testing.T) {
		result := getDefault("value", "default")
		if result != "value" {
			t.Errorf("expected 'value', got '%s'", result)
		}
	})

	t.Run("returns default when value is empty", func(t *testing.T) {
		result := getDefault("", "default")
		if result != "default" {
			t.Errorf("expected 'default', got '%s'", result)
		}
	})

	t.Run("works with integers", func(t *testing.T) {
		result := getDefault(0, 42)
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}

		result = getDefault(10, 42)
		if result != 10 {
			t.Errorf("expected 10, got %d", result)
		}
	})
}

func TestGetDefaultPtr(t *testing.T) {
	t.Run("returns value when not nil", func(t *testing.T) {
		value := "test"
		defaultValue := "default"
		result := getDefaultPtr(&value, &defaultValue)
		if result == nil || *result != "test" {
			t.Errorf("expected 'test', got %v", result)
		}
	})

	t.Run("returns default when value is nil", func(t *testing.T) {
		defaultValue := "default"
		result := getDefaultPtr[string](nil, &defaultValue)
		if result == nil || *result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("returns nil when both are nil", func(t *testing.T) {
		result := getDefaultPtr[string](nil, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns x-request-id header when present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("x-request-id", "test-request-id")

		requestID := getRequestID(req)
		if requestID != "test-request-id" {
			t.Errorf("expected 'test-request-id', got '%s'", requestID)
		}
	})

	t.Run("returns trace ID when x-request-id is missing and trace exists", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		// Create a context with a trace span
		ctx := trace.ContextWithSpanContext(req.Context(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:  trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		}))
		req = req.WithContext(ctx)

		requestID := getRequestID(req)
		if requestID != "0102030405060708090a0b0c0d0e0f10" {
			t.Errorf("expected trace ID '0102030405060708090a0b0c0d0e0f10', got '%s'", requestID)
		}
	})

	t.Run("generates UUID when no x-request-id or trace", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		requestID := getRequestID(req)
		// UUID should be 36 characters with dashes
		if len(requestID) != 36 {
			t.Errorf("expected UUID length 36, got %d", len(requestID))
		}
	})
}

func TestDebugRequestBody(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("reads and restores request body", func(t *testing.T) {
		body := "test body content"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
		w := httptest.NewRecorder()

		result, err := debugRequestBody(w, req, logger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != body {
			t.Errorf("expected '%s', got '%s'", body, result)
		}

		// Verify body can be read again
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read body again: %v", err)
		}

		if string(bodyBytes) != body {
			t.Errorf("body not restored correctly, expected '%s', got '%s'", body, string(bodyBytes))
		}
	})
}

func TestWriteResponseJSON(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	t.Run("writes JSON response correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"message": "test"}

		writeResponseJSON(w, http.StatusOK, body, logger)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
		}

		expected := `{"message":"test"}` + "\n"
		if w.Body.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, w.Body.String())
		}
	})
}

func TestToLowerStrings(t *testing.T) {
	t.Run("converts strings to lowercase", func(t *testing.T) {
		input := []string{"Hello", "WORLD", "TeSt"}
		expected := []string{"hello", "world", "test"}

		result := toLowerStrings(input)

		if len(result) != len(expected) {
			t.Fatalf("expected length %d, got %d", len(expected), len(result))
		}

		for i, v := range result {
			if v != expected[i] {
				t.Errorf("at index %d: expected '%s', got '%s'", i, expected[i], v)
			}
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		result := toLowerStrings([]string{})
		if len(result) != 0 {
			t.Errorf("expected empty slice, got length %d", len(result))
		}
	})
}
