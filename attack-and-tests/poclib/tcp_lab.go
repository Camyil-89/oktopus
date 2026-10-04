package poclib

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// StartTCPLab — TCP echo с marker на первой строке запроса.
func StartTCPLab(addr, marker string) (func(), error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go serveTCPLab(ln, marker)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := WaitListen(ctx, addr); err != nil {
		ln.Close()
		return nil, err
	}
	return func() { _ = ln.Close() }, nil
}

func serveTCPLab(ln net.Listener, marker string) {
	log.Printf("lab: listening tcp://%s", ln.Addr().String())
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func(conn net.Conn) {
			defer conn.Close()
			br := bufio.NewReader(conn)
			line, _ := br.ReadString('\n')
			_, _ = fmt.Fprintf(conn, "%s probe=%q\n", marker, strings.TrimSpace(line))
		}(c)
	}
}
