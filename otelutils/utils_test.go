package otelutils

import (
	"net/http"
	"reflect"
	"testing"
)

func TestNewTelemetryHeaders(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          http.Header
		AllowedHeaders []string
		Expected       map[string][]string
	}{
		{
			Name: "basic",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"Authorization": []string{
					"Bearer abcdefghijkxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				},
				"Api-Key":    []string{"abcxyz"},
				"Secret-Key": []string{"secret-key"},
				"X-Empty":    []string{},
			},
			Expected: map[string][]string{
				"content-type":  {"application/json"},
				"authorization": {MaskString},
				"api-key":       {MaskString},
				"secret-key":    {MaskString},
			},
		},
		{
			Name: "allowed_list",
			Input: http.Header{
				"Content-Type": []string{"application/json"},
				"Authorization": []string{
					"Bearer abcdefghijkxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				},
				"Api-Key":    []string{"abcxyz"},
				"Secret-Key": []string{"secret-key"},
			},
			AllowedHeaders: []string{"Content-Type", "Api-Key"},
			Expected: map[string][]string{
				"content-type": {"application/json"},
				"api-key":      {MaskString},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got := NewTelemetryHeaders(tc.Input, tc.AllowedHeaders...)

			if !reflect.DeepEqual(tc.Expected, got) {
				t.Errorf("expected: %v, got: %v", tc.Expected, got)
			}

			if reflect.DeepEqual(tc.Input, got) {
				t.Errorf("input: %v, got: %v", tc.Input, got)
			}
		})
	}
}
