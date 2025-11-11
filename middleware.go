package gotel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	Options                *tracingMiddlewareOptions
	Exporters              *OTelExporterResults
	Next                   http.Handler
	ActiveRequestsMetric   metric.Int64UpDownCounter
	RequestBodySizeMetric  metric.Int64Histogram
	ResponseBodySizeMetric metric.Int64Histogram
	RequestDurationMetric  metric.Float64Histogram
}

// NewTracingMiddleware creates a middleware with tracing and logger.
func NewTracingMiddleware(
	exporters *OTelExporterResults,
	options ...TracingMiddlewareOption,
) func(http.Handler) http.Handler {
	tmOptions := &tracingMiddlewareOptions{
		DebugPaths: []string{"/metrics", "/health", "/healthz"},
	}

	for _, option := range options {
		option(tmOptions)
	}

	// metrics follow the opentelemetry semantic convention
	// https://opentelemetry.io/docs/specs/semconv/http/http-metrics/
	requestDurationMetric, err := exporters.Meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(fmt.Errorf("failed to create http.server.request.duration metric: %w", err))
	}

	activeRequestsMetric, err := exporters.Meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests"),
	)
	if err != nil {
		panic(fmt.Errorf("failed to create http.server.active_requests metric: %w", err))
	}

	requestBodySizeMetric, err := exporters.Meter.Int64Histogram(
		"http.server.request.body.size",
		metric.WithDescription("Size of HTTP server request bodies"),
		metric.WithUnit("By"),
	)
	if err != nil {
		panic(fmt.Errorf("failed to create http.server.request.body.size metric: %w", err))
	}

	responseBodySizeMetric, err := exporters.Meter.Int64Histogram(
		"http.server.response.body.size",
		metric.WithDescription("Size of HTTP server response bodies"),
		metric.WithUnit("By"),
	)
	if err != nil {
		panic(fmt.Errorf("failed to create http.server.response.body.size metric: %w", err))
	}

	return func(next http.Handler) http.Handler {
		return &tracingMiddleware{
			Options:                tmOptions,
			Exporters:              exporters,
			Next:                   next,
			RequestDurationMetric:  requestDurationMetric,
			RequestBodySizeMetric:  requestBodySizeMetric,
			ResponseBodySizeMetric: responseBodySizeMetric,
			ActiveRequestsMetric:   activeRequestsMetric,
		}
	}
}

// ServeHTTP handles and responds to an HTTP request.
func (tm *tracingMiddleware) ServeHTTP( //nolint:gocognit,cyclop,funlen,maintidx
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	hostName, rawPort, _ := strings.Cut(r.Host, ":")
	port := 80

	if rawPort != "" {
		p, err := strconv.Atoi(rawPort)
		if err == nil {
			port = p
		}
	} else if strings.HasPrefix(r.URL.Scheme, "https") {
		port = 443
	}

	metricAttrs := []attribute.KeyValue{
		attribute.String("http.request.method", r.Method),
		attribute.String("url.scheme", r.URL.Scheme),
		attribute.String("server.address", hostName),
		attribute.Int("server.port", port),
	}
	requestPathAttr := attribute.String("http.request.path", r.URL.Path)

	if !tm.Options.HighCardinalityMetricDisabled {
		metricAttrs = append(metricAttrs, requestPathAttr)
	}

	activeRequestsAttrSet := metric.WithAttributeSet(attribute.NewSet(metricAttrs...))

	tm.ActiveRequestsMetric.Add(ctx, 1, activeRequestsAttrSet)

	metricAttrs = append(
		metricAttrs,
		attribute.String(
			"network.protocol.version",
			fmt.Sprintf("%d.%d", r.ProtoMajor, r.ProtoMinor),
		),
	)

	if !slices.Contains(tm.Options.DebugPaths, strings.ToLower(r.URL.Path)) {
		ctx, span = tm.Exporters.Tracer.Start(
			otel.GetTextMapPropagator().
				Extract(r.Context(), propagation.HeaderCarrier(r.Header)),
			tm.Options.getRequestSpanName(r),
			trace.WithSpanKind(trace.SpanKindServer),
		)

		defer span.End()
	}

	requestID := getRequestID(r)
	logger := tm.Exporters.Logger.With(slog.String("request_id", requestID))
	httpLogger := logger.With(slog.String("type", "http-log"))
	isDebug := logger.Enabled(ctx, slog.LevelDebug)

	// Add HTTP semantic attributes to the server span
	// See: https://opentelemetry.io/docs/specs/semconv/http/http-spans/#http-server-semantic-conventions
	span.SetAttributes(metricAttrs...)
	span.SetAttributes(requestPathAttr)

	requestBodySize := r.ContentLength
	requestLogHeaders := NewTelemetryHeaders(r.Header, tm.Options.AllowedRequestHeaders...)
	requestLogData := map[string]any{
		"url":            r.URL.String(),
		"method":         r.Method,
		"remote_address": r.RemoteAddr,
		"headers":        requestLogHeaders,
		"size":           requestBodySize,
	}

	SetSpanHeaderAttributes(span, "http.request.header", requestLogHeaders)

	var (
		ww             WrapResponseWriter
		responseReader *bytes.Buffer
	)

	if tm.Options.ResponseWriterWrapperFunc != nil {
		ww = tm.Options.ResponseWriterWrapperFunc(w, r.ProtoMajor)
	} else {
		ww = &basicWriter{
			ResponseWriter: w,
		}
	}

	if isDebug {
		responseReader = &bytes.Buffer{}
		ww.Tee(responseReader)
	}

	responseLogData := map[string]any{}

	traceResponse := func(statusCode int, description string, err any) {
		tm.ActiveRequestsMetric.Add(
			ctx,
			-1,
			activeRequestsAttrSet,
		)

		statusCodeAttr := attribute.Int(
			"http.response.status_code",
			statusCode,
		)
		latency := time.Since(start).Seconds()

		responseLogData["status"] = statusCode

		logAttrs := []any{
			slog.Float64("latency", latency),
			slog.Any("request", requestLogData),
			slog.Any("response", responseLogData),
		}

		if err != nil {
			stack := string(debug.Stack())
			logAttrs = append(logAttrs, slog.Any("error", err), slog.String("stacktrace", stack))
			span.SetAttributes(statusCodeAttr, attribute.String("stacktrace", stack))
		}

		metricAttrs = append(metricAttrs, statusCodeAttr)

		metricAttrSet := metric.WithAttributeSet(attribute.NewSet(metricAttrs...))
		if requestBodySize > 0 {
			tm.RequestBodySizeMetric.Record(ctx, requestBodySize, metricAttrSet)
		}

		if ww.BytesWritten() > 0 {
			tm.ResponseBodySizeMetric.Record(ctx, int64(ww.BytesWritten()), metricAttrSet)
		}

		tm.RequestDurationMetric.Record(ctx, latency, metricAttrSet)

		if statusCode >= http.StatusBadRequest {
			span.SetStatus(codes.Error, description)
			httpLogger.Error(description, logAttrs...)

			return
		}

		printSuccess := httpLogger.Info

		if slices.Contains(tm.Options.DebugPaths, r.URL.Path) {
			printSuccess = httpLogger.Debug
		}

		span.SetStatus(codes.Ok, "")
		printSuccess("success", logAttrs...)
	}

	if isDebug && r.Body != nil && isContentTypeDebuggable(r.Header.Get(contentTypeHeader)) {
		bodyStr, err := debugRequestBody(ww, r, logger)
		if err != nil {
			statusCode := http.StatusUnprocessableEntity
			traceResponse(statusCode, "failed to read request body", err)
			span.RecordError(err)

			return
		}

		span.SetAttributes(attribute.String("http.request.body", bodyStr))
		requestLogData["body"] = bodyStr
		requestBodySize = int64(len(bodyStr))
	}

	if requestBodySize > 0 {
		requestLogData["size"] = requestBodySize
		span.SetAttributes(attribute.Int64("http.request.body.size", requestBodySize))
	}

	defer func() {
		if err := recover(); err != nil {
			statusCode := http.StatusInternalServerError
			traceResponse(statusCode, "internal server error", err)

			writeResponseJSON(w, statusCode, map[string]any{
				"status":   statusCode,
				"title":    http.StatusText(statusCode),
				"instance": r.URL.Path,
				"extensions": map[string]any{
					"cause": err,
				},
			}, logger)

			errBytes, jsonErr := json.Marshal(err)
			if jsonErr != nil {
				span.SetAttributes(attribute.String("error", fmt.Sprintf("%v", err)))
			} else {
				span.SetAttributes(attribute.String("error", string(errBytes)))
			}
		}
	}()

	rr := r.WithContext(NewContextWithLogger(ctx, logger))

	tm.Next.ServeHTTP(ww, rr)

	statusCode := ww.Status()
	responseLogHeaders := NewTelemetryHeaders(ww.Header(), tm.Options.AllowedResponseHeaders...)
	responseLogData["size"] = ww.BytesWritten()
	responseLogData["headers"] = responseLogHeaders

	span.SetAttributes(attribute.Int("http.response.body.size", ww.BytesWritten()))
	SetSpanHeaderAttributes(span, "http.response.header", responseLogHeaders)

	// skip printing very large responses.
	if responseReader != nil && ww.BytesWritten() < 100*1024 {
		responseBody := responseReader.String()
		responseLogData["body"] = responseBody
		span.SetAttributes(attribute.String("http.response.body", responseBody))
	}

	if statusCode >= http.StatusBadRequest {
		traceResponse(statusCode, http.StatusText(statusCode), nil)

		return
	}

	traceResponse(statusCode, "success", nil)
}

type tracingMiddlewareOptions struct {
	HighCardinalitySpanDisabled   bool
	HighCardinalityMetricDisabled bool
	DebugPaths                    []string
	AllowedRequestHeaders         []string
	AllowedResponseHeaders        []string
	ResponseWriterWrapperFunc     NewWrapResponseWriterFunc
}

// TracingMiddlewareOption abstracts a function to apply options to the tracing middleware.
type TracingMiddlewareOption func(*tracingMiddlewareOptions)

// DisableHighCardinalitySpans set the option to disable high cardinality spans.
// The request path is removed from the span name.
func DisableHighCardinalitySpans(disabled bool) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.HighCardinalitySpanDisabled = disabled
	}
}

// DisableHighCardinalityMetrics set the option to disable high cardinality http_path labels.
func DisableHighCardinalityMetrics(disabled bool) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.HighCardinalityMetricDisabled = disabled
	}
}

// DebugPaths return an option to add request paths to be printed logs in the debug level.
// By default, metrics and health check endpoints are added to avoid noisy logs.
func DebugPaths(paths []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.DebugPaths = append(tmo.DebugPaths, paths...)
	}
}

// AllowRequestHeaders return an option to set allowed request headers.
// If empty, all headers are allowed.
func AllowRequestHeaders(names []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.AllowedRequestHeaders = toLowerStrings(names)
	}
}

// AllowResponseHeaders return an option to set allowed response headers.
// If empty, all headers are allowed.
func AllowResponseHeaders(names []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.AllowedResponseHeaders = toLowerStrings(names)
	}
}

// ResponseWriterWrapperFunc return an option to set the response writer wrapper function.
func ResponseWriterWrapperFunc(wrapper NewWrapResponseWriterFunc) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.ResponseWriterWrapperFunc = wrapper
	}
}

func (opts *tracingMiddlewareOptions) getRequestSpanName(req *http.Request) string {
	if opts.HighCardinalitySpanDisabled || req.URL.Path == "" {
		return req.Method
	}

	if req.URL.Path[0] == '/' {
		return req.Method + " " + req.URL.Path
	}

	return req.Method + " /" + req.URL.Path
}
