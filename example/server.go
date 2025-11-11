package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hasura/gotel"
)

func main() {
	os.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "1000")

	logger, _, err := gotel.NewJSONLogger("DEBUG")
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	otlpConfig := &gotel.OTLPConfig{
		ServiceName:     "example",
		OtlpEndpoint:    "http://localhost:4317",
		OtlpProtocol:    string(gotel.OTLPProtocolGRPC),
		MetricsExporter: "otlp",
		LogsExporter:    "otlp",
	}

	ts, err := gotel.SetupOTelExporters(context.TODO(), otlpConfig, "v0.1.0", logger)
	if err != nil {
		log.Fatal(err)
	}

	defer ts.Shutdown(context.TODO())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte(fmt.Sprintf("%s %s", r.Method, r.URL.Path)))
	})

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	options := []gotel.TracingMiddlewareOption{
		gotel.AllowRequestHeaders([]string{}),
		gotel.AllowResponseHeaders([]string{"Content-Type"}),
		gotel.DebugPaths([]string{"/world"}),
		gotel.DisableHighCardinalitySpans(true),
		gotel.DisableHighCardinalityMetrics(false),
	}

	mux := http.NewServeMux()
	mux.Handle("/hello", gotel.NewTracingMiddleware(ts, options...)(handler))
	mux.Handle("/panic", gotel.NewTracingMiddleware(ts, options...)(panicHandler))

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	defer server.Close()

	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to serve http: %s", err)
	}
}
