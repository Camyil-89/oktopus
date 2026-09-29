package accesslog

import (
	"log"
	"time"
)

// LogRecorder пишет в стандартный logger (для отладки).
type LogRecorder struct {
	Logger *log.Logger
}

func (r LogRecorder) Record(e Entry) {
	logger := r.Logger
	if logger == nil {
		logger = log.Default()
	}
	user := "null"
	if e.User != nil {
		user = *e.User
	}
	logger.Printf(
		"acl decision action=%d src=%s dst=%s user=%s spend=%s url=%q",
		e.Action,
		e.SourceAddress,
		e.DestinationAddress,
		user,
		e.SpendTime.Round(time.Microsecond),
		e.FullURLRequest,
	)
}
