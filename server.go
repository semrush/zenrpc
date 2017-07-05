package zenrpc

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"unicode"
)

const DefaultBatchMaxLen = 10

// Invoker implements service handler.
type Invoker interface {
	Invoke(ctx context.Context, method string, params json.RawMessage) Response
}

// Service is as struct for discovering JSON-RPC 2.0 serivces for generator cmd.
type Service struct{}

// Options is options for JSON-RPC 2.0 Server
type Options struct {
	// maxBatchRequests sets maximum quantity of requests in single batch
	BatchMaxLen int
}

// Server is JSON-RPC 2.0 Server.
type Server struct {
	services map[string]Invoker
	options  Options
}

// NewServer returns new JSON-RPC 2.0 Server.
func NewServer(opts Options) Server {
	// For safety reasons we do not allowing to much requests in batch
	if opts.BatchMaxLen == 0 {
		opts.BatchMaxLen = DefaultBatchMaxLen
	}

	return Server{
		services: make(map[string]Invoker),
		options:  opts,
	}
}

// Register registers new service for given namespace. For public namespace use empty string.
func (s *Server) Register(namespace string, service Invoker) {
	s.services[namespace] = service
}

// process process JSON-RPC 2.0 message, invokes correct method for namespace and returns JSON-RPC 2.0 Response.
func (s *Server) process(ctx context.Context, message json.RawMessage) interface{} {

	requests := []Request{}

	// parsing batch requests
	batch := isBatch(message)

	// making not batch request looks like batch to simplify further code
	if !batch {
		message = append(append([]byte{'['}, message...), ']')
	}

	// unmarshal request(s)
	if err := json.Unmarshal(message, &requests); err != nil {
		return NewResponseError(nil, ParseError, "", nil)
	}

	reqLen := len(requests)

	// if there no requests to process
	if reqLen == 0 {
		return NewResponseError(nil, InvalidRequest, "", nil)
	}

	if reqLen > s.options.BatchMaxLen {
		return NewResponseError(nil, InvalidRequest, "", "max requests length in batch exceeded")
	}

	// if request single and not notification  - just run it and return result
	if !batch && requests[0].Id != nil {
		return s.processRequest(ctx, &requests[0])
	}

	// running requests in batch asynchronously
	respChan := make(chan Response, reqLen)
	var wg sync.WaitGroup
	wg.Add(reqLen)

	for i := range requests {
		// running request in goroutine
		go func(req *Request) {
			if req.Id == nil {
				// ignoring response if request is notification
				wg.Done()
				s.processRequest(ctx, req)
			} else {
				r := s.processRequest(ctx, req)
				r.Id = req.Id
				respChan <- r
				wg.Done()
			}
		}(&requests[i])
	}

	// TODO what if one of requests freezes?
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
	} else {
		return responses
	}
}

// processRequest processes a single request in service invoker
func (s Server) processRequest(ctx context.Context, req *Request) Response {
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

	resp := s.services[namespace].Invoke(ctx, method, req.Params)
	resp.Id = req.Id

	return resp
}

// ServeHTTP process JSON-RPC 2.0 requests via HTTP.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	var data interface{}

	ctx := context.WithValue(context.Background(), "IP", r.RemoteAddr)

	if err != nil {
		data = NewResponseError(nil, ParseError, "", nil)
	} else {
		data = s.process(ctx, b)
	}

	// if responses is empty -> all requests are notifications -> exit immediately
	if data == nil {
		return
	}

	mes, err := json.Marshal(data)
	if err != nil {
		return
	}

	if _, err := w.Write(mes); err != nil {
		// TODO error
		return
	}
}

// isBatch checks json message if it array or object
func isBatch(message json.RawMessage) bool {
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
