package zenrpc

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// Logger is middleware for JSON-RPC 2.0 Server.
// It's just an example for middleware, will be refactored later.
func Logger(l *log.Logger) MiddlewareFunc {
	return func(h InvokeFunc) InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) Response {
			start, ip := time.Now(), "<nil>"
			if req, ok := RequestFromContext(ctx); ok && req != nil {
				ip = req.RemoteAddr
			}

			r := h(ctx, method, params)
			l.Printf("ip=%s method=%s.%s duration=%v params=%s err=%s", ip, NamespaceFromContext(ctx), method, time.Since(start), params, r.Error)

			return r
		}
	}
}
