package otelutils

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"slices"
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

	err := level.UnmarshalText([]byte(logLevel))
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

// GetLoggerFromContext gets the logger instance from context.
func GetLoggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	value := ctx.Value(LoggerContextKey)
	if value != nil {
		if logger, ok := value.(*slog.Logger); ok {
			return logger, true
		}
	}

	return nil, false
}

// GetLogger gets the logger instance from context.
// If not exists, return the default logger.
func GetLogger(ctx context.Context) *slog.Logger {
	logger, ok := GetLoggerFromContext(ctx)
	if ok {
		return logger
	}

	return slog.Default()
}

// DebugLogger wraps the logger with debug information.
// It won't print if the log level isn't debug.
type DebugLogger struct {
	logger     *slog.Logger
	attributes []slog.Attr
	debug      bool
}

// NewDebugLogger creates a debug logger from context.
func NewDebugLogger(ctx context.Context, logger *slog.Logger) *DebugLogger {
	if logger == nil {
		logger = GetLogger(ctx)
	}

	return &DebugLogger{
		logger: logger,
		debug:  logger.Enabled(ctx, slog.LevelDebug),
	}
}

// Grow grows the capacity of the attributes.
func (dl *DebugLogger) Grow(minCap, maxCap int) {
	capacity := minCap

	if dl.debug && capacity < maxCap {
		capacity = maxCap
	}

	dl.attributes = slices.Grow(dl.attributes, capacity)
}

// AddAttributes add attributes to the logger.
func (dl *DebugLogger) AddAttributes(attrs ...slog.Attr) {
	dl.attributes = append(dl.attributes, attrs...)
}

// AddDebugAttributes add attributes to the logger.
// If the log level isn't debug, those attributes will be discarded.
func (dl *DebugLogger) AddDebugAttributes(attrs ...slog.Attr) {
	if !dl.debug {
		return
	}

	dl.attributes = append(dl.attributes, attrs...)
}

// Log prints the message log by level.
func (dl *DebugLogger) Log(level slog.Level, message string) {
	if dl.debug && level > slog.LevelDebug {
		return
	}

	dl.logger.LogAttrs(context.Background(), level, message, dl.attributes...)
}
