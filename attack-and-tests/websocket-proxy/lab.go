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

	"golang.org/x/net/websocket"
)

const (
	wssListen = "127.0.0.1:9443"
	wsListen  = "127.0.0.1:9080"
)

func cmdLab(args []string) int {
	if len(args) > 0 {
		log.Printf("lab: лишние аргументы %v", args)
	}
	log.SetFlags(0)
	stop, err := startLabServers()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	defer stop()
	log.Printf("wss (TLS)  https://%s/", wssListen)
	log.Printf("ws (HTTP)  http://%s/", wsListen)
	select {}
}

func startLabServers() (func(), error) {
	echo := websocket.Handler(func(ws *websocket.Conn) {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			return
		}
		_ = websocket.Message.Send(ws, "echo:"+msg)
	})

	muxTLS := stdhttp.NewServeMux()
	muxTLS.Handle("/", echo)
	muxPlain := stdhttp.NewServeMux()
	muxPlain.Handle("/", echo)

	tlsSrv := &stdhttp.Server{
		Addr:    wssListen,
		Handler: muxTLS,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	plainSrv := &stdhttp.Server{
		Addr:    wsListen,
		Handler: muxPlain,
	}

	s1, err := listenAndServeTLS(tlsSrv)
	if err != nil {
		return nil, err
	}
	lnPlain, err := net.Listen("tcp", wsListen)
	if err != nil {
		_ = s1.Shutdown(context.Background())
		return nil, err
	}
	go func() {
		log.Printf("lab: listening http://%s", wsListen)
		if err := plainSrv.Serve(lnPlain); err != nil && err != stdhttp.ErrServerClosed {
			log.Printf("lab plain: %v", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitListen(ctx, wssListen, wsListen); err != nil {
		_ = s1.Shutdown(context.Background())
		_ = plainSrv.Shutdown(context.Background())
		return nil, err
	}

	return func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = s1.Shutdown(shutdownCtx)
		_ = plainSrv.Shutdown(shutdownCtx)
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
			log.Printf("lab tls: %v", err)
			os.Exit(1)
		}
	}()
	return srv, nil
}
