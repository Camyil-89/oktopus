package service

import (
	"testing"

	"github.com/google/uuid"
)

func Test_normalizeSyncInputs_preservesClientID(t *testing.T) {
	clientID := uuid.MustParse("018f0000-0000-7000-8000-000000000099")
	rules, err := normalizeSyncInputs([]SyncRuleInput{{
		ID:        strPtr(clientID.String()),
		Name:      "r1",
		Script:    strPtr("function inspect(ctx) return false end"),
		Action:    0,
		Enabled:   true,
		SortOrder: 0,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].ID != clientID {
		t.Fatalf("got %v", rules[0].ID)
	}
}

func strPtr(s string) *string { return &s }
