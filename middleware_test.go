package gotel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func TestTracingMiddleware(t *testing.T) {
	server := createMockServerWithTracingMiddleware()
	defer server.Close()

	t.Run("GET", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/hello")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := "GET /hello"
		if expected != string(respBody) {
			t.Fatalf("expected %s; got %s", expected, respBody)
		}
	})

	t.Run("POST", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/hello", "text/plain", bytes.NewReader([]byte("world")))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := "POST /hello"
		if expected != string(respBody) {
			t.Fatalf("expected %s; got %s", expected, respBody)
		}
	})

	t.Run("panic", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/panic", "text/plain", bytes.NewReader([]byte("world")))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := `Internal Server Error`
		if !strings.Contains(string(respBody), expected) {
			t.Fatalf("expected %s; got %s", expected, respBody)
		}
	})
}

// cpu: Apple M3 Pro
// BenchmarkTracingMiddleware/GET-11         	    9240	    127783 ns/op	   23334 B/op	     204 allocs/op
// BenchmarkTracingMiddleware/POST-11        	    9022	    423028 ns/op	   22979 B/op	     202 allocs/op
func BenchmarkTracingMiddleware(b *testing.B) {
	server := createMockServerWithTracingMiddleware()
	defer server.Close()

	b.Run("GET", func(b *testing.B) {
		for b.Loop() {
			resp, _ := http.Get(server.URL + "/hello")
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})

	b.Run("POST", func(b *testing.B) {
		for b.Loop() {
			resp, _ := http.Post(server.URL+"/hello", "text/plain", bytes.NewReader([]byte("world")))
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})
}

func TestWithSensitivePatterns(t *testing.T) {
	// Build a server that uses a custom sensitive pattern so we can verify
	// the option is wired through without panicking and the middleware still
	// serves requests correctly.
	mux := http.NewServeMux()

	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	exporters := &OTelExporters{
		Tracer: NewTracer("test-sensitive"),
		Meter:  otel.Meter("test-sensitive"),
		Logger: logger,
		Shutdown: func(ctx context.Context) error {
			return nil
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	options := []TracingMiddlewareOption{
		WithAllowedRequestHeaders([]string{}),
		WithAllowedResponseHeaders([]string{"Content-Type"}),
		WithSensitivePatterns([]string{"credential", "x-custom-secret"}),
	}

	mux.Handle("/sensitive", NewTracingMiddleware(exporters, options...)(handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("request with sensitive header is served successfully", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/sensitive", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Custom-Secret", "should-be-masked")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if string(body) != "ok" {
			t.Fatalf("expected body 'ok', got %q", string(body))
		}
	})
}

func TestWithAllowedHeaders(t *testing.T) {
	// Verify that WithAllowedRequestHeaders and WithAllowedResponseHeaders
	// apply their allow-lists to the middleware options correctly.
	tmo := &tracingMiddlewareOptions{}

	WithAllowedRequestHeaders([]string{"Content-Type", "X-Request-ID"})(tmo)
	if len(tmo.AllowedRequestHeaders) != 2 {
		t.Fatalf("expected 2 allowed request headers, got %d", len(tmo.AllowedRequestHeaders))
	}
	// Values should be lower-cased.
	if tmo.AllowedRequestHeaders[0] != "content-type" {
		t.Errorf("expected 'content-type', got %q", tmo.AllowedRequestHeaders[0])
	}
	if tmo.AllowedRequestHeaders[1] != "x-request-id" {
		t.Errorf("expected 'x-request-id', got %q", tmo.AllowedRequestHeaders[1])
	}

	WithAllowedResponseHeaders([]string{"Cache-Control"})(tmo)
	if len(tmo.AllowedResponseHeaders) != 1 {
		t.Fatalf("expected 1 allowed response header, got %d", len(tmo.AllowedResponseHeaders))
	}
	if tmo.AllowedResponseHeaders[0] != "cache-control" {
		t.Errorf("expected 'cache-control', got %q", tmo.AllowedResponseHeaders[0])
	}
}

func createMockServerWithTracingMiddleware() *httptest.Server {
	mux := http.NewServeMux()

	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	exporters := &OTelExporters{
		Tracer: NewTracer("test"),
		Meter:  otel.Meter("test"),
		Logger: logger,
		Shutdown: func(ctx context.Context) error {
			return nil
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%s %s", r.Method, r.URL.Path)))
	})

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	options := []TracingMiddlewareOption{
		WithAllowedRequestHeaders([]string{}),
		WithAllowedResponseHeaders([]string{"Content-Type"}),
		WithDebugPaths([]string{"/world"}),
		WithHighCardinalitySpans(false),
		WithHighCardinalityMetrics(true),
		WithCustomAttributesFunc(func(r *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{}
		}),
	}

	mux.Handle("/hello", NewTracingMiddleware(exporters, options...)(handler))
	mux.Handle("/panic", NewTracingMiddleware(exporters, options...)(panicHandler))

	return httptest.NewServer(mux)
}
