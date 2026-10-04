package wsactive

import "sync/atomic"

var active atomic.Int64

// Inc — после успешного 101 и старта relay.
func Inc() {
	active.Add(1)
}

// Dec — при закрытии WS relay.
func Dec() {
	active.Add(-1)
}

// Count — число активных WebSocket-сессий.
func Count() int {
	n := active.Load()
	if n < 0 {
		return 0
	}
	return int(n)
}
