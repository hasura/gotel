package gotel

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	traceapi "go.opentelemetry.io/otel/trace"
)

var sensitiveHeaderRegex = regexp.MustCompile(`auth|key|secret|token`)

// SetSpanHeaderAttributes sets header attributes to the otel span.
func SetSpanHeaderAttributes(
	span traceapi.Span,
	prefix string,
	httpHeaders http.Header,
	allowedHeaders ...string,
) {
	headers := NewTelemetryHeaders(httpHeaders, allowedHeaders...)

	for key, values := range headers {
		span.SetAttributes(attribute.StringSlice(prefix+strings.ToLower(key), values))
	}
}

// NewTelemetryHeaders creates a new header map with sensitive values masked.
func NewTelemetryHeaders(httpHeaders http.Header, allowedHeaders ...string) http.Header {
	result := http.Header{}

	if len(allowedHeaders) > 0 {
		for _, key := range allowedHeaders {
			value := httpHeaders.Get(key)

			if value == "" {
				continue
			}

			if IsSensitiveHeader(key) {
				result.Set(strings.ToLower(key), MaskString(value))
			} else {
				result.Set(strings.ToLower(key), value)
			}
		}

		return result
	}

	for key, headers := range httpHeaders {
		if len(headers) == 0 {
			continue
		}

		values := headers
		if IsSensitiveHeader(key) {
			values = make([]string, len(headers))
			for i, header := range headers {
				values[i] = MaskString(header)
			}
		}

		result[key] = values
	}

	return result
}

// IsSensitiveHeader checks if the header name is sensitive.
func IsSensitiveHeader(name string) bool {
	return sensitiveHeaderRegex.MatchString(strings.ToLower(name))
}

// MaskString masks the string value for security.
func MaskString(input string) string {
	inputLength := len(input)

	switch {
	case inputLength <= 6:
		return strings.Repeat("*", inputLength)
	case inputLength < 12:
		return input[0:1] + strings.Repeat("*", inputLength-1)
	default:
		return input[0:3] + strings.Repeat("*", 7) + fmt.Sprintf("(%d)", inputLength)
	}
}

// returns the value or default one if value is empty.
func getDefault[T comparable](value T, defaultValue T) T {
	var empty T

	if value == empty {
		return defaultValue
	}

	return value
}

// returns the first pointer or default one if GetDefaultPtr is nil.
func getDefaultPtr[T any](value *T, defaultValue *T) *T {
	if value == nil {
		return defaultValue
	}

	return value
}
