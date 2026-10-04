package metrics

import "oktopus/internal/proxy/wsactive"

// IncActiveWebSocket — после успешного 101 и старта relay (MITM / plain HTTP).
func IncActiveWebSocket() {
	wsactive.Inc()
}

// DecActiveWebSocket — при закрытии WS relay.
func DecActiveWebSocket() {
	wsactive.Dec()
}

// ActiveWebSocketConnections — число активных WebSocket-сессий через прокси.
func ActiveWebSocketConnections() int {
	return wsactive.Count()
}
