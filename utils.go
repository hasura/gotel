package gotel

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

const (
	contentTypeJSON   = "application/json"
	contentTypeHeader = "Content-Type"
)

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

func toLowerStrings(values []string) []string {
	results := make([]string, len(values))

	for i, item := range values {
		results[i] = strings.Map(unicode.ToLower, item)
	}

	return results
}
