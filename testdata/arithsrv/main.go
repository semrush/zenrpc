package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sergeyfast/zenrpc"
	"github.com/sergeyfast/zenrpc/testdata"
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
	http.HandleFunc("/doc", BoxHandler)

	log.Printf("starting arithsrv on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// BoxHandler is a handler for SMDBox web app.
// TODO(sergeyfast): move to cdn
func BoxHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>SMD Box</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css">
<link href="https://raw.githubusercontent.com/mikhail-eremin/smd-box/master/dist/app.css" rel="stylesheet"></head>
<body>
<div id="json-rpc-root"></div>
<script type="text/javascript" src="https://raw.githubusercontent.com/mikhail-eremin/smd-box/master/dist/app.js"></script></body>
</html>
	`))
}
