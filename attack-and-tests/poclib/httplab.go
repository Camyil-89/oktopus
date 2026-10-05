package poclib

import (
	"context"
	"fmt"
	"log"
	"net"
	stdhttp "net/http"
	"os"
	"time"
)

// StartHTTPLab — plain HTTP origin.
func StartHTTPLab(addr, marker, path string) (func(), error) {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc(path, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintf(w, "%s Host=%q\n", marker, r.Host)
	})
	srv := &stdhttp.Server{Addr: addr, Handler: mux}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		log.Printf("lab: listening http://%s%s", addr, path)
		if err := srv.Serve(ln); err != nil && err != stdhttp.ErrServerClosed {
			log.Printf("lab: %v", err)
			os.Exit(1)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := WaitListen(ctx, addr); err != nil {
		return nil, err
	}
	return func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}, nil
}
