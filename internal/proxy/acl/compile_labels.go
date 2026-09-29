package acl

import "sync"

// Сессия intern для label/хостов при одной сборке engine (сбрасывается в squid.Analyze).

type compileLabelIntern struct {
	mu    sync.Mutex
	table map[string]string
}

var activeLabelIntern compileLabelIntern

// BeginCompileLabels сбрасывает таблицу intern перед компиляцией ACL.
func BeginCompileLabels() {
	activeLabelIntern = compileLabelIntern{table: make(map[string]string)}
}

// EndCompileLabels освобождает таблицу intern после сборки.
func EndCompileLabels() {
	activeLabelIntern.mu.Lock()
	activeLabelIntern.table = nil
	activeLabelIntern.mu.Unlock()
}

func internCompileLabel(s string) string {
	activeLabelIntern.mu.Lock()
	defer activeLabelIntern.mu.Unlock()
	if activeLabelIntern.table == nil {
		return s
	}
	if v, ok := activeLabelIntern.table[s]; ok {
		return v
	}
	activeLabelIntern.table[s] = s
	return s
}
