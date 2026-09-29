package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entry — строка журнала ACL.
type Entry struct {
	ID                 uuid.UUID
	CreatedAt          time.Time
	SourceAddress      string
	DestinationAddress string
	UserName           *string
	DecideDurationUs   int64
	FullURL            string
	Action             int16
	InspectRuleID      *uuid.UUID
	DeniedBy           *string
	DecisionRuleRef    string // UUID или system_* — правило, определившее исход
	Extra              []byte // JSON при insert: search, inspect_error, ctx:log по id правила → ClickHouse KV
}
