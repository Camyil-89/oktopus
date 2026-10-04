package server_test

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/config"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/server"
)

func TestApplyListenBindErrorNotActive(t *testing.T) {
	t.Parallel()

	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()

	addr := hold.Addr().String()
	logger := log.New(io.Discard, "", 0)
	m := server.NewManager(&hooks.Hooks{}, logger)

	err = m.Apply(context.Background(), config.Config{
		Listen:  addr,
		Connect: config.ConnectTunnel,
	}, acl.EmptyEngine(), nil)
	if err == nil {
		t.Fatal("expected bind error")
	}
	if m.ProxyActive() {
		t.Fatal("proxy must not be active when listen failed")
	}
	if got := m.ProxyLastStartError(); got != "listen_address_in_use" {
		t.Fatalf("expected listen_address_in_use, got %q", got)
	}
}

func TestApplyListenSuccessActive(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)
	m := server.NewManager(&hooks.Hooks{}, logger)

	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := hold.Addr().String()
	hold.Close()

	err = m.Apply(context.Background(), config.Config{
		Listen:  addr,
		Connect: config.ConnectTunnel,
	}, acl.EmptyEngine(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !m.ProxyActive() {
		t.Fatal("proxy should be active after successful listen")
	}
	if m.ProxyLastStartError() != "" {
		t.Fatalf("start error should be cleared, got %q", m.ProxyLastStartError())
	}
	m.Stop()
}

func TestRunContextListenBindErrorWaitsForCancel(t *testing.T) {
	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()

	addr := hold.Addr().String()
	logger := log.New(io.Discard, "", 0)
	m := server.NewManager(&hooks.Hooks{}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- m.RunContext(ctx, config.Config{
			Listen:  addr,
			Connect: config.ConnectTunnel,
		}, acl.EmptyEngine(), nil)
	}()

	select {
	case err := <-done:
		t.Fatalf("unexpected early exit: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunContext: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunContext did not stop after cancel")
	}
}
