// Package gotel is a reusable library for setting up OpenTelemetry exporters in Go with configurations.
package gotel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelPrometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/log/global"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	traceapi "go.opentelemetry.io/otel/trace"
)

const (
	otlpDefaultHTTPPort = "4318"
)

// OTelExporters contains outputs of OpenTelemetry exporters.
type OTelExporters struct {
	Tracer   *Tracer
	Meter    metricapi.Meter
	Logger   *slog.Logger
	Shutdown func(context.Context) error
}

// SetupOTelExporters set up OpenTelemetry exporters from configuration.
func SetupOTelExporters(
	ctx context.Context,
	config *OTLPConfig,
	serviceVersion string,
	logger *slog.Logger,
) (*OTelExporters, error) {
	otel.SetLogger(logr.FromSlogHandler(logger.Handler()))

	otelDisabled := os.Getenv("OTEL_SDK_DISABLED") == "true"

	// Set up resource.
	res := newResource(config.ServiceName, serviceVersion)

	traceProvider, err := setupOTelTraceProvider(ctx, config, res, otelDisabled)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(traceProvider)

	meterProvider, err := setupOTelMetricsProvider(ctx, config, res, otelDisabled)
	if err != nil {
		return nil, err
	}

	// configure metrics exporter
	loggerProvider, err := newLoggerProvider(ctx, config, otelDisabled, res)
	if err != nil {
		return nil, err
	}

	global.SetLoggerProvider(loggerProvider)

	shutdownFunc := func(ctx context.Context) error {
		errorMsgs := []error{}

		err := traceProvider.Shutdown(ctx)
		if err != nil {
			errorMsgs = append(errorMsgs, err)
		}

		meterErr := meterProvider.Shutdown(ctx)
		if meterErr != nil {
			errorMsgs = append(errorMsgs, meterErr)
		}

		loggerErr := loggerProvider.Shutdown(ctx)
		if loggerErr != nil {
			errorMsgs = append(errorMsgs, loggerErr)
		}

		if len(errorMsgs) > 0 {
			return errors.Join(errorMsgs...)
		}

		return nil
	}

	otelLogger := slog.New(createLogHandler(config.ServiceName, logger, loggerProvider))
	state := &OTelExporters{
		Tracer: &Tracer{
			traceProvider.Tracer(config.ServiceName, traceapi.WithSchemaURL(semconv.SchemaURL)),
		},
		Meter: meterProvider.Meter(
			config.ServiceName,
			metricapi.WithSchemaURL(semconv.SchemaURL),
		),
		Logger:   otelLogger,
		Shutdown: shutdownFunc,
	}

	return state, err
}

func setupOTelTraceProvider(
	ctx context.Context,
	config *OTLPConfig,
	resources *resource.Resource,
	otelDisabled bool,
) (*trace.TracerProvider, error) {
	tracesEndpoint := config.OtlpTracesEndpoint
	if tracesEndpoint == "" && config.OtlpEndpoint != "" {
		tracesEndpoint = config.OtlpEndpoint + "/v1/traces"
	}

	if otelDisabled || tracesEndpoint == "" {
		return trace.NewTracerProvider(trace.WithResource(resources)), nil
	}

	endpoint, protocol, insecure, err := parseOTLPEndpoint(
		tracesEndpoint,
		config.GetOTLPTracesProtocol(),
		getDefaultPtr(config.OtlpTracesInsecure, config.OtlpInsecure),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP traces endpoint: %w", err)
	}

	compressorStr, compressorInt, err := parseOTLPCompression(
		config.GetOTLPTracesCompression(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP traces compression: %w", err)
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	var traceExporter *otlptrace.Exporter

	if protocol == OTLPProtocolGRPC {
		options := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithCompressor(string(compressorStr)),
		}

		if insecure {
			options = append(options, otlptracegrpc.WithInsecure())
		}

		traceExporter, err = otlptracegrpc.New(ctx, options...)
		if err != nil {
			return nil, err
		}

		return trace.NewTracerProvider(
			trace.WithResource(resources),
			trace.WithBatcher(traceExporter),
		), nil
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithCompression(otlptracehttp.Compression(compressorInt)),
	}

	if insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	traceExporter, err = otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	return trace.NewTracerProvider(
		trace.WithResource(resources),
		trace.WithBatcher(traceExporter),
	), nil
}

func setupOTelMetricsProvider(
	ctx context.Context,
	config *OTLPConfig,
	resources *resource.Resource,
	otelDisabled bool,
) (*metric.MeterProvider, error) {
	// configure metrics exporter
	metricsExporterType := config.GetMetricsExporter()
	metricOptions := []metric.Option{metric.WithResource(resources)}

	var err error

	if config.DisableGoMetrics != nil && !*config.DisableGoMetrics {
		// disable default process and go collector metrics
		prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		prometheus.Unregister(collectors.NewGoCollector())
	}

	switch metricsExporterType {
	case OTELMetricsExporterPrometheus:
		// The exporter embeds a default OpenTelemetry Reader and
		// implements prometheus.Collector, allowing it to be used as
		// both a Reader and Collector.
		prometheusExporter, err := otelPrometheus.New()
		if err != nil {
			return nil, err
		}

		metricOptions = append(metricOptions, metric.WithReader(prometheusExporter))
	case OTELMetricsExporterOTLP:
		if otelDisabled {
			break
		}

		metricOptions, err = setupMetricExporterOTLP(ctx, config, metricOptions)
		if err != nil {
			return nil, err
		}
	case OTELMetricsExporterNone:
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidOTELMetricExporterType, metricsExporterType)
	}

	meterProvider := metric.NewMeterProvider(metricOptions...)
	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}

func setupMetricExporterOTLP(
	ctx context.Context,
	config *OTLPConfig,
	metricOptions []metric.Option,
) ([]metric.Option, error) {
	metricsEndpoint := config.OtlpMetricsEndpoint
	if metricsEndpoint == "" && config.OtlpEndpoint != "" {
		metricsEndpoint = config.OtlpEndpoint + "/v1/metrics"
	}

	if metricsEndpoint == "" {
		return nil, errMetricsOTLPEndpointRequired
	}

	endpoint, protocol, insecure, err := parseOTLPEndpoint(
		metricsEndpoint,
		config.GetOTLPMetricsProtocol(),
		getDefaultPtr(config.OtlpMetricsInsecure, config.OtlpInsecure),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP metrics endpoint: %w", err)
	}

	compressorStr, compressorInt, err := parseOTLPCompression(
		config.GetOTLPMetricsCompression(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OTLP metrics compression: %w", err)
	}

	if protocol == OTLPProtocolGRPC {
		options := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithCompressor(string(compressorStr)),
		}

		if insecure {
			options = append(options, otlpmetricgrpc.WithInsecure())
		}

		metricExporter, err := otlpmetricgrpc.New(ctx, options...)
		if err != nil {
			return nil, err
		}

		return append(
			metricOptions,
			metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		), nil
	}

	options := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(endpoint),
		otlpmetrichttp.WithCompression(otlpmetrichttp.Compression(compressorInt)),
	}
	if insecure {
		options = append(options, otlpmetrichttp.WithInsecure())
	}

	metricExporter, err := otlpmetrichttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	metricOptions = append(
		metricOptions,
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
	)

	return metricOptions, nil
}

func newResource(serviceName, serviceVersion string) *resource.Resource {
	hostname, _ := os.Hostname()
	attrs := append(
		resource.Environment().Attributes(),
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.HostNameKey.String(hostname),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersion(sdk.Version()),
		semconv.ProcessPIDKey.Int64(int64(os.Getpid())),
	)

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	)
}

func parseOTLPEndpoint(
	endpoint string,
	protocol OTLPProtocol,
	insecurePtr *bool,
) (string, OTLPProtocol, bool, error) {
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	uri, err := url.Parse(endpoint)
	if err != nil {
		return "", "", false, err
	}

	insecure := (insecurePtr != nil && *insecurePtr) || uri.Scheme == "http"
	host := uri.Host

	if uri.Port() == "" {
		port := 443
		if insecure {
			port = 80
		}

		host = fmt.Sprintf("%s:%d", uri.Hostname(), port)
	}

	switch protocol {
	case OTLPProtocolGRPC:
		return host, protocol, insecure, nil
	case OTLPProtocolHTTPProtobuf:
		return endpoint, protocol, insecure, nil
	case "":
		// auto detect via default OTLP port
		if uri.Port() == otlpDefaultHTTPPort {
			return host, protocol, insecure, nil
		}

		return host, OTLPProtocolGRPC, insecure, nil
	default:
		return "", "", false, fmt.Errorf("%w: %s", errInvalidOTLPProtocol, protocol)
	}
}

func parseOTLPCompression(input OTLPCompressionType) (OTLPCompressionType, int, error) {
	switch input {
	case OTLPCompressionGzip, "":
		return OTLPCompressionGzip, int(otlptracehttp.GzipCompression), nil
	case OTLPCompressionNone:
		return input, int(otlptracehttp.NoCompression), nil
	default:
		return "", 0, errInvalidOTLPCompressionType
	}
}
