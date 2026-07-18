package middleware

import (
	"context"
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
			var traceSpan *observability.Span
			var handlerSpan *observability.Span
			spanFinished := false
			handlerSpanFinished := false
			if observer != nil {
				// The marker lets lower layers record that they already captured
				// or deliberately handled an error, so this middleware does not
				// emit a duplicate generic 5xx Sentry event after the handler returns.
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
				handlerCtx, span := observer.StartSpan(spanCtx, observability.SpanContext{
					Operation: "http.handler",
					Component: "http",
					Attributes: observability.EventContext{
						"component":   "http",
						"operation":   "http.handler",
						"http_method": req.Method,
					},
				})
				handlerSpan = span
				req = req.WithContext(handlerCtx)
				defer func() {
					// If the downstream handler panics, normal span finalization below
					// will be skipped. Close the span here so traces are not left open.
					if handlerSpan != nil && !handlerSpanFinished {
						handlerSpan.SetAttributes(observability.EventContext{
							"component":     "http",
							"operation":     "http.handler",
							"route_pattern": observability.RoutePattern(req),
							"http_method":   req.Method,
							"result":        "error",
						})
						handlerSpan.Finish("error")
					}
					if traceSpan != nil && !spanFinished {
						traceSpan.Finish("error")
					}
				}()
			}
			// responseLogger captures the status and response byte count while
			// preserving normal ResponseWriter behavior for the wrapped handler.
			rw := &responseLogger{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, req)
			if observer != nil {
				routePattern := observability.RoutePattern(req)
				observability.RecordRoutePattern(req.Context(), routePattern)
				duration := time.Since(started)
				observedStatus := rw.status
				result := "success"
				if errors.Is(req.Context().Err(), context.Canceled) {
					// The peer has already gone away, so any downstream 5xx response
					// is not a server failure and cannot be delivered. Keep the
					// conventional 499 classification internal to telemetry.
					observedStatus = statusClientClosedRequest
					result = "canceled"
				} else if rw.status >= http.StatusInternalServerError {
					result = "error"
				}
				observer.ObserveHTTPRequest(req.Method, routePattern, observedStatus, duration, rw.bytes)
				if handlerSpan != nil {
					handlerSpan.SetAttributes(observability.EventContext{
						"component":         "http",
						"operation":         "http.handler",
						"route_pattern":     routePattern,
						"http_method":       req.Method,
						"http_status":       observedStatus,
						"http_status_class": strconv.Itoa(observedStatus/100) + "xx",
						"duration":          duration.String(),
						"result":            result,
					})
				}
				if traceSpan != nil {
					traceSpan.SetTransactionName(req.Method + " " + routePattern)
					traceSpan.SetAttributes(observability.EventContext{
						"component":         "http",
						"route_pattern":     routePattern,
						"http_method":       req.Method,
						"http_status":       observedStatus,
						"http_status_class": strconv.Itoa(observedStatus/100) + "xx",
						"duration":          duration.String(),
						"result":            result,
						"run_id":            GetRunID(req.Context()),
					})
				}
				// If a request ends as a 5xx and no deeper layer captured a more
				// specific error, emit one generic HTTP error event as a fallback.
				if observedStatus >= http.StatusInternalServerError && !observability.CaptureRecorded(req.Context()) {
					observer.CaptureError(req.Context(), observability.EventContext{
						"component":         "http",
						"route_pattern":     routePattern,
						"http_method":       req.Method,
						"http_status":       observedStatus,
						"http_status_class": strconv.Itoa(observedStatus/100) + "xx",
						"error_category":    "server",
						"duration":          duration.String(),
						"run_id":            GetRunID(req.Context()),
					}, errors.New("http server error response"))
				}
				if handlerSpan != nil {
					handlerSpan.Finish(result)
					handlerSpanFinished = true
				}
				if traceSpan != nil {
					traceSpan.Finish(result)
					spanFinished = true
				}
			}
		})
	}
}

// HTTPInFlight records active requests. It should be applied inside ServeMux
// route handlers, after Request.Pattern has been populated, so the gauge can
// use the same bounded route label as completed request metrics.
func HTTPInFlight(observer *observability.Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if observer == nil {
				next.ServeHTTP(w, r)
				return
			}
			routePattern := observability.RoutePattern(r)
			observability.RecordRoutePattern(r.Context(), routePattern)
			inFlightRoutePattern := observer.BeginHTTPRequest(r.Method, routePattern)
			defer observer.EndHTTPRequest(r.Method, inFlightRoutePattern)
			next.ServeHTTP(w, r)
		})
	}
}
