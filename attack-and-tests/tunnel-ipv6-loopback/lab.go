package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	labListen = "[::1]:9888"
	labMarker = "IPV6_LOOPBACK_HIT"
)

func cmdLab(args []string) int {
	log.SetFlags(0)
	stop, err := startLab()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	_ = stop
	select {}
}

func startLab() (func(), error) {
	ln, err := net.Listen("tcp6", "[::1]:9888")
	if err != nil {
		return nil, err
	}
	go func() {
		log.Printf("lab: listening tcp6://%s", labListen)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = fmt.Fprintf(conn, "%s\n", labMarker)
			}(c)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp6", "[::1]:9888", 200*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			ln.Close()
			return nil, fmt.Errorf("lab not ready: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return func() { _ = ln.Close() }, nil
}
