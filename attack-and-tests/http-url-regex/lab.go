package main

import (
	"context"
	"fmt"
	"log"
	"net"
	stdhttp "net/http"
	"time"

	"oktopus/attack-and-tests/poclib"
)

const labHTTP = "127.0.0.1:9090"

func cmdLab(args []string) int {
	stop, _ := startLab()
	_ = stop
	select {}
}

func startLab() (func(), error) {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/public", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = fmt.Fprint(w, "PUBLIC_OK\n")
	})
	mux.HandleFunc("/secret/data", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = fmt.Fprint(w, poclib.SecretMarker+"\n")
	})
	srv := &stdhttp.Server{Addr: labHTTP, Handler: mux}
	ln, err := net.Listen("tcp", labHTTP)
	if err != nil {
		return nil, err
	}
	go func() {
		log.Printf("lab: http://%s", labHTTP)
		_ = srv.Serve(ln)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := poclib.WaitListen(ctx, labHTTP); err != nil {
		return nil, err
	}
	return func() { _ = srv.Shutdown(context.Background()) }, nil
}
