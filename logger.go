package gotel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
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

var loggerContextKey = contextKey{"LogEntry"}

// LogHandler wraps slog logger with the OpenTelemetry logs exporter handler.
type LogHandler struct {
	otelHandler slog.Handler
	stdHandler  slog.Handler
}

func createLogHandler(
	serviceName string,
	logger *slog.Logger,
	provider *log.LoggerProvider,
) slog.Handler {
	options := []otelslog.Option{}
	if provider != nil {
		options = append(options, otelslog.WithLoggerProvider(provider))
	}

	otelHandler := otelslog.NewHandler(serviceName, options...)
	loggerHandler := logger.Handler()

	return LogHandler{
		otelHandler: otelHandler,
		stdHandler:  loggerHandler,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (l LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return l.stdHandler.Enabled(ctx, level)
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
func (l LogHandler) Handle(ctx context.Context, record slog.Record) error {
	_ = l.stdHandler.Handle(ctx, record)

	return l.otelHandler.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (l LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return LogHandler{
		otelHandler: l.otelHandler.WithAttrs(attrs),
		stdHandler:  l.stdHandler.WithAttrs(attrs),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (l LogHandler) WithGroup(name string) slog.Handler {
	return LogHandler{
		otelHandler: l.otelHandler.WithGroup(name),
		stdHandler:  l.stdHandler.WithGroup(name),
	}
}

// create OpenTelemetry logger provider.
func newLoggerProvider(
	ctx context.Context,
	config *OTLPConfig,
	otelDisabled bool,
	res *resource.Resource,
) (*log.LoggerProvider, error) {
	logsEndpoint := getDefault(config.OtlpLogsEndpoint, config.OtlpEndpoint)
	if otelDisabled || config.LogsExporter != "otlp" || logsEndpoint == "" {
		return log.NewLoggerProvider(), nil
	}

	endpoint, protocol, insecure, err := parseOTLPEndpoint(
		logsEndpoint,
		getDefault(config.OtlpLogsProtocol, config.OtlpProtocol),
		getDefaultPtr(config.OtlpLogsInsecure, config.OtlpInsecure),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP logs endpoint: %w", err)
	}

	compressorStr, compressorInt, err := parseOTLPCompression(
		getDefault(config.OtlpLogsCompression, config.OtlpCompression),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP logs compression: %w", err)
	}

	opts := []log.LoggerProviderOption{log.WithResource(res)}

	if protocol == OTLPProtocolGRPC {
		options := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(endpoint),
			otlploggrpc.WithCompressor(compressorStr),
		}

		if insecure {
			options = append(options, otlploggrpc.WithInsecure())
		}

		logExporter, err := otlploggrpc.New(ctx, options...)
		if err != nil {
			return nil, err
		}

		opts = append(opts, log.WithProcessor(log.NewBatchProcessor(logExporter)))

		return log.NewLoggerProvider(opts...), nil
	}

	options := []otlploghttp.Option{
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithCompression(otlploghttp.Compression(compressorInt)),
	}

	if insecure {
		options = append(options, otlploghttp.WithInsecure())
	}

	logExporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, log.WithProcessor(log.NewBatchProcessor(logExporter)))

	return log.NewLoggerProvider(opts...), nil
}

// GetLogger gets the logger instance from context.
func GetLogger(ctx context.Context) *slog.Logger {
	logger, _ := getLogger(ctx)

	return logger
}

// GetRequestLogger get the logger from the an http request.
func GetRequestLogger(r *http.Request) *slog.Logger {
	ctx := r.Context()
	logger, present := getLogger(ctx)

	if present {
		return logger
	}

	requestID := getRequestID(r)

	return logger.With(slog.String("request_id", requestID))
}

// NewContextWithLogger creates a new context with a logger set.
func NewContextWithLogger(parentContext context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(parentContext, loggerContextKey, logger)
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

func getLogger(ctx context.Context) (*slog.Logger, bool) {
	value := ctx.Value(loggerContextKey)
	if value != nil {
		if logger, ok := value.(*slog.Logger); ok {
			return logger, true
		}
	}

	return slog.New(createLogHandler("hasura-ndc-go", slog.Default(), nil)), false
}
