package zenrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ArithService struct {
}

func (a *ArithService) Execute(method string, params json.RawMessage) Response {
	switch method {
	case "divide":
		var args = struct {
			A int `json:"b"`
			B int `json:"a"`
		}{}

		if err := json.Unmarshal(params, &args); err != nil {
			return Response{Error: &Error{Code: 401, Err: errors.New("divide by zero")}}
		}

		// set default values
		// possible err with custom marshaller

		if v, err := a.Divide(args.A, args.B); err != nil {
			return Response{Error: &Error{Err: err}} // TODO handle Error
		} else if r, err := json.Marshal(v); err != nil {
			return Response{Error: &Error{Err: err}}
		} else {
			return Response{Result: r}
		}
	}

	return Response{} // TODO
}

func (t *ArithService) Multiply(a, b int) int {
	return a * b
}

type Quotient struct {
	Quo, Rem int
}

func (t *ArithService) Divide(a, b int) (quo *Quotient, err error) {
	if b == 0 {
		return nil, errors.New("divide by zero")
	} else if b == 1 {
		return nil, &Error{Code: 401, Err: errors.New("we do not serve 1")}
	}

	return &Quotient{
		Quo: a / b,
		Rem: a % b,
	}, nil
}

func TestServer_ServeHTTP(t *testing.T) {
	s := NewServer()
	s.Register("arith", &ArithService{})

	ts := httptest.NewServer(http.HandlerFunc(s.ServeHTTP))
	defer ts.Close()

	v := bytes.NewBuffer([]byte(`{"jsonrpc": "2.0", "method": "arith.divide", "params": { "a": 42, "b": 23 }, "id": 1}`))
	res, err := http.Post(ts.URL, "application/json", v)
	if err != nil {
		log.Fatal(err)
	}

	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", greeting)
}
