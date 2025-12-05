package main

import (
	"encoding/json"
	"testing"

	"github.com/caarlos0/env/v11"
	"github.com/hasura/gotel"
)

func TestOTLPConfig_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals complete config", func(t *testing.T) {
		jsonData := `{
			"serviceName": "test-service",
			"otlpEndpoint": "http://localhost:4317",
			"otlpTracesEndpoint": "http://localhost:4318",
			"otlpMetricsEndpoint": "http://localhost:4319",
			"otlpLogsEndpoint": "http://localhost:4320",
			"otlpInsecure": true,
			"otlpTracesInsecure": false,
			"otlpMetricsInsecure": true,
			"otlpLogsInsecure": false,
			"otlpProtocol": "grpc",
			"otlpTracesProtocol": "http/protobuf",
			"otlpMetricsProtocol": "grpc",
			"otlpLogsProtocol": "http/protobuf",
			"otlpCompression": "gzip",
			"otlpTracesCompression": "none",
			"otlpMetricsCompression": "gzip",
			"otlpLogsCompression": "none",
			"metricsExporter": "otlp",
			"logsExporter": "otlp",
			"prometheusPort": 9090,
			"disableGoMetrics": true
		}`

		var config gotel.OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		assertConfig := func() {
			if config.ServiceName != "test-service" {
				t.Errorf("expected ServiceName 'test-service', got '%s'", config.ServiceName)
			}
			if config.OtlpEndpoint != "http://localhost:4317" {
				t.Errorf("expected OtlpEndpoint 'http://localhost:4317', got '%s'", config.OtlpEndpoint)
			}
			if config.OtlpMetricsEndpoint != "http://localhost:4319" {
				t.Errorf("expected OtlpMetricsEndpoint 'http://localhost:4319', got '%s'", config.OtlpMetricsEndpoint)
			}
			if config.OtlpLogsEndpoint != "http://localhost:4320" {
				t.Errorf("expected OtlpLogsEndpoint 'http://localhost:4320', got '%s'", config.OtlpLogsEndpoint)
			}
			if config.OtlpInsecure == nil || *config.OtlpInsecure != true {
				t.Error("expected OtlpInsecure to be true")
			}
			if config.OtlpTracesInsecure == nil || *config.OtlpTracesInsecure != false {
				t.Error("expected OtlpTracesInsecure to be false")
			}
			if config.OtlpMetricsInsecure == nil || *config.OtlpMetricsInsecure != true {
				t.Error("expected OtlpMetricsInsecure to be true")
			}
			if config.OtlpLogsInsecure == nil || *config.OtlpLogsInsecure != false {
				t.Error("expected OtlpLogsInsecure to be false")
			}
			if config.OtlpProtocol != gotel.OTLPProtocolGRPC {
				t.Errorf("expected OtlpProtocol 'grpc', got '%s'", config.OtlpProtocol)
			}
			if config.OtlpTracesProtocol != gotel.OTLPProtocolHTTPProtobuf {
				t.Errorf("expected OtlpTracesProtocol 'http/protobuf', got '%s'", config.OtlpTracesProtocol)
			}
			if config.OtlpMetricsProtocol != gotel.OTLPProtocolGRPC {
				t.Errorf("expected OtlpMetricsProtocol 'grpc', got '%s'", config.OtlpMetricsProtocol)
			}
			if config.OtlpLogsProtocol != gotel.OTLPProtocolHTTPProtobuf {
				t.Errorf("expected OtlpLogsProtocol 'http/protobuf', got '%s'", config.OtlpLogsProtocol)
			}
			if config.OtlpCompression != gotel.OTLPCompressionGzip {
				t.Errorf("expected OtlpCompression 'gzip', got '%s'", config.OtlpCompression)
			}
			if config.OtlpTracesCompression != gotel.OTLPCompressionNone {
				t.Errorf("expected OtlpTracesCompression 'none', got '%s'", config.OtlpTracesCompression)
			}
			if config.OtlpMetricsCompression != gotel.OTLPCompressionGzip {
				t.Errorf("expected OtlpMetricsCompression 'gzip', got '%s'", config.OtlpMetricsCompression)
			}
			if config.OtlpLogsCompression != gotel.OTLPCompressionNone {
				t.Errorf("expected OtlpLogsCompression 'none', got '%s'", config.OtlpLogsCompression)
			}
			if config.MetricsExporter != gotel.OTELMetricsExporterOTLP {
				t.Errorf("expected MetricsExporter 'otlp', got '%s'", config.MetricsExporter)
			}
			if config.LogsExporter != gotel.OTELLogsExporterOTLP {
				t.Errorf("expected LogsExporter 'otlp', got '%s'", config.LogsExporter)
			}
			if config.PrometheusPort == nil || *config.PrometheusPort != 9090 {
				t.Error("expected PrometheusPort to be 9090")
			}
			if config.DisableGoMetrics == nil || *config.DisableGoMetrics != true {
				t.Error("expected DisableGoMetrics to be true")
			}
		}

		assertConfig()

		if config.OtlpTracesEndpoint != "http://localhost:4318" {
			t.Errorf("expected OtlpTracesEndpoint 'http://localhost:4318', got '%s'", config.OtlpTracesEndpoint)
		}

		t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://localhost:8080")

		// the config should not be changed if the env config is empty.
		err = env.Parse(&config)
		if err != nil {
			t.Fatalf("failed to load env: %v", err)
		}

		assertConfig()

		if config.OtlpTracesEndpoint != "http://localhost:8080" {
			t.Errorf("expected OtlpTracesEndpoint 'http://localhost:8080', got '%s'", config.OtlpTracesEndpoint)
		}
	})
}
