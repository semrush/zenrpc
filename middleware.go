package zenrpc

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Logger is middleware for JSON-RPC 2.0 Server.
// It's just an example for middleware, will be refactored later.
func Logger(l *log.Logger) MiddlewareFunc {
	return func(h InvokeFunc) InvokeFunc {
		return func(c Context, method string, params json.RawMessage) Response {
			start, ip := time.Now(), "<nil>"
			if req := c.Request(); req != nil {
				ip = req.RemoteAddr
			}

			r := h(c, method, params)
			l.Printf("ip=%s method=%s.%s duration=%v params=%s err=%s", ip, c.Namespace(), method, time.Since(start), params, r.Error)

			return r
		}
	}
}

// Metrics is a middleware for logging duration of RPC requests via Prometheus. Default AppName is zenrpc.
// It exposes two metrics: appName_rpc_error_requests_count and appName_rpc_responses_duration_seconds.
func Metrics(appName string) MiddlewareFunc {
	if appName == "" {
		appName = "zenrpc"
	}

	rpcErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: appName,
		Subsystem: "rpc",
		Name:      "error_requests_count",
		Help:      "Error requests count by method and error code.",
	}, []string{"method", "code"})

	rpcDurations := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: appName,
		Subsystem: "rpc",
		Name:      "responses_duration_seconds",
		Help:      "Response time by method and error code.",
	}, []string{"method", "code"})

	prometheus.MustRegister(rpcErrors, rpcDurations)

	return func(h InvokeFunc) InvokeFunc {
		return func(c Context, method string, params json.RawMessage) Response {
			start, code := time.Now(), ""
			r := h(c, method, params)

			// log metrics
			if n := c.Namespace(); n != "" {
				method = n + "." + method
			}

			if r.Error != nil {
				code = strconv.Itoa(r.Error.Code)
				rpcErrors.WithLabelValues(method, code).Inc()
			}

			rpcDurations.WithLabelValues(method, code).Observe(time.Since(start).Seconds())

			return r
		}
	}
}
