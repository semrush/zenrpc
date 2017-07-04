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
func (s *Server) process(ctx context.Context, message json.RawMessage) interface{} {

	requests := []Request{}

	// parsing batch requests
	batch := false
	for _, b := range message {
		if unicode.IsSpace(rune(b)) {
			continue
		}

		if b == '[' {
			batch = true
		}
		break
	}

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
	}

	// running request asynchronously
	reqLen := len(requests)
	respChan := make(chan Response, reqLen)
	var wg sync.WaitGroup
	wg.Add(reqLen)

	for _, req := range requests {
		// running request in gorutine
		go func() {
			if req.Id == nil {
				wg.Done()
				s.processRequest(ctx, &req)
			} else {
				r := s.processRequest(ctx, &req)
				r.Id = req.Id
				respChan <- r
				wg.Done()
			}
		}()
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

	// sending single response or array of responses if batch
	if batch {
		return responses
	} else if len(responses) == 0 {
		// no responses -> all requests are notifications
		return nil
	} else {
		return responses[0]
	}
}

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
