package aclrun

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"oktopus/attack-and-tests/poclib"
)

const aclHTTPLabAddr = "127.0.0.1:9090"

func startACLHTTPLab() (func(), error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "PUBLIC_OK\n")
	})
	mux.HandleFunc("/other", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OTHER_OK\n")
	})
	mux.HandleFunc("/secret/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, poclib.SecretMarker+"\n")
	})
	srv := &http.Server{Addr: aclHTTPLabAddr, Handler: mux}
	ln, err := net.Listen("tcp", aclHTTPLabAddr)
	if err != nil {
		return nil, err
	}
	go func() { _ = srv.Serve(ln) }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := poclib.WaitListen(ctx, aclHTTPLabAddr); err != nil {
		ln.Close()
		return nil, err
	}
	return func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}, nil
}
