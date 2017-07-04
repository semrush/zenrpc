package zenrpc

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// Invoker implements service handler.
type Invoker interface {
	Invoke(ctx context.Context, method string, params json.RawMessage) Response
}

// Service is as struct for discovering JSON-RPC 2.0 serivces for generator cmd.
type Service struct{}

// Server is JSON-RPC 2.0 Server.
type Server struct {
	services map[string]Invoker
}

// NewServer returns new JSON-RPC 2.0 Server.
func NewServer() Server {
	return Server{
		services: make(map[string]Invoker),
	}
}

// Register registers new service for given namespace. For public namespace use empty string.
func (s *Server) Register(namespace string, service Invoker) {
	s.services[namespace] = service
}

// process process JSON-RPC 2.0 message, invokes correct method for namespace and returns JSON-RPC 2.0 Response.
func (s *Server) process(ctx context.Context, message json.RawMessage) Response {
	req := &Request{}

	// unmarshal request
	// TODO process batch
	if err := json.Unmarshal(message, req); err != nil {
		return NewResponseError(nil, ParseError, "", nil)
	}

	// checks for json-rpc version and method
	if req.Version != Version || req.Method == "" {
		return NewResponseError(req.Id, InvalidRequest, "", nil)
	}

	// convert method to lower and find namespace
	lowerM := strings.ToLower(req.Method)
	sp := strings.SplitN(lowerM, ".", 2)
	namespace, method := "", lowerM
	if len(sp) == 2 {
		namespace, method = sp[0], sp[1]
	}

	if _, ok := s.services[namespace]; !ok {
		return NewResponseError(req.Id, MethodNotFound, "", nil)
	}

	// TODO Notifications

	resp := s.services[namespace].Invoke(ctx, method, req.Params)
	resp.Id = req.Id

	return resp
}

// ServeHTTP process JSON-RPC 2.0 requests via HTTP.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	var data Response

	ctx := context.WithValue(context.Background(), "IP", r.RemoteAddr)

	if err != nil {
		data = NewResponseError(nil, ParseError, "", nil)
	} else {
		data = s.process(ctx, b)
	}

	if _, err := w.Write(data.JSON()); err != nil {
		// TODO error
		return
	}
}
