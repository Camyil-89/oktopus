package poclib

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	stdhttp "net/http"
	"os"
	"time"
)

// StartTLSLab поднимает HTTPS сервер с телом marker на path.
func StartTLSLab(addr, marker, path string) (func(), error) {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc(path, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "%s via %s Host=%q\n", marker, addr, r.Host)
	})
	srv := &stdhttp.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	cert, err := LoadDevKeyPair()
	if err != nil {
		ln.Close()
		return nil, err
	}
	srv.TLSConfig.Certificates = []tls.Certificate{cert}
	go func() {
		log.Printf("lab: listening https://%s%s", addr, path)
		if err := srv.ServeTLS(ln, "", ""); err != nil && err != stdhttp.ErrServerClosed {
			log.Printf("lab %s: %v", addr, err)
			os.Exit(1)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := WaitListen(ctx, addr); err != nil {
		_ = srv.Shutdown(context.Background())
		return nil, err
	}
	return func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}, nil
}
