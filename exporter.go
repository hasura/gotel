package gotel

import "errors"

// OTLPCompressionType represents the compression type enum for OTLP.
type OTLPCompressionType string

const (
	// OTLPCompressionNone is the enum that disables compression.
	OTLPCompressionNone OTLPCompressionType = "none"
	// OTLPCompressionGzip is the enum that enable compression of gzip algorithm.
	OTLPCompressionGzip OTLPCompressionType = "gzip"
)

// OTLPProtocol represents the OTLP protocol enum.
type OTLPProtocol string

const (
	// OTLPProtocolGRPC represents the gRPC OTLP protocol enum.
	OTLPProtocolGRPC OTLPProtocol = "grpc"
	// OTLPProtocolHTTPProtobuf represents the HTTP Protobuf OTLP protocol enum.
	OTLPProtocolHTTPProtobuf OTLPProtocol = "http/protobuf"
)

// OTELMetricsExporterType defines the type of OpenTelemetry metrics exporter.
type OTELMetricsExporterType string

const (
	// OTELMetricsExporterNone represents a enum that disables the metrics exporter.
	OTELMetricsExporterNone OTELMetricsExporterType = "none"
	// OTELMetricsExporterOTLP represents a enum that enables the metrics exporter via OTLP protocol.
	OTELMetricsExporterOTLP OTELMetricsExporterType = "otlp"
	// OTELMetricsExporterPrometheus represents a enum that enables the metrics exporter via Prometheus.
	OTELMetricsExporterPrometheus OTELMetricsExporterType = "prometheus"
)

// OTELLogsExporterType defines the type of OpenTelemetry logs exporter.
type OTELLogsExporterType string

const (
	// OTELLogsExporterNone represents a enum that disables the logs exporter.
	OTELLogsExporterNone OTELLogsExporterType = "none"
	// OTELLogsExporterOTLP represents a enum that enables the logs exporter via OTLP protocol.
	OTELLogsExporterOTLP OTELLogsExporterType = "otlp"
)

var (
	errInvalidOTLPCompressionType = errors.New(
		"invalid OTLP compression type, accept none, gzip only",
	)
	errInvalidOTELMetricExporterType = errors.New("invalid OTEL metrics exporter type")
	errInvalidOTLPProtocol           = errors.New("invalid OTLP protocol")
	errMetricsOTLPEndpointRequired   = errors.New("OTLP endpoint is required for metrics exporter")
)

// OTLPConfig contains configuration for OpenTelemetry exporter.
type OTLPConfig struct {
	// OpenTelemetry service name.
	ServiceName string `json:"serviceName,omitempty" yaml:"serviceName,omitempty" env:"OTEL_SERVICE_NAME" help:"OpenTelemetry service name."`
	// OTLP receiver endpoint that is set as default for all types.
	OtlpEndpoint string `json:"otlpEndpoint,omitempty" yaml:"otlpEndpoint,omitempty" env:"OTEL_EXPORTER_OTLP_ENDPOINT" help:"OTLP receiver endpoint that is set as default for all types."`
	// OTLP receiver endpoint for traces exporter.
	OtlpTracesEndpoint string `json:"otlpTracesEndpoint,omitempty" yaml:"otlpTracesEndpoint,omitempty" env:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT" help:"OTLP receiver endpoint for traces."`
	// OTLP receiver endpoint for metrics exporter.
	OtlpMetricsEndpoint string `json:"otlpMetricsEndpoint,omitempty" yaml:"otlpMetricsEndpoint,omitempty" env:"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT" help:"OTLP receiver endpoint for metrics."`
	// OTLP receiver endpoint for logs exporter.
	OtlpLogsEndpoint string `json:"otlpLogsEndpoint,omitempty" yaml:"otlpLogsEndpoint,omitempty" env:"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT" help:"OTLP receiver endpoint for logs."`
	// Disable TLS for OpenTelemetry exporters.
	OtlpInsecure *bool `json:"otlpInsecure,omitempty" yaml:"otlpInsecure,omitempty" env:"OTEL_EXPORTER_OTLP_INSECURE" help:"Disable TLS for OpenTelemetry exporters."`
	// Disable TLS for OpenTelemetry traces exporter.
	OtlpTracesInsecure *bool `json:"otlpTracesInsecure,omitempty" yaml:"otlpTracesInsecure,omitempty" env:"OTEL_EXPORTER_OTLP_TRACES_INSECURE" help:"Disable TLS for OpenTelemetry traces exporter."`
	// Disable TLS for OpenTelemetry metrics exporter.
	OtlpMetricsInsecure *bool `json:"otlpMetricsInsecure,omitempty" yaml:"otlpMetricsInsecure,omitempty" env:"OTEL_EXPORTER_OTLP_METRICS_INSECURE" help:"Disable TLS for OpenTelemetry metrics exporter."`
	// Disable TLS for OpenTelemetry logs exporter.
	OtlpLogsInsecure *bool `json:"otlpLogsInsecure,omitempty" yaml:"otlpLogsInsecure,omitempty" env:"OTEL_EXPORTER_OTLP_LOGS_INSECURE" help:"Disable TLS for OpenTelemetry logs exporter."`
	// OTLP receiver protocol for all exporters. Default is grpc.
	OtlpProtocol OTLPProtocol `json:"otlpProtocol,omitempty" yaml:"otlpProtocol,omitempty" env:"OTEL_EXPORTER_OTLP_PROTOCOL" enum:"grpc,http/protobuf" jsonschema:"enum=grpc,enum=http/protobuf" help:"OTLP receiver protocol for all exporters. Default is grpc"`
	// OTLP receiver protocol for traces.
	OtlpTracesProtocol OTLPProtocol `json:"otlpTracesProtocol,omitempty" yaml:"otlpTracesProtocol,omitempty" env:"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL" enum:"grpc,http/protobuf" jsonschema:"enum=grpc,enum=http/protobuf" help:"OTLP receiver protocol for traces."`
	// OTLP receiver protocol for metrics.
	OtlpMetricsProtocol OTLPProtocol `json:"otlpMetricsProtocol,omitempty" yaml:"otlpMetricsProtocol,omitempty" env:"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL" enum:"grpc,http/protobuf" jsonschema:"enum=grpc,enum=http/protobuf" help:"OTLP receiver protocol for metrics."`
	// OTLP receiver protocol for logs.
	OtlpLogsProtocol OTLPProtocol `json:"otlpLogsProtocol,omitempty" yaml:"otlpLogsProtocol,omitempty" env:"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL" enum:"grpc,http/protobuf" jsonschema:"enum=grpc,enum=http/protobuf" help:"OTLP receiver protocol for logs."`
	// Enable compression for OTLP exporters. Accept: none, gzip
	OtlpCompression OTLPCompressionType `json:"otlpCompression,omitempty" yaml:"otlpCompression,omitempty" env:"OTEL_EXPORTER_OTLP_COMPRESSION" default:"gzip" enum:"none,gzip" jsonschema:"enum=none,enum=gzip" help:"Enable compression for OTLP exporters. Accept: none, gzip"`
	// Enable compression for OTLP traces exporter. Accept: none, gzip
	OtlpTracesCompression OTLPCompressionType `json:"otlpTracesCompression,omitempty" yaml:"otlpTracesCompression,omitempty" env:"OTEL_EXPORTER_OTLP_TRACES_COMPRESSION" enum:"none,gzip" jsonschema:"enum=none,enum=gzip" help:"Enable compression for OTLP traces exporter. Accept: none, gzip"`
	// Enable compression for OTLP metrics exporter. Accept: none, gzip
	OtlpMetricsCompression OTLPCompressionType `json:"otlpMetricsCompression,omitempty" yaml:"otlpMetricsCompression,omitempty" env:"OTEL_EXPORTER_OTLP_METRICS_COMPRESSION" enum:"none,gzip" jsonschema:"enum=none,enum=gzip" help:"Enable compression for OTLP metrics exporter. Accept: none, gzip"`
	// Enable compression for OTLP logs exporter. Accept: none, gzip
	OtlpLogsCompression OTLPCompressionType `json:"otlpLogsCompression,omitempty" yaml:"otlpLogsCompression,omitempty" env:"OTEL_EXPORTER_OTLP_LOGS_COMPRESSION" enum:"none,gzip" jsonschema:"enum=none,enum=gzip" help:"Enable compression for OTLP logs exporter. Accept: none, gzip"`
	// Metrics export type. Accept: none, otlp, prometheus
	MetricsExporter OTELMetricsExporterType `json:"metricsExporter,omitempty" yaml:"metricsExporter,omitempty" env:"OTEL_METRICS_EXPORTER" default:"none" enum:"none,otlp,prometheus" jsonschema:"enum=none,enum=otlp,enum=prometheus" help:"Metrics export type. Accept: none, otlp, prometheus"`
	// Logs export type. Accept: none, otlp
	LogsExporter OTELLogsExporterType `json:"logsExporter,omitempty" yaml:"logsExporter,omitempty" env:"OTEL_LOGS_EXPORTER" default:"none" enum:"none,otlp" jsonschema:"enum=none,enum=otlp" help:"Logs export type. Accept: none, otlp"`
	// Prometheus port for the Prometheus HTTP server. Use /metrics endpoint of the connector server if empty.
	PrometheusPort *uint `json:"prometheusPort,omitempty" yaml:"prometheusPort,omitempty" env:"OTEL_EXPORTER_PROMETHEUS_PORT" jsonschema:"minimum=1000,maximum=65535" help:"Prometheus port for the Prometheus HTTP server. Use /metrics endpoint of the connector server if empty"`
	// Disable internal Go and process metrics (prometheus exporter only).
	DisableGoMetrics *bool `json:"disableGoMetrics,omitempty" yaml:"disableGoMetrics,omitempty" help:"Disable internal Go and process metrics"`
}

// GetOTLPProtocol returns the OTLP protocol for OpenTelemetry exporters. Default is grpc.
func (oc OTLPConfig) GetOTLPProtocol() OTLPProtocol {
	if oc.OtlpProtocol == "" {
		return OTLPProtocolGRPC
	}

	return oc.OtlpProtocol
}

// GetOTLPTracesProtocol returns the OTLP protocol for OTEL traces exporter.
func (oc OTLPConfig) GetOTLPTracesProtocol() OTLPProtocol {
	if oc.OtlpTracesProtocol != "" {
		return oc.OtlpTracesProtocol
	}

	return oc.GetOTLPProtocol()
}

// GetOTLPMetricsProtocol returns the OTLP protocol for OTEL metrics exporter.
func (oc OTLPConfig) GetOTLPMetricsProtocol() OTLPProtocol {
	if oc.OtlpMetricsProtocol != "" {
		return oc.OtlpMetricsProtocol
	}

	return oc.GetOTLPProtocol()
}

// GetOTLPLogsProtocol returns the OTLP protocol for OTEL logs exporter.
func (oc OTLPConfig) GetOTLPLogsProtocol() OTLPProtocol {
	if oc.OtlpLogsProtocol != "" {
		return oc.OtlpLogsProtocol
	}

	return oc.GetOTLPProtocol()
}

// GetOTLPCompression returns the OTLP compression type. Default is gzip.
func (oc OTLPConfig) GetOTLPCompression() OTLPCompressionType {
	if oc.OtlpCompression == "" {
		return OTLPCompressionGzip
	}

	return oc.OtlpCompression
}

// GetOTLPTracesCompression returns the OTLP traces compression type. Default is the otlpCompression value.
func (oc OTLPConfig) GetOTLPTracesCompression() OTLPCompressionType {
	if oc.OtlpTracesCompression != "" {
		return oc.OtlpTracesCompression
	}

	return oc.GetOTLPCompression()
}

// GetOTLPMetricsCompression returns the OTLP metrics compression type. Default is the otlpCompression value.
func (oc OTLPConfig) GetOTLPMetricsCompression() OTLPCompressionType {
	if oc.OtlpMetricsCompression != "" {
		return oc.OtlpMetricsCompression
	}

	return oc.GetOTLPCompression()
}

// GetOTLPLogsCompression returns the OTLP logs compression type. Default is the otlpCompression value.
func (oc OTLPConfig) GetOTLPLogsCompression() OTLPCompressionType {
	if oc.OtlpLogsCompression != "" {
		return oc.OtlpLogsCompression
	}

	return oc.GetOTLPCompression()
}

// GetMetricsExporter returns the type of metrics exporter. Default is none.
func (oc OTLPConfig) GetMetricsExporter() OTELMetricsExporterType {
	if oc.MetricsExporter == "" {
		return OTELMetricsExporterNone
	}

	return oc.MetricsExporter
}

// GetLogsExporter returns the type of logs exporter. Default is none.
func (oc OTLPConfig) GetLogsExporter() OTELLogsExporterType {
	if oc.LogsExporter == "" {
		return OTELLogsExporterNone
	}

	return OTELLogsExporterOTLP
}
