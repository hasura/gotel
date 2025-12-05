package gotel

import (
	"encoding/json"
	"testing"
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

		var config OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if config.ServiceName != "test-service" {
			t.Errorf("expected ServiceName 'test-service', got '%s'", config.ServiceName)
		}
		if config.OtlpEndpoint != "http://localhost:4317" {
			t.Errorf("expected OtlpEndpoint 'http://localhost:4317', got '%s'", config.OtlpEndpoint)
		}
		if config.OtlpTracesEndpoint != "http://localhost:4318" {
			t.Errorf("expected OtlpTracesEndpoint 'http://localhost:4318', got '%s'", config.OtlpTracesEndpoint)
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
		if config.OtlpProtocol != OTLPProtocolGRPC {
			t.Errorf("expected OtlpProtocol 'grpc', got '%s'", config.OtlpProtocol)
		}
		if config.OtlpTracesProtocol != OTLPProtocolHTTPProtobuf {
			t.Errorf("expected OtlpTracesProtocol 'http/protobuf', got '%s'", config.OtlpTracesProtocol)
		}
		if config.OtlpMetricsProtocol != OTLPProtocolGRPC {
			t.Errorf("expected OtlpMetricsProtocol 'grpc', got '%s'", config.OtlpMetricsProtocol)
		}
		if config.OtlpLogsProtocol != OTLPProtocolHTTPProtobuf {
			t.Errorf("expected OtlpLogsProtocol 'http/protobuf', got '%s'", config.OtlpLogsProtocol)
		}
		if config.OtlpCompression != OTLPCompressionGzip {
			t.Errorf("expected OtlpCompression 'gzip', got '%s'", config.OtlpCompression)
		}
		if config.OtlpTracesCompression != OTLPCompressionNone {
			t.Errorf("expected OtlpTracesCompression 'none', got '%s'", config.OtlpTracesCompression)
		}
		if config.OtlpMetricsCompression != OTLPCompressionGzip {
			t.Errorf("expected OtlpMetricsCompression 'gzip', got '%s'", config.OtlpMetricsCompression)
		}
		if config.OtlpLogsCompression != OTLPCompressionNone {
			t.Errorf("expected OtlpLogsCompression 'none', got '%s'", config.OtlpLogsCompression)
		}
		if config.MetricsExporter != OTELMetricsExporterOTLP {
			t.Errorf("expected MetricsExporter 'otlp', got '%s'", config.MetricsExporter)
		}
		if config.LogsExporter != OTELLogsExporterOTLP {
			t.Errorf("expected LogsExporter 'otlp', got '%s'", config.LogsExporter)
		}
		if config.PrometheusPort == nil || *config.PrometheusPort != 9090 {
			t.Error("expected PrometheusPort to be 9090")
		}
		if config.DisableGoMetrics == nil || *config.DisableGoMetrics != true {
			t.Error("expected DisableGoMetrics to be true")
		}
	})

	t.Run("unmarshals minimal config", func(t *testing.T) {
		jsonData := `{
			"serviceName": "minimal-service"
		}`

		var config OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if config.ServiceName != "minimal-service" {
			t.Errorf("expected ServiceName 'minimal-service', got '%s'", config.ServiceName)
		}
		if config.OtlpEndpoint != "" {
			t.Errorf("expected empty OtlpEndpoint, got '%s'", config.OtlpEndpoint)
		}
		if config.OtlpInsecure != nil {
			t.Error("expected OtlpInsecure to be nil")
		}
		if config.OtlpProtocol != "" {
			t.Errorf("expected empty OtlpProtocol, got '%s'", config.OtlpProtocol)
		}
		if config.OtlpCompression != "" {
			t.Errorf("expected empty OtlpCompression, got '%s'", config.OtlpCompression)
		}
		if config.MetricsExporter != "" {
			t.Errorf("expected empty MetricsExporter, got '%s'", config.MetricsExporter)
		}
		if config.LogsExporter != "" {
			t.Errorf("expected empty LogsExporter, got '%s'", config.LogsExporter)
		}
	})

	t.Run("unmarshals empty config", func(t *testing.T) {
		jsonData := `{}`

		var config OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if config.ServiceName != "" {
			t.Errorf("expected empty ServiceName, got '%s'", config.ServiceName)
		}
	})

	t.Run("unmarshals prometheus exporter config", func(t *testing.T) {
		jsonData := `{
			"serviceName": "prometheus-service",
			"metricsExporter": "prometheus",
			"prometheusPort": 8080,
			"disableGoMetrics": false
		}`

		var config OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if config.ServiceName != "prometheus-service" {
			t.Errorf("expected ServiceName 'prometheus-service', got '%s'", config.ServiceName)
		}
		if config.MetricsExporter != OTELMetricsExporterPrometheus {
			t.Errorf("expected MetricsExporter 'prometheus', got '%s'", config.MetricsExporter)
		}
		if config.PrometheusPort == nil || *config.PrometheusPort != 8080 {
			t.Error("expected PrometheusPort to be 8080")
		}
		if config.DisableGoMetrics == nil || *config.DisableGoMetrics != false {
			t.Error("expected DisableGoMetrics to be false")
		}
	})

	t.Run("unmarshals none exporters config", func(t *testing.T) {
		jsonData := `{
			"serviceName": "none-service",
			"metricsExporter": "none",
			"logsExporter": "none"
		}`

		var config OTLPConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if config.MetricsExporter != OTELMetricsExporterNone {
			t.Errorf("expected MetricsExporter 'none', got '%s'", config.MetricsExporter)
		}
		if config.LogsExporter != OTELLogsExporterNone {
			t.Errorf("expected LogsExporter 'none', got '%s'", config.LogsExporter)
		}
	})
}

func TestOTLPConfig_GetOTLPProtocol(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPProtocol
	}{
		{
			name:     "returns default grpc when empty",
			config:   OTLPConfig{},
			expected: OTLPProtocolGRPC,
		},
		{
			name: "returns configured protocol",
			config: OTLPConfig{
				OtlpProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "returns grpc protocol",
			config: OTLPConfig{
				OtlpProtocol: OTLPProtocolGRPC,
			},
			expected: OTLPProtocolGRPC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPProtocol()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPTracesProtocol(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPProtocol
	}{
		{
			name:     "returns default grpc when empty",
			config:   OTLPConfig{},
			expected: OTLPProtocolGRPC,
		},
		{
			name: "returns traces protocol when set",
			config: OTLPConfig{
				OtlpTracesProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "falls back to general protocol",
			config: OTLPConfig{
				OtlpProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "prefers traces protocol over general",
			config: OTLPConfig{
				OtlpProtocol:       OTLPProtocolGRPC,
				OtlpTracesProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPTracesProtocol()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPMetricsProtocol(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPProtocol
	}{
		{
			name:     "returns default grpc when empty",
			config:   OTLPConfig{},
			expected: OTLPProtocolGRPC,
		},
		{
			name: "returns metrics protocol when set",
			config: OTLPConfig{
				OtlpMetricsProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "falls back to general protocol",
			config: OTLPConfig{
				OtlpProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "prefers metrics protocol over general",
			config: OTLPConfig{
				OtlpProtocol:        OTLPProtocolGRPC,
				OtlpMetricsProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPMetricsProtocol()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPLogsProtocol(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPProtocol
	}{
		{
			name:     "returns default grpc when empty",
			config:   OTLPConfig{},
			expected: OTLPProtocolGRPC,
		},
		{
			name: "returns logs protocol when set",
			config: OTLPConfig{
				OtlpLogsProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "falls back to general protocol",
			config: OTLPConfig{
				OtlpProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
		{
			name: "prefers logs protocol over general",
			config: OTLPConfig{
				OtlpProtocol:     OTLPProtocolGRPC,
				OtlpLogsProtocol: OTLPProtocolHTTPProtobuf,
			},
			expected: OTLPProtocolHTTPProtobuf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPLogsProtocol()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPCompression(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPCompressionType
	}{
		{
			name:     "returns default gzip when empty",
			config:   OTLPConfig{},
			expected: OTLPCompressionGzip,
		},
		{
			name: "returns configured compression",
			config: OTLPConfig{
				OtlpCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "returns gzip compression",
			config: OTLPConfig{
				OtlpCompression: OTLPCompressionGzip,
			},
			expected: OTLPCompressionGzip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPCompression()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPTracesCompression(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPCompressionType
	}{
		{
			name:     "returns default gzip when empty",
			config:   OTLPConfig{},
			expected: OTLPCompressionGzip,
		},
		{
			name: "returns traces compression when set",
			config: OTLPConfig{
				OtlpTracesCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "falls back to general compression",
			config: OTLPConfig{
				OtlpCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "prefers traces compression over general",
			config: OTLPConfig{
				OtlpCompression:       OTLPCompressionGzip,
				OtlpTracesCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPTracesCompression()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPMetricsCompression(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPCompressionType
	}{
		{
			name:     "returns default gzip when empty",
			config:   OTLPConfig{},
			expected: OTLPCompressionGzip,
		},
		{
			name: "returns metrics compression when set",
			config: OTLPConfig{
				OtlpMetricsCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "falls back to general compression",
			config: OTLPConfig{
				OtlpCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "prefers metrics compression over general",
			config: OTLPConfig{
				OtlpCompression:        OTLPCompressionGzip,
				OtlpMetricsCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPMetricsCompression()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetOTLPLogsCompression(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTLPCompressionType
	}{
		{
			name:     "returns default gzip when empty",
			config:   OTLPConfig{},
			expected: OTLPCompressionGzip,
		},
		{
			name: "returns logs compression when set",
			config: OTLPConfig{
				OtlpLogsCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "falls back to general compression",
			config: OTLPConfig{
				OtlpCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
		{
			name: "prefers logs compression over general",
			config: OTLPConfig{
				OtlpCompression:     OTLPCompressionGzip,
				OtlpLogsCompression: OTLPCompressionNone,
			},
			expected: OTLPCompressionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetOTLPLogsCompression()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetMetricsExporter(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTELMetricsExporterType
	}{
		{
			name:     "returns default none when empty",
			config:   OTLPConfig{},
			expected: OTELMetricsExporterNone,
		},
		{
			name: "returns otlp exporter",
			config: OTLPConfig{
				MetricsExporter: OTELMetricsExporterOTLP,
			},
			expected: OTELMetricsExporterOTLP,
		},
		{
			name: "returns prometheus exporter",
			config: OTLPConfig{
				MetricsExporter: OTELMetricsExporterPrometheus,
			},
			expected: OTELMetricsExporterPrometheus,
		},
		{
			name: "returns none exporter",
			config: OTLPConfig{
				MetricsExporter: OTELMetricsExporterNone,
			},
			expected: OTELMetricsExporterNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetMetricsExporter()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_GetLogsExporter(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPConfig
		expected OTELLogsExporterType
	}{
		{
			name:     "returns default none when empty",
			config:   OTLPConfig{},
			expected: OTELLogsExporterNone,
		},
		{
			name: "returns otlp exporter when set to otlp",
			config: OTLPConfig{
				LogsExporter: OTELLogsExporterOTLP,
			},
			expected: OTELLogsExporterOTLP,
		},
		{
			name: "returns none exporter when set to none",
			config: OTLPConfig{
				LogsExporter: OTELLogsExporterNone,
			},
			expected: OTELLogsExporterNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetLogsExporter()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOTLPConfig_MarshalJSON(t *testing.T) {
	t.Run("marshals complete config", func(t *testing.T) {
		trueVal := true
		falseVal := false
		port := uint(9090)

		config := OTLPConfig{
			ServiceName:            "test-service",
			OtlpEndpoint:           "http://localhost:4317",
			OtlpTracesEndpoint:     "http://localhost:4318",
			OtlpMetricsEndpoint:    "http://localhost:4319",
			OtlpLogsEndpoint:       "http://localhost:4320",
			OtlpInsecure:           &trueVal,
			OtlpTracesInsecure:     &falseVal,
			OtlpMetricsInsecure:    &trueVal,
			OtlpLogsInsecure:       &falseVal,
			OtlpProtocol:           OTLPProtocolGRPC,
			OtlpTracesProtocol:     OTLPProtocolHTTPProtobuf,
			OtlpMetricsProtocol:    OTLPProtocolGRPC,
			OtlpLogsProtocol:       OTLPProtocolHTTPProtobuf,
			OtlpCompression:        OTLPCompressionGzip,
			OtlpTracesCompression:  OTLPCompressionNone,
			OtlpMetricsCompression: OTLPCompressionGzip,
			OtlpLogsCompression:    OTLPCompressionNone,
			MetricsExporter:        OTELMetricsExporterOTLP,
			LogsExporter:           OTELLogsExporterOTLP,
			PrometheusPort:         &port,
			DisableGoMetrics:       &trueVal,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		// Unmarshal back to verify
		var decoded OTLPConfig
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if decoded.ServiceName != config.ServiceName {
			t.Errorf("expected ServiceName '%s', got '%s'", config.ServiceName, decoded.ServiceName)
		}
		if decoded.OtlpProtocol != config.OtlpProtocol {
			t.Errorf("expected OtlpProtocol '%s', got '%s'", config.OtlpProtocol, decoded.OtlpProtocol)
		}
		if decoded.MetricsExporter != config.MetricsExporter {
			t.Errorf("expected MetricsExporter '%s', got '%s'", config.MetricsExporter, decoded.MetricsExporter)
		}
	})

	t.Run("marshals minimal config", func(t *testing.T) {
		config := OTLPConfig{
			ServiceName: "minimal-service",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		// Unmarshal back to verify
		var decoded OTLPConfig
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if decoded.ServiceName != config.ServiceName {
			t.Errorf("expected ServiceName '%s', got '%s'", config.ServiceName, decoded.ServiceName)
		}
	})
}
