package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrListNotFound = errors.New("proxy acl list not found")

const (
	ListSourceModeManual = "manual"
	ListSourceModeRemote = "remote"
)

// NamedList — именованный multiline list (src / dstdomain / port).
type NamedList struct {
	ID                  uuid.UUID
	Name                string
	ListType            string
	Body                string
	SourceMode          string
	SourceURL           string
	PollIntervalMinutes int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
