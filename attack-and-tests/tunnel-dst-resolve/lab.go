package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	labListen = "127.0.0.1:9666"
	labMarker = "TUNNEL_INTERNAL_HIT"
)

func cmdLab(args []string) int {
	if len(args) > 0 {
		log.Printf("lab: лишние аргументы %v", args)
	}
	log.SetFlags(0)
	stop, err := startLab()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	log.Printf("lab: TCP %s (после CONNECT шлите PROBE\\r\\n)", labListen)
	log.Println("bypass: CONNECT localhost:9666 (системный DNS → 127.0.0.1)")
	_ = stop
	select {}
}

func startLab() (func(), error) {
	ln, err := net.Listen("tcp", labListen)
	if err != nil {
		return nil, err
	}
	go serveLab(ln)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitListen(ctx, labListen); err != nil {
		ln.Close()
		return nil, err
	}
	return func() { _ = ln.Close() }, nil
}

func serveLab(ln net.Listener) {
	log.Printf("lab: listening tcp://%s", labListen)
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go handleLabConn(c)
	}
}

func handleLabConn(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)
	line, _ := br.ReadString('\n')
	_, _ = fmt.Fprintf(c, "%s peer=%q probe=%q\n", labMarker, c.RemoteAddr().String(), strings.TrimSpace(line))
}

func waitListen(ctx context.Context, addr string) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s: %v", addr, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
