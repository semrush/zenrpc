package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/semrush/zenrpc"
	"github.com/semrush/zenrpc/testdata"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", "localhost:9999", "listen address")
	flag.Parse()

	const phonebook = "phonebook"

	rpc := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD:              true,
		AllowCORS:              true,
		DisableTransportChecks: true,
	})
	rpc.Register(phonebook, testdata.PhoneBook{DB: testdata.People})
	rpc.Register("arith", testdata.ArithService{})
	rpc.Register("", testdata.ArithService{}) // public

	rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
	rpc.Use(zenrpc.Metrics(""), testdata.SerialPeopleAccess(phonebook))

	http.Handle("/", rpc)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/doc", zenrpc.SMDBoxHandler)

	log.Printf("starting arithsrv on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
