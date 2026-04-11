// Package otelutils contain reusable utilities for OpenTelemetry attributes.
package otelutils

import (
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MaskString is the constant string for masking sensitive data.
const MaskString = "[REDACTED]"

// UserVisibilityAttribute is the attribute to display on the Trace view.
var UserVisibilityAttribute = attribute.String("internal.visibility", "user")

var excludedSpanHeaderAttributes = map[string]bool{
	"baggage":           true,
	"traceparent":       true,
	"traceresponse":     true,
	"tracestate":        true,
	"x-b3-sampled":      true,
	"x-b3-spanid":       true,
	"x-b3-traceid":      true,
	"x-b3-parentspanid": true,
	"x-b3-flags":        true,
	"b3":                true,
}

var sensitiveKeywords = []string{
	"auth",
	"password",
	"key",
	"secret",
	"token",
}

var errInvalidHostPort = errors.New("invalid host port")

// SetSpanHeaderAttributes sets header attributes to the otel span.
func SetSpanHeaderAttributes(
	span trace.Span,
	prefix string,
	headers map[string][]string,
	allowedHeaders ...string,
) {
	allowedHeadersLength := len(allowedHeaders)
	maxLength := len(headers)

	if allowedHeadersLength > 0 && allowedHeadersLength < maxLength {
		maxLength = allowedHeadersLength
	}

	attrs := make([]attribute.KeyValue, 0, maxLength)

	for key, values := range headers {
		if (allowedHeadersLength == 0 && !excludedSpanHeaderAttributes[key]) ||
			(allowedHeadersLength > 0 && slices.Contains(allowedHeaders, key)) {
			attrs = append(attrs, attribute.StringSlice(prefix+"."+key, values))
		}
	}

	span.SetAttributes(attrs...)
}

// SetSpanHeaderMatrixAttributes sets header attributes from a matrix to the otel span.
func SetSpanHeaderMatrixAttributes(
	span trace.Span,
	prefix string,
	headers [][]string,
	allowedHeaders ...string,
) {
	allowedHeadersLength := len(allowedHeaders)
	maxLength := len(headers)

	if allowedHeadersLength > 0 && allowedHeadersLength < maxLength {
		maxLength = allowedHeadersLength
	}

	attrs := make([]attribute.KeyValue, 0, maxLength)

	for _, values := range headers {
		if (allowedHeadersLength == 0 && !excludedSpanHeaderAttributes[values[0]]) ||
			(allowedHeadersLength > 0 && slices.Contains(allowedHeaders, values[0])) {
			attrs = append(attrs, attribute.StringSlice(prefix+"."+values[0], values[1:]))
		}
	}

	span.SetAttributes(attrs...)
}

// ExtractTelemetryHeaders creates matrix with sensitive values masked.
func ExtractTelemetryHeaders(
	httpHeaders http.Header,
	sensitivePatterns []string,
	allowedHeaders ...string,
) [][]string {
	if len(httpHeaders) == 0 {
		return nil
	}

	if len(allowedHeaders) > 0 {
		result := make([][]string, 0, len(allowedHeaders))

		for _, key := range allowedHeaders {
			value := httpHeaders.Get(key)
			if value == "" {
				continue
			}

			lowerKey := strings.ToLower(key)
			isSensitive := IsSensitiveHeader(lowerKey, sensitivePatterns...)

			if isSensitive {
				result = append(result, []string{lowerKey, MaskString})
			} else {
				result = append(result, []string{lowerKey, value})
			}
		}

		return result
	}

	result := make([][]string, 0, len(httpHeaders))

	for key, headers := range httpHeaders {
		if len(headers) == 0 {
			continue
		}

		lowerKey := strings.ToLower(key)
		isSensitive := IsSensitiveHeader(lowerKey, sensitivePatterns...)

		if isSensitive {
			result = append(result, []string{lowerKey, MaskString})
		} else {
			row := make([]string, len(headers)+1)

			row[0] = lowerKey
			copy(row[1:], headers)

			result = append(result, row)
		}
	}

	return result
}

// IsSensitiveHeader checks if the header name is sensitive.
func IsSensitiveHeader(name string, patterns ...string) bool {
	if len(patterns) == 0 {
		if len(name) < 3 {
			return false
		}

		patterns = sensitiveKeywords
	}

	for _, word := range patterns {
		if word == "" {
			continue
		}

		if strings.Contains(name, word) {
			return true
		}
	}

	return false
}

// SplitHostPort splits a network address hostport of the form "host",
// "host%zone", "[host]", "[host%zone]", "host:port", "host%zone:port",
// "[host]:port", "[host%zone]:port", or ":port" into host or host%zone and
// port.
//
// An empty host is returned if it is not provided or unparsable. A negative
// port is returned if it is not provided or unparsable.
func SplitHostPort(hostport string, urlScheme string) (string, int, error) {
	port := -1

	switch urlScheme {
	case "http":
		port = 80
	case "https":
		port = 443
	}

	// Check if the host is IPv6, and set the default HTTP(S) port if the input string does not have the port.
	if strings.HasPrefix(hostport, "[") {
		addrEnd := strings.LastIndex(hostport, "]")
		if addrEnd < 0 {
			// Invalid hostport.
			return "", port, errInvalidHostPort
		}

		if i := strings.LastIndex(hostport[addrEnd:], ":"); i < 0 {
			host := hostport[1:addrEnd]

			return host, port, nil
		}
	} else {
		if i := strings.LastIndex(hostport, ":"); i < 0 {
			return hostport, port, nil
		}
	}

	host, pStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return host, port, err
	}

	p, err := strconv.Atoi(pStr)
	if err != nil {
		return "", port, err
	}

	return host, p, err
}

// IsContentTypeDebuggable checks if the content type can be debugged.
func IsContentTypeDebuggable(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "multipart/form-data")
}

// NormalizeStrings normalize input strings to the standard format for telemetry evaluation.
func NormalizeStrings(values []string) []string {
	results := make([]string, len(values))

	for i, item := range values {
		results[i] = strings.ToLower(strings.TrimSpace(item))
	}

	return results
}
