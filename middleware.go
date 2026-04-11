package gotel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/hasura/gotel/otelutils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	Options                *tracingMiddlewareOptions
	Exporters              *OTelExporters
	Next                   http.Handler
	ActiveRequestsMetric   metric.Int64UpDownCounter
	RequestBodySizeMetric  metric.Int64Histogram
	ResponseBodySizeMetric metric.Int64Histogram
	RequestDurationMetric  metric.Float64Histogram
}

// NewTracingMiddleware creates a middleware with tracing and logger.
func NewTracingMiddleware(
	exporters *OTelExporters,
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
		metric.WithExplicitBucketBoundaries(
			0.005,
			0.01,
			0.025,
			0.05,
			0.075,
			0.1,
			0.25,
			0.5,
			0.75,
			1,
			2.5,
			5,
			7.5,
			10,
		),
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
	urlPath := strings.ToLower(r.URL.Path)

	urlScheme := r.URL.Scheme
	if urlScheme == "" {
		urlScheme = "http"
	}

	_, port, _ := otelutils.SplitHostPort(r.Host, urlScheme)

	metricAttrs := []attribute.KeyValue{
		{
			Key:   semconv.HTTPRequestMethodKey,
			Value: attribute.StringValue(r.Method),
		},
		semconv.URLScheme(urlScheme),
		semconv.ServerAddress(r.Host),
		semconv.ServerPort(port),
	}

	if !slices.Contains(tm.Options.DebugPaths, urlPath) {
		ctx, span = tm.Exporters.Tracer.Start(
			otel.GetTextMapPropagator().
				Extract(r.Context(), propagation.HeaderCarrier(r.Header)),
			tm.Options.getRequestSpanName(r),
			trace.WithSpanKind(trace.SpanKindServer),
		)

		defer span.End()
	}

	requestID := getRequestID(r)
	logger := tm.Exporters.Logger.With(
		slog.String("request_id", requestID),
		slog.String("type", "http-log"),
	)
	isDebug := logger.Enabled(ctx, slog.LevelDebug)

	if tm.Options.CustomAttributesFunc != nil {
		metricAttrs = append(metricAttrs, tm.Options.CustomAttributesFunc(r)...)
	}
	// Add HTTP semantic attributes to the server span
	// See: https://opentelemetry.io/docs/specs/semconv/http/http-spans/#http-server-semantic-conventions
	span.SetAttributes(metricAttrs...)
	span.SetAttributes(semconv.ClientAddress(r.RemoteAddr))

	if tm.Options.HighCardinalityMetrics {
		metricAttrs = append(metricAttrs, semconv.URLPath(r.URL.Path))
	}

	activeRequestsAttrSet := metric.WithAttributeSet(attribute.NewSet(metricAttrs...))

	tm.ActiveRequestsMetric.Add(ctx, 1, activeRequestsAttrSet)

	protocolAttr := semconv.NetworkProtocolVersion(fmt.Sprintf("%d.%d", r.ProtoMajor, r.ProtoMinor))

	metricAttrs = append(
		metricAttrs,
		protocolAttr,
	)

	span.SetAttributes(
		protocolAttr,
		semconv.URLFull(r.URL.String()),
		semconv.UserAgentOriginal(r.UserAgent()),
	)

	peer, peerPort, _ := otelutils.SplitHostPort(r.RemoteAddr, "")

	if peer != "" {
		span.SetAttributes(semconv.NetworkPeerAddress(peer))

		if peerPort > 0 {
			span.SetAttributes(semconv.NetworkPeerPort(peerPort))
		}
	}

	requestBodySize := r.ContentLength
	requestLogHeaders := otelutils.ExtractTelemetryHeaders(
		r.Header,
		tm.Options.SensitivePatterns,
		tm.Options.AllowedRequestHeaders...)

	otelutils.SetSpanHeaderMatrixAttributes(span, "http.request.header", requestLogHeaders)

	var (
		ww             WrapResponseWriter
		responseReader *bytes.Buffer
		bodyStr        string
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

	traceResponse := func(statusCode int, err any) {
		tm.ActiveRequestsMetric.Add(
			ctx,
			-1,
			activeRequestsAttrSet,
		)

		statusCodeAttr := semconv.HTTPResponseStatusCode(statusCode)
		span.SetAttributes(statusCodeAttr)

		latency := time.Since(start).Seconds()

		metricAttrs = append(metricAttrs, statusCodeAttr)
		metricAttrSet := metric.WithAttributeSet(attribute.NewSet(metricAttrs...))

		if requestBodySize > 0 {
			tm.RequestBodySizeMetric.Record(ctx, requestBodySize, metricAttrSet)
		}

		bytesWritten := ww.BytesWritten()
		responseLogHeaders := otelutils.ExtractTelemetryHeaders(
			ww.Header(),
			tm.Options.SensitivePatterns,
			tm.Options.AllowedResponseHeaders...)

		span.SetAttributes(semconv.HTTPResponseBodySizeKey.Int(bytesWritten))
		otelutils.SetSpanHeaderMatrixAttributes(span, "http.response.header", responseLogHeaders)

		if bytesWritten > 0 {
			tm.ResponseBodySizeMetric.Record(ctx, int64(bytesWritten), metricAttrSet)
		}

		tm.RequestDurationMetric.Record(ctx, latency, metricAttrSet)

		successLevel := slog.LevelInfo

		if slices.Contains(tm.Options.DebugPaths, urlPath) {
			successLevel = slog.LevelDebug
		}

		if !logger.Enabled(ctx, successLevel) && err == nil &&
			statusCode < http.StatusBadRequest {
			span.SetStatus(codes.Ok, "")

			return
		}

		// build request attributes
		requestLogAttrs := make([]slog.Attr, 0, 6)
		requestLogAttrs = append(
			requestLogAttrs,
			slog.String("url", r.URL.String()),
			slog.String("method", r.Method),
			slog.String("remote_address", r.RemoteAddr),
			otelutils.NewHeaderMatrixLogGroupAttrs("headers", requestLogHeaders),
		)

		if requestBodySize > 0 {
			requestLogAttrs = append(requestLogAttrs, slog.Int64("size", requestBodySize))
		}

		if bodyStr != "" {
			requestLogAttrs = append(requestLogAttrs, slog.String("body", bodyStr))
		}

		// build response attributes
		responseLogAttrs := make([]slog.Attr, 0, 4)
		responseLogAttrs = append(responseLogAttrs, slog.Int("status", statusCode))
		responseLogAttrs = append(
			responseLogAttrs,
			slog.Int("size", bytesWritten),
			otelutils.NewHeaderMatrixLogGroupAttrs("headers", responseLogHeaders),
		)

		// skip printing very large responses.
		if responseReader != nil {
			responseBody := responseReader.String()
			responseLogAttrs = append(responseLogAttrs, slog.String("body", responseBody))
			span.SetAttributes(attribute.String("http.response.body", responseBody))
		}

		var logAttrs []slog.Attr

		if err != nil {
			logAttrs = make([]slog.Attr, 0, 5)
			stack := string(debug.Stack())
			logAttrs = append(
				logAttrs,
				slog.Any("error", fmt.Sprint(err)),
				slog.String("stacktrace", stack),
			)

			span.SetAttributes(semconv.ExceptionStacktrace(stack))
		} else {
			logAttrs = make([]slog.Attr, 0, 3)
		}

		logAttrs = append(
			logAttrs,
			slog.Float64("latency", latency),
			slog.GroupAttrs("request", requestLogAttrs...),
			slog.GroupAttrs("response", responseLogAttrs...),
		)

		if statusCode >= http.StatusBadRequest {
			message := http.StatusText(statusCode)

			logger.LogAttrs(ctx, slog.LevelError, message, logAttrs...)
			span.SetStatus(codes.Error, message)

			return
		}

		logger.LogAttrs(ctx, successLevel, http.StatusText(statusCode), logAttrs...)
		span.SetStatus(codes.Ok, "")
	}

	if isDebug && r.Body != nil && r.Body != http.NoBody &&
		otelutils.IsContentTypeDebuggable(r.Header.Get(contentTypeHeader)) {
		var err error

		bodyStr, err = debugRequestBody(ww, r, logger)
		if err != nil {
			statusCode := http.StatusUnprocessableEntity
			traceResponse(statusCode, err)
			span.RecordError(err)

			return
		}

		span.SetAttributes(attribute.String("http.request.body", bodyStr))
		requestBodySize = int64(len(bodyStr))
	}

	if requestBodySize > 0 {
		span.SetAttributes(semconv.HTTPRequestBodySize(int(requestBodySize)))
	}

	defer func() {
		if err := recover(); err != nil {
			statusCode := http.StatusInternalServerError
			traceResponse(statusCode, err)

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
				span.SetAttributes(attribute.String("exception.error", fmt.Sprintf("%v", err)))
			} else {
				span.SetAttributes(attribute.String("exception.error", string(errBytes)))
			}
		}
	}()

	rr := r.WithContext(otelutils.NewContextWithLogger(ctx, tm.Exporters.Logger.With(
		slog.String("request_id", requestID),
	)))

	tm.Next.ServeHTTP(ww, rr)

	statusCode := ww.Status()

	if statusCode >= http.StatusBadRequest {
		traceResponse(statusCode, nil)

		return
	}

	traceResponse(statusCode, nil)
}

type tracingMiddlewareOptions struct {
	DebugPaths                []string
	AllowedRequestHeaders     []string
	AllowedResponseHeaders    []string
	SensitivePatterns         []string
	ResponseWriterWrapperFunc NewWrapResponseWriterFunc
	CustomAttributesFunc      CustomAttributesFunc
	HighCardinalitySpans      bool
	HighCardinalityMetrics    bool
}

// CustomAttributesFunc abstracts a hook function to add custom attributes.
type CustomAttributesFunc func(r *http.Request) []attribute.KeyValue

// TracingMiddlewareOption abstracts a function to apply options to the tracing middleware.
type TracingMiddlewareOption func(*tracingMiddlewareOptions)

// WithHighCardinalitySpans set the option to enable high cardinality spans.
// The request path is removed from the span name.
func WithHighCardinalitySpans(enabled bool) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.HighCardinalitySpans = enabled
	}
}

// WithHighCardinalityMetrics set the option to enable high cardinality request path labels.
func WithHighCardinalityMetrics(enabled bool) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.HighCardinalityMetrics = enabled
	}
}

// WithCustomAttributesFunc set the option to add custom OpenTelemetry attributes.
func WithCustomAttributesFunc(fn CustomAttributesFunc) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.CustomAttributesFunc = fn
	}
}

// WithSensitivePatterns set the option to add sensitive patterns to be masked.
func WithSensitivePatterns(patterns []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.SensitivePatterns = otelutils.NormalizeStrings(patterns)
	}
}

// WithDebugPaths return an option to add request paths to be printed logs in the debug level.
// By default, metrics and health check endpoints are added to avoid noisy logs.
func WithDebugPaths(paths []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.DebugPaths = append(tmo.DebugPaths, paths...)
	}
}

// WithAllowedRequestHeaders return an option to set allowed request headers.
// If empty, all headers are allowed.
func WithAllowedRequestHeaders(names []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.AllowedRequestHeaders = otelutils.NormalizeStrings(names)
	}
}

// WithAllowedResponseHeaders return an option to set allowed response headers.
// If empty, all headers are allowed.
func WithAllowedResponseHeaders(names []string) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.AllowedResponseHeaders = otelutils.NormalizeStrings(names)
	}
}

// ResponseWriterWrapperFunc return an option to set the response writer wrapper function.
func ResponseWriterWrapperFunc(wrapper NewWrapResponseWriterFunc) TracingMiddlewareOption {
	return func(tmo *tracingMiddlewareOptions) {
		tmo.ResponseWriterWrapperFunc = wrapper
	}
}

func (opts *tracingMiddlewareOptions) getRequestSpanName(req *http.Request) string {
	if !opts.HighCardinalitySpans {
		return req.Method
	}

	if req.URL.Path == "" {
		return req.Method + " /"
	}

	if req.URL.Path[0] == '/' {
		return req.Method + " " + req.URL.Path
	}

	return req.Method + " /" + req.URL.Path
}
