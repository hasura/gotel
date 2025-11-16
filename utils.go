package gotel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var sensitiveHeaderRegex = regexp.MustCompile(`auth|key|secret|token|password`)

const (
	contentTypeJSON   = "application/json"
	contentTypeHeader = "Content-Type"
)

var excludedSpanHeaderAttributes = map[string]bool{
	"baggage":       true,
	"traceparent":   true,
	"traceresponse": true,
	"tracestate":    true,
	"x-b3-sampled":  true,
	"x-b3-spanid":   true,
	"x-b3-traceid":  true,
}

// SetSpanHeaderAttributes sets header attributes to the otel span.
func SetSpanHeaderAttributes(
	span trace.Span,
	prefix string,
	headers http.Header,
	allowedHeaders ...string,
) {
	allowedHeadersLength := len(allowedHeaders)

	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if (allowedHeadersLength == 0 && !excludedSpanHeaderAttributes[lowerKey]) ||
			slices.Contains(allowedHeaders, strings.ToLower(key)) {
			span.SetAttributes(
				attribute.StringSlice(fmt.Sprintf("%s.%s", prefix, strings.ToLower(key)), values),
			)
		}
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
		return input[0:2] + strings.Repeat("*", 8) + fmt.Sprintf("(%d)", inputLength)
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

func getRequestID(r *http.Request) string {
	requestID := r.Header.Get("x-request-id")
	if requestID != "" {
		return requestID
	}

	spanContext := trace.SpanContextFromContext(r.Context())
	if spanContext.HasTraceID() {
		return spanContext.TraceID().String()
	}

	return uuid.NewString()
}

func debugRequestBody(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)

		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusUnprocessableEntity)

		err := enc.Encode(map[string]any{
			"title":  "Failed to read request body",
			"detail": err.Error(),
		})
		if err != nil {
			logger.Error("failed to write response: " + err.Error())
		}

		return "", err
	}

	bodyStr := string(bodyBytes)

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyStr, nil
}

func writeResponseJSON(w http.ResponseWriter, statusCode int, body any, logger *slog.Logger) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(statusCode)

	err := enc.Encode(body)
	if err != nil {
		logger.Error("failed to write response: " + err.Error())
	}
}

func isContentTypeDebuggable(contentType string) bool {
	return strings.HasPrefix(contentType, contentTypeJSON) ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "multipart/form-data")
}

func toLowerStrings(values []string) []string {
	results := make([]string, len(values))

	for i, item := range values {
		results[i] = strings.ToLower(item)
	}

	return results
}
