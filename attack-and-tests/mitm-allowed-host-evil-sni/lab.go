package main

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

const (
	allowedListen  = "127.0.0.1:9443"
	internalListen = "127.0.0.1:9555"
)

func cmdLab(args []string) int {
	if len(args) > 0 {
		log.Printf("lab: лишние аргументы %v", args)
	}
	log.SetFlags(0)
	if _, err := startLabServers(); err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	logLabHints()
	select {}
}

func logLabHints() {
	log.Printf("allowed TLS  %s  GET /", allowedListen)
	log.Printf("internal TLS %s  GET /secret", internalListen)
	log.Println("CONNECT через прокси: localhost:9443")
}

func startLabServers() (func(), error) {
	allowed := newTLSServer(allowedListen, "ALLOWED_ORIGIN", "/")
	internal := newTLSServer(internalListen, "SECRET_INTERNAL_HIT", "/secret")

	s1, err := listenAndServeTLS(allowed)
	if err != nil {
		return nil, err
	}
	s2, err := listenAndServeTLS(internal)
	if err != nil {
		_ = s1.Shutdown(context.Background())
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitListen(ctx, allowedListen, internalListen); err != nil {
		_ = s1.Shutdown(context.Background())
		_ = s2.Shutdown(context.Background())
		return nil, err
	}

	return func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = s1.Shutdown(shutdownCtx)
		_ = s2.Shutdown(shutdownCtx)
	}, nil
}

func waitListen(ctx context.Context, addrs ...string) error {
	deadline := time.Now().Add(5 * time.Second)
	for _, addr := range addrs {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err == nil {
				conn.Close()
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for %s: %v", addr, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}

func newTLSServer(addr, body, path string) *stdhttp.Server {
	mux := stdhttp.NewServeMux()
	mux.HandleFunc(path, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "%s via %s Host=%q\n", body, addr, r.Host)
	})
	return &stdhttp.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
}

func listenAndServeTLS(srv *stdhttp.Server) (*stdhttp.Server, error) {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, err
	}
	cert, err := loadDevKeyPair()
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("%s: %w (запускайте из корня репозитория)", srv.Addr, err)
	}
	srv.TLSConfig.Certificates = []tls.Certificate{cert}
	go func() {
		log.Printf("lab: listening https://%s", srv.Addr)
		if err := srv.ServeTLS(ln, "", ""); err != nil && err != stdhttp.ErrServerClosed {
			log.Printf("lab %s: %v", srv.Addr, err)
			os.Exit(1)
		}
	}()
	return srv, nil
}
