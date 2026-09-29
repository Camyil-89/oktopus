package metrics

import "sync/atomic"

var activeWebSockets atomic.Int64

// IncActiveWebSocket — после успешного 101 и старта relay (MITM).
func IncActiveWebSocket() {
	activeWebSockets.Add(1)
}

// DecActiveWebSocket — при закрытии WS relay.
func DecActiveWebSocket() {
	activeWebSockets.Add(-1)
}

// ActiveWebSocketConnections — число активных WebSocket-сессий через прокси (MITM).
func ActiveWebSocketConnections() int {
	n := activeWebSockets.Load()
	if n < 0 {
		return 0
	}
	return int(n)
}
