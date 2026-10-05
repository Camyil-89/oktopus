package bytecount

import "github.com/google/uuid"

var (
	recordUp   func(uuid.UUID, int, bool)
	recordDown func(uuid.UUID, int, bool)
)

// SetRecordHooks подключает учёт байтов к metrics.Collector (init в metrics).
func SetRecordHooks(
	up func(uuid.UUID, int, bool),
	down func(uuid.UUID, int, bool),
) {
	recordUp = up
	recordDown = down
}
