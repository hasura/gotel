package otelutils

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "context value " + k.name
}

// LoggerContextKey is the global context key constant.
var LoggerContextKey = &contextKey{"LogEntry"}

// NewContextWithLogger creates a new context with a logger set.
func NewContextWithLogger(parentContext context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(parentContext, LoggerContextKey, logger)
}

// NewJSONLogger creates a JSON logger from a log level string.
func NewJSONLogger(logLevel string) (*slog.Logger, slog.Level, error) {
	level := slog.LevelInfo

	err := level.UnmarshalText([]byte(strings.ToUpper(logLevel)))
	if err != nil {
		return nil, level, err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	return logger, level, nil
}

// NewHeaderLogGroupAttrs converts HTTP header to slog attributes.
func NewHeaderLogGroupAttrs(key string, headers http.Header) slog.Attr {
	headerAttrs := make([]slog.Attr, 0, len(headers))

	for name, values := range headers {
		switch len(values) {
		case 0:
		case 1:
			headerAttrs = append(headerAttrs, slog.String(name, values[0]))
		default:
			headerAttrs = append(headerAttrs, slog.String(name, strings.Join(values, ", ")))
		}
	}

	return slog.GroupAttrs(key, headerAttrs...)
}
