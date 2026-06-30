package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"social.craftsky/appview/internal/observability"
)

// HTTPMetrics records completed HTTP requests with low-cardinality route labels.
func HTTPMetrics(observer *observability.Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			req := r
			var inFlightRoutePattern string
			var traceSpan *observability.Span
			spanFinished := false
			if observer != nil {
				req = r.WithContext(observability.WithCaptureMarker(r.Context()))
				spanCtx, span := observer.StartSpan(req.Context(), observability.SpanContext{
					Operation: "http.server",
					Component: "http",
					Attributes: observability.EventContext{
						"component":   "http",
						"http_method": req.Method,
						"run_id":      GetRunID(req.Context()),
					},
				})
				traceSpan = span
				req = req.WithContext(spanCtx)
				inFlightRoutePattern = observer.BeginHTTPRequest(req.Method, observability.RoutePattern(req))
				defer observer.EndHTTPRequest(req.Method, inFlightRoutePattern)
				defer func() {
					if traceSpan != nil && !spanFinished {
						traceSpan.Finish("error")
					}
				}()
			}
			rw := &responseLogger{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, req)
			if observer != nil {
				routePattern := observability.RoutePattern(req)
				duration := time.Since(started)
				observer.ObserveHTTPRequest(req.Method, routePattern, rw.status, duration, rw.bytes)
				result := "success"
				if rw.status >= http.StatusInternalServerError {
					result = "error"
				}
				if traceSpan != nil {
					traceSpan.SetTransactionName(req.Method + " " + routePattern)
					traceSpan.SetAttributes(observability.EventContext{
						"component":         "http",
						"route_pattern":     routePattern,
						"http_method":       req.Method,
						"http_status":       rw.status,
						"http_status_class": strconv.Itoa(rw.status/100) + "xx",
						"duration":          duration.String(),
						"result":            result,
						"run_id":            GetRunID(req.Context()),
					})
					traceSpan.Finish(result)
					spanFinished = true
				}
				if rw.status >= http.StatusInternalServerError && !observability.CaptureRecorded(req.Context()) {
					observer.CaptureError(req.Context(), observability.EventContext{
						"component":         "http",
						"route_pattern":     routePattern,
						"http_method":       req.Method,
						"http_status":       rw.status,
						"http_status_class": strconv.Itoa(rw.status/100) + "xx",
						"error_category":    "server",
						"duration":          duration.String(),
						"run_id":            GetRunID(req.Context()),
					}, errors.New("http server error response"))
				}
			}
		})
	}
}
