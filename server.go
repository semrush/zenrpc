package zenrpc

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// Executer implements service handler.
type Executer interface {
	Execute(method string, params json.RawMessage) Response
}

type Service struct {
}

type Server struct {
	services map[string]Executer
}

func NewServer() Server {
	return Server{
		services: make(map[string]Executer),
	}
}

func (s *Server) Register(namespace string, service Executer) {
	s.services[namespace] = service
}

func (s *Server) handle(message json.RawMessage) json.RawMessage {
	// TODO handle batch
	t := &Request{}
	if err := json.Unmarshal(message, t); err != nil {
		return NewResponseError(nil, ParseError, errorMessages[ParseError], nil).JSON()
	}

	if t.Version != "2.0" || t.Method == "" {
		return NewResponseError(t.Id, InvalidRequest, errorMessages[InvalidRequest], nil).JSON()
	}

	sp := strings.SplitN(t.Method, ".", 2)
	namespace, method := "", t.Method
	if len(sp) == 2 {
		namespace, method = sp[0], sp[1]
	}

	if _, ok := s.services[namespace]; !ok {
		return NewResponseError(t.Id, MethodNotFound, errorMessages[MethodNotFound], nil).JSON()
	}

	// TODO Notifications
	d := s.services[namespace].Execute(method, t.Params)
	b, _ := json.Marshal(d)
	return b
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	var data json.RawMessage

	if err != nil {
		data = NewResponseError(nil, ParseError, errorMessages[ParseError], nil).JSON()
	} else {
		data = s.handle(b)
	}

	if _, err := w.Write(data); err != nil {
		// TODO error
		return
	}
}
