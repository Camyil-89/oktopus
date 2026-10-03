package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"oktopus/internal/apperr"
)

func TestMapProxyApplyMITMCA(t *testing.T) {
	inner := fmt.Errorf("read ca cert: open config/ca.crt: no such file or directory")
	err := apperr.MapProxyApply(fmt.Errorf("mitm requires CA (config/ca.crt, config/ca.key): %w", inner))
	code, ok := apperr.CodeOf(err)
	if !ok || code != apperr.ProxyMITMCAMissing {
		t.Fatalf("expected proxy_mitm_ca_missing, got %v ok=%v", code, ok)
	}
}

func TestMapProxyApplyPassthrough(t *testing.T) {
	raw := errors.New("some upstream failure")
	err := apperr.MapProxyApply(raw)
	if _, ok := apperr.CodeOf(err); ok {
		t.Fatal("expected uncoded passthrough")
	}
	if err.Error() != raw.Error() {
		t.Fatalf("got %q", err.Error())
	}
}
