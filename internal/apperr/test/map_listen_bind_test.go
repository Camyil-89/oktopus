package apperr_test

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"oktopus/internal/apperr"
)

func TestMapListenBind(t *testing.T) {
	inner := &net.OpError{Op: "listen", Err: errors.New("bind: address already in use")}
	err := apperr.MapProxyApply(fmt.Errorf("listen tcp 127.0.0.1:8080: %w", inner))
	code, ok := apperr.CodeOf(err)
	if !ok || code != apperr.ListenAddressInUse {
		t.Fatalf("expected listen_address_in_use, got %v ok=%v", code, ok)
	}
}

func TestProxyStartErrorMessage(t *testing.T) {
	inner := &net.OpError{Op: "listen", Err: errors.New("Only one usage of each socket address")}
	got := apperr.ProxyStartErrorMessage(fmt.Errorf("listen tcp 127.0.0.1:8000: %w", inner))
	if got != "listen_address_in_use" {
		t.Fatalf("got %q", got)
	}
}
