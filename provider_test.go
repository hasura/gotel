package gotel

import (
	"testing"
)

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

func TestParseOTLPEndpoint(t *testing.T) {
	testCases := []struct {
		Name             string
		Endpoint         string
		Protocol         OTLPProtocol
		InsecurePtr      *bool
		ExpectedEndpoint string
		ExpectedProtocol OTLPProtocol
		ExpectedInsecure bool
		ExpectError      bool
	}{
		{
			Name:             "http endpoint with grpc protocol",
			Endpoint:         "http://localhost:4317",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      nil,
			ExpectedEndpoint: "localhost:4317",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: true,
			ExpectError:      false,
		},
		{
			Name:             "https endpoint with grpc protocol",
			Endpoint:         "https://localhost:4317",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      nil,
			ExpectedEndpoint: "localhost:4317",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: false,
			ExpectError:      false,
		},
		{
			Name:             "endpoint without scheme defaults to https",
			Endpoint:         "localhost:4317",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      nil,
			ExpectedEndpoint: "localhost:4317",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: false,
			ExpectError:      false,
		},
		{
			Name:             "http/protobuf protocol returns full URL",
			Endpoint:         "http://localhost:4318/v1/traces",
			Protocol:         OTLPProtocolHTTPProtobuf,
			InsecurePtr:      nil,
			ExpectedEndpoint: "http://localhost:4318/v1/traces",
			ExpectedProtocol: OTLPProtocolHTTPProtobuf,
			ExpectedInsecure: true,
			ExpectError:      false,
		},
		{
			Name:             "insecure flag overrides scheme",
			Endpoint:         "https://localhost:4317",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      boolPtr(true),
			ExpectedEndpoint: "localhost:4317",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: true,
			ExpectError:      false,
		},
		{
			Name:             "endpoint without port adds default https port",
			Endpoint:         "example.com",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      nil,
			ExpectedEndpoint: "example.com:443",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: false,
			ExpectError:      false,
		},
		{
			Name:             "endpoint without port adds default http port",
			Endpoint:         "http://example.com",
			Protocol:         OTLPProtocolGRPC,
			InsecurePtr:      nil,
			ExpectedEndpoint: "example.com:80",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: true,
			ExpectError:      false,
		},
		{
			Name:             "auto-detect protocol with default HTTP port",
			Endpoint:         "localhost:4318",
			Protocol:         "",
			InsecurePtr:      nil,
			ExpectedEndpoint: "localhost:4318",
			ExpectedProtocol: "",
			ExpectedInsecure: false,
			ExpectError:      false,
		},
		{
			Name:             "auto-detect protocol defaults to gRPC",
			Endpoint:         "localhost:4317",
			Protocol:         "",
			InsecurePtr:      nil,
			ExpectedEndpoint: "localhost:4317",
			ExpectedProtocol: OTLPProtocolGRPC,
			ExpectedInsecure: false,
			ExpectError:      false,
		},
		{
			Name:        "invalid protocol returns error",
			Endpoint:    "localhost:4317",
			Protocol:    "invalid",
			InsecurePtr: nil,
			ExpectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			endpoint, protocol, insecure, err := parseOTLPEndpoint(tc.Endpoint, tc.Protocol, tc.InsecurePtr)

			if tc.ExpectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if endpoint != tc.ExpectedEndpoint {
				t.Errorf("expected endpoint '%s', got '%s'", tc.ExpectedEndpoint, endpoint)
			}

			if protocol != tc.ExpectedProtocol {
				t.Errorf("expected protocol '%s', got '%s'", tc.ExpectedProtocol, protocol)
			}

			if insecure != tc.ExpectedInsecure {
				t.Errorf("expected insecure %v, got %v", tc.ExpectedInsecure, insecure)
			}
		})
	}
}

func TestParseOTLPCompression(t *testing.T) {
	testCases := []struct {
		Name                   string
		Input                  OTLPCompressionType
		ExpectedCompression    OTLPCompressionType
		ExpectedCompressionInt int
		ExpectError            bool
	}{
		{
			Name:                   "gzip compression",
			Input:                  OTLPCompressionGzip,
			ExpectedCompression:    OTLPCompressionGzip,
			ExpectedCompressionInt: 1, // GzipCompression value
			ExpectError:            false,
		},
		{
			Name:                   "no compression",
			Input:                  OTLPCompressionNone,
			ExpectedCompression:    OTLPCompressionNone,
			ExpectedCompressionInt: 0, // NoCompression value
			ExpectError:            false,
		},
		{
			Name:                   "empty defaults to gzip",
			Input:                  "",
			ExpectedCompression:    OTLPCompressionGzip,
			ExpectedCompressionInt: 1,
			ExpectError:            false,
		},
		{
			Name:        "invalid compression type",
			Input:       "invalid",
			ExpectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			compression, compressionInt, err := parseOTLPCompression(tc.Input)

			if tc.ExpectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if compression != tc.ExpectedCompression {
				t.Errorf("expected compression '%s', got '%s'", tc.ExpectedCompression, compression)
			}

			if compressionInt != tc.ExpectedCompressionInt {
				t.Errorf("expected compression int %d, got %d", tc.ExpectedCompressionInt, compressionInt)
			}
		})
	}
}

func TestNewResource(t *testing.T) {
	t.Run("creates resource with service name and version", func(t *testing.T) {
		serviceName := "test-service"
		serviceVersion := "v1.0.0"

		resource := newResource(serviceName, serviceVersion)

		if resource == nil {
			t.Fatal("expected non-nil resource")
		}

		// Check that the resource has attributes
		attrs := resource.Attributes()
		if len(attrs) == 0 {
			t.Error("expected resource to have attributes")
		}

		// Check for service name
		hasServiceName := false
		hasServiceVersion := false
		for _, attr := range attrs {
			if string(attr.Key) == "service.name" && attr.Value.AsString() == serviceName {
				hasServiceName = true
			}
			if string(attr.Key) == "service.version" && attr.Value.AsString() == serviceVersion {
				hasServiceVersion = true
			}
		}

		if !hasServiceName {
			t.Error("expected resource to have service.name attribute")
		}

		if !hasServiceVersion {
			t.Error("expected resource to have service.version attribute")
		}
	})
}

func TestNewPropagator(t *testing.T) {
	t.Run("creates composite propagator", func(t *testing.T) {
		propagator := newPropagator()

		if propagator == nil {
			t.Fatal("expected non-nil propagator")
		}

		// The propagator should have fields (TraceContext and B3)
		fields := propagator.Fields()
		if len(fields) == 0 {
			t.Error("expected propagator to have fields")
		}
	})
}
