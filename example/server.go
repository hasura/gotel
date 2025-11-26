package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/hasura/gotel"
	"github.com/hasura/gotel/otelutils"
)

func main() {
	os.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "1000")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	os.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	os.Setenv("OTEL_LOGS_EXPORTER", "otlp")

	logger, _, err := otelutils.NewJSONLogger("DEBUG")
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	otlpConfig, err := env.ParseAsWithOptions[gotel.OTLPConfig](env.Options{
		DefaultValueTagName: "default",
	})
	if err != nil {
		log.Fatal(err)
	}

	ts, err := gotel.SetupOTelExporters(context.TODO(), &otlpConfig, "v0.1.0", logger)
	if err != nil {
		log.Fatal(err)
	}

	defer ts.Shutdown(context.TODO())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := ts.Tracer.Start(r.Context(), "hello")
		defer span.End()

		w.Write([]byte(fmt.Sprintf("%s %s", r.Method, r.URL.Path)))
	})

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	healthzHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	options := []gotel.TracingMiddlewareOption{
		gotel.AllowRequestHeaders([]string{}),
		gotel.AllowResponseHeaders([]string{"Content-Type"}),
		gotel.WithDebugPaths([]string{"/world"}),
		gotel.WithHighCardinalitySpans(true),
		gotel.WithHighCardinalityMetrics(false),
	}

	mux := http.NewServeMux()
	mux.Handle("/hello", gotel.NewTracingMiddleware(ts, options...)(handler))
	mux.Handle("/panic", gotel.NewTracingMiddleware(ts, options...)(panicHandler))
	mux.Handle("/healthz", gotel.NewTracingMiddleware(ts, options...)(healthzHandler))

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
