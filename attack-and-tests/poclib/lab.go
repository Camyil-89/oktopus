package poclib

import (
	"context"
	"fmt"
	"net"
	"time"
)

// WaitListen ждёт, пока TCP-порты начнут принимать соединения.
func WaitListen(ctx context.Context, addrs ...string) error {
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
