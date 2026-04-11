package otelutils

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestExtractTelemetryHeaders(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          http.Header
		AllowedHeaders []string
		Expected       [][]string
	}{
		{
			Name: "basic",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"Authorization": []string{
					"Bearer abcdefghijkxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				},
				"Api-Key":    []string{"abcxyz"},
				"Secret-Key": []string{"secret-key"},
				"X-Empty":    []string{},
			},
			Expected: [][]string{
				{"api-key", MaskString},
				{"authorization", MaskString},
				{"content-type", "application/json"},
				{"secret-key", MaskString},
			},
		},
		{
			Name: "allowed_list",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"Authorization": []string{
					"Bearer abcdefghijkxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				},
				"Api-Key":    []string{"abcxyz"},
				"Secret-Key": []string{"secret-key"},
			},
			AllowedHeaders: []string{"Content-Type", "Api-Key"},
			Expected: [][]string{
				{"api-key", MaskString},
				{"content-type", "application/json"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got := ExtractTelemetryHeaders(tc.Input, nil, tc.AllowedHeaders...)
			slices.SortFunc(got, func(a, b []string) int {
				return strings.Compare(a[0], b[0])
			})

			if !reflect.DeepEqual(tc.Expected, got) {
				t.Errorf("expected: %v, got: %v", tc.Expected, got)
			}

			if reflect.DeepEqual(tc.Input, got) {
				t.Errorf("input: %v, got: %v", tc.Input, got)
			}
		})
	}
}

func TestEvaluateSensitiveHeader(t *testing.T) {
	testCases := []struct {
		Name        string
		Input       string
		ExpectedKey string
		IsSensitive bool
	}{
		{
			Name:        "authorization header",
			Input:       "Authorization",
			ExpectedKey: "authorization",
			IsSensitive: true,
		},
		{
			Name:        "api-key header",
			Input:       "Api-Key",
			ExpectedKey: "api-key",
			IsSensitive: true,
		},
		{
			Name:        "secret header",
			Input:       "X-Secret-Token",
			ExpectedKey: "x-secret-token",
			IsSensitive: true,
		},
		{
			Name:        "password header",
			Input:       "Password",
			ExpectedKey: "password",
			IsSensitive: true,
		},
		{
			Name:        "token header",
			Input:       "X-Auth-Token",
			ExpectedKey: "x-auth-token",
			IsSensitive: true,
		},
		{
			Name:        "content-type header",
			Input:       "Content-Type",
			ExpectedKey: "content-type",
			IsSensitive: false,
		},
		{
			Name:        "accept header",
			Input:       "Accept",
			ExpectedKey: "accept",
			IsSensitive: false,
		},
		{
			Name:        "short header",
			Input:       "X",
			ExpectedKey: "x",
			IsSensitive: false,
		},
		{
			Name:        "two char header",
			Input:       "XY",
			ExpectedKey: "xy",
			IsSensitive: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			key := strings.ToLower(tc.Input)
			isSensitive := IsSensitiveHeader(key)

			if key != tc.ExpectedKey {
				t.Errorf("expected key '%s', got '%s'", tc.ExpectedKey, key)
			}

			if isSensitive != tc.IsSensitive {
				t.Errorf("expected isSensitive %v, got %v", tc.IsSensitive, isSensitive)
			}
		})
	}
}

func TestIsSensitiveHeaderCustomPatterns(t *testing.T) {
	testCases := []struct {
		Name        string
		Input       string
		Patterns    []string
		IsSensitive bool
	}{
		{
			Name:        "matches custom pattern",
			Input:       "x-my-credential",
			Patterns:    []string{"credential"},
			IsSensitive: true,
		},
		{
			Name:        "does not match custom pattern",
			Input:       "content-type",
			Patterns:    []string{"credential"},
			IsSensitive: false,
		},
		{
			Name:        "custom pattern overrides default keywords",
			Input:       "authorization",
			Patterns:    []string{"credential"},
			IsSensitive: false,
		},
		{
			Name:        "multiple patterns – first matches",
			Input:       "x-api-token",
			Patterns:    []string{"credential", "token"},
			IsSensitive: true,
		},
		{
			Name:        "multiple patterns – second matches",
			Input:       "x-api-credential",
			Patterns:    []string{"token", "credential"},
			IsSensitive: true,
		},
		{
			Name:        "multiple patterns – none match",
			Input:       "content-type",
			Patterns:    []string{"token", "credential"},
			IsSensitive: false,
		},
		{
			Name:        "empty pattern list falls back to default keywords",
			Input:       "authorization",
			Patterns:    []string{},
			IsSensitive: true,
		},
		{
			Name:        "short header with empty patterns falls back to default and is not sensitive",
			Input:       "xy",
			Patterns:    []string{},
			IsSensitive: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			isSensitive := IsSensitiveHeader(tc.Input, tc.Patterns...)
			if isSensitive != tc.IsSensitive {
				t.Errorf("IsSensitiveHeader(%q, %v): expected %v, got %v",
					tc.Input, tc.Patterns, tc.IsSensitive, isSensitive)
			}
		})
	}
}

func TestExtractTelemetryHeadersWithSensitivePatterns(t *testing.T) {
	testCases := []struct {
		Name              string
		Input             http.Header
		SensitivePatterns []string
		AllowedHeaders    []string
		Expected          [][]string
	}{
		{
			Name: "custom sensitive pattern masks matching header",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"X-My-Cred":    []string{"super-secret"},
				"X-Request-Id": []string{"req-123"},
			},
			SensitivePatterns: []string{"cred"},
			Expected: [][]string{
				{"content-type", "application/json"},
				{"x-my-cred", MaskString},
				{"x-request-id", "req-123"},
			},
		},
		{
			Name: "custom patterns override defaults – authorization not masked",
			Input: http.Header{
				"Authorization": []string{"Bearer token123"},
				"X-Api-Cred":    []string{"secret-value"},
			},
			SensitivePatterns: []string{"cred"},
			Expected: [][]string{
				{"authorization", "Bearer token123"},
				{"x-api-cred", MaskString},
			},
		},
		{
			Name: "custom patterns with allowed headers filter",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"X-My-Cred":    []string{"super-secret"},
				"Accept":       []string{"*/*"},
			},
			SensitivePatterns: []string{"cred"},
			AllowedHeaders:    []string{"Content-Type", "X-My-Cred"},
			Expected: [][]string{
				{"content-type", "application/json"},
				{"x-my-cred", MaskString},
			},
		},
		{
			Name: "nil sensitive patterns uses default keywords",
			Input: http.Header{
				"Authorization": []string{"Bearer abc"},
				"Content-Type":  []string{"application/json"},
			},
			SensitivePatterns: nil,
			Expected: [][]string{
				{"authorization", MaskString},
				{"content-type", "application/json"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got := ExtractTelemetryHeaders(tc.Input, tc.SensitivePatterns, tc.AllowedHeaders...)
			slices.SortFunc(got, func(a, b []string) int {
				return strings.Compare(a[0], b[0])
			})
			slices.SortFunc(tc.Expected, func(a, b []string) int {
				return strings.Compare(a[0], b[0])
			})

			if !reflect.DeepEqual(tc.Expected, got) {
				t.Errorf("expected: %v, got: %v", tc.Expected, got)
			}
		})
	}
}

func TestSplitHostPort(t *testing.T) {
	testCases := []struct {
		Name         string
		HostPort     string
		URLScheme    string
		ExpectedHost string
		ExpectedPort int
		ExpectError  bool
	}{
		{
			Name:         "host with port",
			HostPort:     "example.com:8080",
			URLScheme:    "",
			ExpectedHost: "example.com",
			ExpectedPort: 8080,
			ExpectError:  false,
		},
		{
			Name:         "host without port http",
			HostPort:     "example.com",
			URLScheme:    "http",
			ExpectedHost: "example.com",
			ExpectedPort: 80,
			ExpectError:  false,
		},
		{
			Name:         "host without port https",
			HostPort:     "example.com",
			URLScheme:    "https",
			ExpectedHost: "example.com",
			ExpectedPort: 443,
			ExpectError:  false,
		},
		{
			Name:         "IPv6 with port",
			HostPort:     "[::1]:8080",
			URLScheme:    "",
			ExpectedHost: "::1",
			ExpectedPort: 8080,
			ExpectError:  false,
		},
		{
			Name:         "IPv6 without port",
			HostPort:     "[::1]",
			URLScheme:    "http",
			ExpectedHost: "::1",
			ExpectedPort: 80,
			ExpectError:  false,
		},
		{
			Name:         "IPv6 with zone",
			HostPort:     "[fe80::1%lo0]:8080",
			URLScheme:    "",
			ExpectedHost: "fe80::1%lo0",
			ExpectedPort: 8080,
			ExpectError:  false,
		},
		{
			Name:         "port only",
			HostPort:     ":8080",
			URLScheme:    "",
			ExpectedHost: "",
			ExpectedPort: 8080,
			ExpectError:  false,
		},
		{
			Name:         "invalid IPv6 missing bracket",
			HostPort:     "[::1",
			URLScheme:    "",
			ExpectedHost: "",
			ExpectedPort: -1,
			ExpectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			host, port, err := SplitHostPort(tc.HostPort, tc.URLScheme)

			if tc.ExpectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if host != tc.ExpectedHost {
				t.Errorf("expected host '%s', got '%s'", tc.ExpectedHost, host)
			}

			if port != tc.ExpectedPort {
				t.Errorf("expected port %d, got %d", tc.ExpectedPort, port)
			}
		})
	}
}

func TestIsContentTypeDebuggable(t *testing.T) {
	testCases := []struct {
		Name        string
		ContentType string
		Expected    bool
	}{
		{
			Name:        "application/json",
			ContentType: "application/json",
			Expected:    true,
		},
		{
			Name:        "application/json with charset",
			ContentType: "application/json; charset=utf-8",
			Expected:    true,
		},
		{
			Name:        "text/plain",
			ContentType: "text/plain",
			Expected:    true,
		},
		{
			Name:        "text/html",
			ContentType: "text/html",
			Expected:    true,
		},
		{
			Name:        "application/xml",
			ContentType: "application/xml",
			Expected:    true,
		},
		{
			Name:        "multipart/form-data",
			ContentType: "multipart/form-data; boundary=----WebKitFormBoundary",
			Expected:    true,
		},
		{
			Name:        "image/png",
			ContentType: "image/png",
			Expected:    false,
		},
		{
			Name:        "application/octet-stream",
			ContentType: "application/octet-stream",
			Expected:    false,
		},
		{
			Name:        "video/mp4",
			ContentType: "video/mp4",
			Expected:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := IsContentTypeDebuggable(tc.ContentType)

			if result != tc.Expected {
				t.Errorf("expected %v, got %v", tc.Expected, result)
			}
		})
	}
}

func TestSetSpanHeaderAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")

	t.Run("sets header attributes on span", func(t *testing.T) {
		exporter.Reset()

		_, span := tracer.Start(context.Background(), "test-span")

		headers := map[string][]string{
			"content-type": {"application/json"},
			"accept":       {"application/json"},
			"user-agent":   {"test-agent"},
		}

		SetSpanHeaderAttributes(span, "http.request.header", headers)
		span.End()

		tp.ForceFlush(context.Background())

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span")
		}

		testSpan := spans[0]

		// Check that header attributes were set
		foundContentType := false
		foundAccept := false
		foundUserAgent := false

		for _, attr := range testSpan.Attributes {
			if string(attr.Key) == "http.request.header.content-type" {
				foundContentType = true
			}
			if string(attr.Key) == "http.request.header.accept" {
				foundAccept = true
			}
			if string(attr.Key) == "http.request.header.user-agent" {
				foundUserAgent = true
			}
		}

		if !foundContentType {
			t.Error("expected content-type header attribute")
		}
		if !foundAccept {
			t.Error("expected accept header attribute")
		}
		if !foundUserAgent {
			t.Error("expected user-agent header attribute")
		}
	})

	t.Run("excludes tracing headers", func(t *testing.T) {
		exporter.Reset()

		_, span := tracer.Start(context.Background(), "test-span")

		headers := map[string][]string{
			"content-type": {"application/json"},
			"traceparent":  {"00-trace-id-span-id-01"},
			"baggage":      {"key=value"},
		}

		SetSpanHeaderAttributes(span, "http.request.header", headers)
		span.End()

		tp.ForceFlush(context.Background())

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span")
		}

		testSpan := spans[0]

		// Check that tracing headers were excluded
		for _, attr := range testSpan.Attributes {
			if string(attr.Key) == "http.request.header.traceparent" {
				t.Error("traceparent header should be excluded")
			}
			if string(attr.Key) == "http.request.header.baggage" {
				t.Error("baggage header should be excluded")
			}
		}
	})

	t.Run("respects allowed headers list", func(t *testing.T) {
		exporter.Reset()

		_, span := tracer.Start(context.Background(), "test-span")

		headers := map[string][]string{
			"content-type": {"application/json"},
			"accept":       {"application/json"},
			"user-agent":   {"test-agent"},
		}

		SetSpanHeaderAttributes(span, "http.request.header", headers, "content-type", "accept")
		span.End()

		tp.ForceFlush(context.Background())

		spans := exporter.GetSpans()
		if len(spans) == 0 {
			t.Fatal("expected at least one span")
		}

		testSpan := spans[0]

		foundContentType := false
		foundAccept := false
		foundUserAgent := false

		for _, attr := range testSpan.Attributes {
			if string(attr.Key) == "http.request.header.content-type" {
				foundContentType = true
			}
			if string(attr.Key) == "http.request.header.accept" {
				foundAccept = true
			}
			if string(attr.Key) == "http.request.header.user-agent" {
				foundUserAgent = true
			}
		}

		if !foundContentType {
			t.Error("expected content-type header attribute")
		}
		if !foundAccept {
			t.Error("expected accept header attribute")
		}
		if foundUserAgent {
			t.Error("user-agent should not be included when not in allowed list")
		}
	})
}

func TestNormalizeStrings(t *testing.T) {
	t.Run("converts strings to lowercase", func(t *testing.T) {
		input := []string{"Hello", "WORLD", "TeSt"}
		expected := []string{"hello", "world", "test"}

		result := NormalizeStrings(input)

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
		result := NormalizeStrings([]string{})
		if len(result) != 0 {
			t.Errorf("expected empty slice, got length %d", len(result))
		}
	})
}
