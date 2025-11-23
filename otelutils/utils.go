// Package otelutils contain reusable utilities for OpenTelemetry attributes.
package otelutils

import (
	"errors"
	"fmt"
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

var sensitiveKeywords = map[byte]string{
	'a': "uth",
	'k': "ey",
	's': "ecret",
	't': "oken",
	'p': "assword",
}

var errInvalidHostPort = errors.New("invalid host port")

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
			(allowedHeadersLength > 0 && slices.Contains(allowedHeaders, lowerKey)) {
			span.SetAttributes(
				attribute.StringSlice(fmt.Sprintf("%s.%s", prefix, lowerKey), values),
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
				result.Set(strings.ToLower(key), MaskString)
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

		if IsSensitiveHeader(key) {
			result[key] = []string{MaskString}

			continue
		}

		result[key] = headers
	}

	return result
}

// IsSensitiveHeader checks if the header name is sensitive.
func IsSensitiveHeader(name string) bool {
	if len(name) < 3 {
		return false
	}

	lowerBytes := make([]byte, len(name))

	for i := range name {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}

		lowerBytes[i] = c
	}

	for i := range len(lowerBytes) - 2 {
		lc := lowerBytes[i]

		keyword, ok := sensitiveKeywords[lc]
		if !ok {
			continue
		}

		j := 0
		keywordLength := len(keyword)

		for ; j < keywordLength; j++ {
			if lowerBytes[i+j+1] != keyword[j] {
				break
			}
		}

		if j == keywordLength {
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
