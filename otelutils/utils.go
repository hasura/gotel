// Package otelutils contain reusable utilities for OpenTelemetry attributes.
package otelutils

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserVisibilityAttribute is the attribute to display on the Trace view.
var UserVisibilityAttribute = attribute.String("internal.visibility", "user")

var sensitiveHeaderRegex = regexp.MustCompile(`auth|key|secret|token|password`)

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

// SplitHostPort splits a network address hostport of the form "host",
// "host%zone", "[host]", "[host%zone]", "host:port", "host%zone:port",
// "[host]:port", "[host%zone]:port", or ":port" into host or host%zone and
// port.
//
// An empty host is returned if it is not provided or unparsable. A negative
// port is returned if it is not provided or unparsable.
func SplitHostPort(hostport string) (string, int, error) {
	port := -1

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

	p, err := strconv.ParseUint(pStr, 10, 16)
	if err != nil {
		return "", port, err
	}

	return host, int(p), err
}
