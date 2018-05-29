package zenrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/semrush/zenrpc/smd"
)

type contextKey string

const (
	// defaultBatchMaxLen is default value of BatchMaxLen option in rpc Server options.
	defaultBatchMaxLen = 10

	// defaultTargetURL is default value for SMD target url.
	defaultTargetURL = "/"

	// context key for http.Request object.
	requestKey contextKey = "request"

	// context key for namespace.
	namespaceKey contextKey = "namespace"

	// context key for ID.
	IDKey contextKey = "id"

	// contentTypeJSON is default content type for HTTP transport.
	contentTypeJSON = "application/json"
)

// MiddlewareFunc is a function for executing as middleware.
type MiddlewareFunc func(InvokeFunc) InvokeFunc

// InvokeFunc is a function for processing single JSON-RPC 2.0 Request after validation and parsing.
type InvokeFunc func(context.Context, string, json.RawMessage) Response

// Invoker implements service handler.
type Invoker interface {
	Invoke(ctx context.Context, method string, params json.RawMessage) Response
	SMD() smd.ServiceInfo
}

// Service is as struct for discovering JSON-RPC 2.0 services for zenrpc generator cmd.
type Service struct{}

// Options is options for JSON-RPC 2.0 Server.
type Options struct {
	// BatchMaxLen sets maximum quantity of requests in single batch.
	BatchMaxLen int

	// TargetURL is RPC endpoint.
	TargetURL string

	// ExposeSMD exposes SMD schema with ?smd GET parameter.
	ExposeSMD bool

	// DisableTransportChecks disables Content-Type and methods checks. Use only for development mode.
	DisableTransportChecks bool

	// AllowCORS adds header Access-Control-Allow-Origin with *.
	AllowCORS bool

	// Upgrader sets options for gorilla websocket. If nil, default options will be used
	Upgrader *websocket.Upgrader

	// Removes data field from response error
	HideErrorDataField bool
}

// Server is JSON-RPC 2.0 Server.
type Server struct {
	services   map[string]Invoker
	options    Options
	middleware []MiddlewareFunc
	logger     Printer
}

// NewServer returns new JSON-RPC 2.0 Server.
func NewServer(opts Options) Server {
	// For safety reasons we do not allowing to much requests in batch
	if opts.BatchMaxLen == 0 {
		opts.BatchMaxLen = defaultBatchMaxLen
	}

	if opts.TargetURL == "" {
		opts.TargetURL = defaultTargetURL
	}

	if opts.Upgrader == nil {
		opts.Upgrader = &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return opts.AllowCORS },
		}
	}

	return Server{
		services: make(map[string]Invoker),
		options:  opts,
	}
}

// Use registers middleware.
func (s *Server) Use(m ...MiddlewareFunc) {
	s.middleware = append(s.middleware, m...)
}

// Register registers new service for given namespace. For public namespace use empty string.
func (s *Server) Register(namespace string, service Invoker) {
	s.services[strings.ToLower(namespace)] = service
}

// RegisterAll registers all services listed in map.
func (s *Server) RegisterAll(services map[string]Invoker) {
	for ns, srv := range services {
		s.Register(ns, srv)
	}
}

// SetLogger sets logger for debug
func (s *Server) SetLogger(printer Printer) {
	s.logger = printer
}

// process process JSON-RPC 2.0 message, invokes correct method for namespace and returns JSON-RPC 2.0 Response.
func (s *Server) process(ctx context.Context, message json.RawMessage) interface{} {
	requests := []Request{}
	// parsing batch requests
	batch := IsArray(message)

	// making not batch request looks like batch to simplify further code
	if !batch {
		message = append(append([]byte{'['}, message...), ']')
	}

	// unmarshal request(s)
	if err := json.Unmarshal(message, &requests); err != nil {
		return NewResponseError(nil, ParseError, "", nil)
	}

	// if there no requests to process
	if len(requests) == 0 {
		return NewResponseError(nil, InvalidRequest, "", nil)
	} else if len(requests) > s.options.BatchMaxLen {
		return NewResponseError(nil, InvalidRequest, "", "max requests length in batch exceeded")
	}

	// process single request: if request single and not notification  - just run it and return result
	if !batch && requests[0].ID != nil {
		return s.processRequest(ctx, requests[0])
	}

	// process batch requests
	if res := s.processBatch(ctx, requests); len(res) > 0 {
		return res
	}

	return nil
}

// processBatch process batch requests with context.
func (s Server) processBatch(ctx context.Context, requests []Request) []Response {
	reqLen := len(requests)

	// running requests in batch asynchronously
	respChan := make(chan Response, reqLen)

	var wg sync.WaitGroup
	wg.Add(reqLen)

	for _, req := range requests {
		// running request in goroutine
		go func(req Request) {
			if req.ID == nil {
				// ignoring response if request is notification
				wg.Done()
				s.processRequest(ctx, req)
			} else {
				respChan <- s.processRequest(ctx, req)
				wg.Done()
			}
		}(req)
	}

	// waiting to complete
	wg.Wait()
	close(respChan)

	// collecting responses
	responses := make([]Response, 0, reqLen)
	for r := range respChan {
		responses = append(responses, r)
	}

	// no responses -> all requests are notifications
	if len(responses) == 0 {
		return nil
	}
	return responses
}

// processRequest processes a single request in service invoker.
func (s Server) processRequest(ctx context.Context, req Request) Response {
	// checks for json-rpc version and method
	if req.Version != Version || req.Method == "" {
		return NewResponseError(req.ID, InvalidRequest, "", nil)
	}

	// convert method to lower and find namespace
	lowerM := strings.ToLower(req.Method)
	sp := strings.SplitN(lowerM, ".", 2)
	namespace, method := "", lowerM
	if len(sp) == 2 {
		namespace, method = sp[0], sp[1]
	}

	if _, ok := s.services[namespace]; !ok {
		return NewResponseError(req.ID, MethodNotFound, "", nil)
	}

	// set namespace to context
	ctx = newNamespaceContext(ctx, namespace)

	// set id to context
	ctx = newIDContext(ctx, req.ID)

	// set middleware to func
	f := InvokeFunc(s.services[namespace].Invoke)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		f = s.middleware[i](f)
	}

	// invoke func with middleware
	resp := f(ctx, method, req.Params)
	resp.ID = req.ID

	if s.options.HideErrorDataField && resp.Error != nil {
		resp.Error.Data = nil
	}

	return resp
}

func (s Server) printf(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Printf(format, v...)
	}
}

// SMD returns Service Mapping Description object with all registered methods.
func (s Server) SMD() smd.Schema {
	sch := smd.Schema{
		Transport:   "POST",
		Envelope:    "JSON-RPC-2.0",
		SMDVersion:  "2.0",
		ContentType: contentTypeJSON,
		Target:      s.options.TargetURL,
		Services:    make(map[string]smd.Service),
	}

	for n, v := range s.services {
		info, namespace := v.SMD(), ""
		if n != "" {
			namespace = n + "."
		}

		for m, d := range info.Methods {
			method := namespace + m
			sch.Services[method] = d
			sch.Description += info.Description // TODO formatting
		}
	}

	return sch
}

// IsArray checks json message if it array or object.
func IsArray(message json.RawMessage) bool {
	for _, b := range message {
		if unicode.IsSpace(rune(b)) {
			continue
		}

		if b == '[' {
			return true
		}
		break
	}

	return false
}

// ConvertToObject converts json array into object using key by index from keys array.
func ConvertToObject(keys []string, params json.RawMessage) (json.RawMessage, error) {
	paramCount := len(keys)

	rawParams := []json.RawMessage{}
	if err := json.Unmarshal(params, &rawParams); err != nil {
		return nil, err
	}

	rawParamCount := len(rawParams)
	if paramCount < rawParamCount {
		return nil, fmt.Errorf("invalid params number, expected %d, got %d", paramCount, len(rawParams))
	}

	buf := bytes.Buffer{}
	if _, err := buf.WriteString(`{`); err != nil {
		return nil, err
	}

	for i, p := range rawParams {
		// Writing key
		if _, err := buf.WriteString(`"` + keys[i] + `":`); err != nil {
			return nil, err
		}

		// Writing value
		if _, err := buf.Write(p); err != nil {
			return nil, err
		}

		// Writing trailing comma if not last argument
		if i != rawParamCount-1 {
			if _, err := buf.WriteString(`,`); err != nil {
				return nil, err
			}
		}

	}
	if _, err := buf.WriteString(`}`); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// newRequestContext creates new context with http.Request.
func newRequestContext(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestKey, req)
}

// RequestFromContext returns http.Request from context.
func RequestFromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestKey).(*http.Request)
	return r, ok
}

// newNamespaceContext creates new context with current method namespace.
func newNamespaceContext(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, namespaceKey, namespace)
}

// NamespaceFromContext returns method's namespace from context.
func NamespaceFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(namespaceKey).(string); ok {
		return r
	}

	return ""
}

// newIDContext creates new context with current request ID.
func newIDContext(ctx context.Context, ID *json.RawMessage) context.Context {
	return context.WithValue(ctx, IDKey, ID)
}

// IDFromContext returns request ID from context.
func IDFromContext(ctx context.Context) *json.RawMessage {
	if r, ok := ctx.Value(IDKey).(*json.RawMessage); ok {
		return r
	}

	return nil
}
