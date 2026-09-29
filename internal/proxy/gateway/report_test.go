package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/proxy/accesslog"
)

type recordSpy struct {
	entries []accesslog.Entry
}

func (s *recordSpy) Record(e accesslog.Entry) {
	s.entries = append(s.entries, e)
}

func TestRecordGatewayError_matchesPageID(t *testing.T) {
	_ = Reload()
	spy := &recordSpy{}
	req := httptest.NewRequest(http.MethodGet, "https://example.com/path", nil)
	err := errSample("dial tcp: connection refused")

	page := RecordGatewayError(context.Background(), spy, req, err)
	if len(spy.entries) != 1 {
		t.Fatalf("entries: %d", len(spy.entries))
	}
	e := spy.entries[0]
	if e.ID != page.ErrorID {
		t.Fatalf("id mismatch log=%s page=%s", e.ID, page.ErrorID)
	}
	if e.RuleRef != accesslog.RuleRefGatewayError {
		t.Fatalf("ruleRef: %q", e.RuleRef)
	}
	if e.GatewayErrorType != ErrTypeDial {
		t.Fatalf("type: %q", e.GatewayErrorType)
	}
	if e.InspectError != err.Error() {
		t.Fatalf("message: %q", e.InspectError)
	}
	if _, parseErr := uuid.Parse(page.ErrorID.String()); parseErr != nil {
		t.Fatal(parseErr)
	}
}
