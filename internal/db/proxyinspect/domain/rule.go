package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("proxy inspect rule not found")

const (
	ActionDenyOnMatch  int16 = 0
	ActionAllowOnMatch int16 = 1
)

type RuleSummary struct {
	ID           uuid.UUID
	Name         string
	Action       int16
	Enabled      bool
	SortOrder    int
	ScriptLength int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Rule struct {
	ID        uuid.UUID
	Name      string
	Script    string
	Action    int16
	Enabled   bool
	SortOrder int
	CreatedAt time.Time
	UpdatedAt time.Time
}
