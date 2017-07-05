package main

import (
	"flag"
	"github.com/sergeyfast/zenrpc"
	"github.com/sergeyfast/zenrpc/testdata"
	"log"
	"net/http"
	"os"
)

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
