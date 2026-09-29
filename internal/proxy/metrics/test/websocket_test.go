package metrics_test

import (
	"testing"

	"oktopus/internal/proxy/metrics"
)

func TestActiveWebSocketConnections(t *testing.T) {
	t.Parallel()
	if metrics.ActiveWebSocketConnections() != 0 {
		t.Fatal("expected 0")
	}
	metrics.IncActiveWebSocket()
	metrics.IncActiveWebSocket()
	if metrics.ActiveWebSocketConnections() != 2 {
		t.Fatalf("got %d", metrics.ActiveWebSocketConnections())
	}
	metrics.DecActiveWebSocket()
	if metrics.ActiveWebSocketConnections() != 1 {
		t.Fatalf("got %d", metrics.ActiveWebSocketConnections())
	}
	metrics.DecActiveWebSocket()
	if metrics.ActiveWebSocketConnections() != 0 {
		t.Fatalf("got %d", metrics.ActiveWebSocketConnections())
	}
}
