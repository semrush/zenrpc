# zenrpc: JSON-RPC 2.0 Server Implementation with SMD support [WIP]

[![Go Report Card](https://goreportcard.com/badge/github.com/semrush/zenrpc)](https://goreportcard.com/report/github.com/semrush/zenrpc) [![Build Status](https://travis-ci.org/semrush/zenrpc.svg?branch=master)](https://travis-ci.org/semrush/zenrpc) [![codecov](https://codecov.io/gh/semrush/zenrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/semrush/zenrpc)

`zenrpc` is a JSON-RPC 2.0 server library with Service Mapping Description support. 
It's built on top of `go generate` instead of reflection. 

# How to Use

```Service is struct with RPC methods, service represents RPC namespace.```

  1. Install zenrpc generator `go get github.com/semrush/zenrpc/zenrpc`
  1. Import `github.com/semrush/zenrpc` into our code with rpc service.
  1. Add trailing comment `//zenrpc` to your service or embed `zenrpc.Service` into your service struct.
  1. Write your funcs almost as usual.
  1. Do not forget run `go generate` or `zenrpc` for magic

### Accepted Method Signatures

    func(Service) Method([args]) (<value>, <error>)
    func(Service) Method([args]) <value>
    func(Service) Method([args]) <error>
    func(Service) Method([args])

- Value could be a pointer
- Error is error or *zenrpc.Error

## Example
```go
package main

import (
	"flag"
	"context"
	"errors"
	"math"
	"log"
	"net/http"
	"os"	
	
	"github.com/semrush/zenrpc"
	"github.com/semrush/zenrpc/testdata"
)

type ArithService struct{ zenrpc.Service }

// Sum sums two digits and returns error with error code as result and IP from context.
func (as ArithService) Sum(ctx context.Context, a, b int) (bool, *zenrpc.Error) {
	r, _ := zenrpc.RequestFromContext(ctx)

	return true, zenrpc.NewStringError(a+b, r.Host)
}

// Multiply multiples two digits and returns result.
func (as ArithService) Multiply(a, b int) int {
	return a * b
}

type Quotient struct {
	Quo, Rem int
}

func (as ArithService) Divide(a, b int) (quo *Quotient, err error) {
	if b == 0 {
		return nil, errors.New("divide by zero")
	} else if b == 1 {
		return nil, zenrpc.NewError(401, errors.New("we do not serve 1"))
	}

	return &Quotient{
		Quo: a / b,
		Rem: a % b,
	}, nil
}

// Pow returns x**y, the base-x exponential of y. If Exp is not set then default value is 2.
//zenrpc:exp:2
func (as ArithService) Pow(base float64, exp float64) float64 {
	return math.Pow(base, exp)
}

//go:generate zenrpc

func main() {
	addr := flag.String("addr", "localhost:9999", "listen address")
	flag.Parse()

	rpc := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true})
	rpc.Register("arith", testdata.ArithService{})
	rpc.Register("", testdata.ArithService{}) // public
	rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))

	http.Handle("/", rpc)

	log.Printf("starting arithsrv on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

```


## Magic comments

All comments are optional.

    Method comments
    //zenrpc:<method parameter>[:<default value>][whitespaces<description>]
    //zenrpc:<error code>[whitespaces<description>]
    //zenrpc:return[whitespaces<description>]
     
    Struct comments
    type MyService struct {} //zenrpc
    


# JSON-RPC 2.0 Supported Features

  * [x] Requests
    * [x] Single requests
    * [x] Batch requests
    * [x] Notifications
  * [x] Parameters
    * [x] Named
    * [x] Position
    * [x] Default values
  * [x] SMD Schema
    * [x] Input
    * [ ] Output
    * [ ] Codes
    * [ ] Scopes for OAuth

# Server Library Features

 * [x] go generate
 * [ ] Transports
   * [x] HTTP
   * [ ] Websocket
   * [ ] RabbitMQ
 * [x] Server middleware
   * [x] Basic support
   * [x] Metrics
   * [x] Logging