package service

import (
	"testing"

	"github.com/google/uuid"
)

func Test_normalizeSyncInputs_preservesClientID(t *testing.T) {
	instanceID := uuid.MustParse("018f0000-0000-7000-8000-000000000001")
	clientID := uuid.MustParse("018f0000-0000-7000-8000-000000000099")
	rules, err := normalizeSyncInputs(instanceID, []SyncRuleInput{{
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
	if len(rules) != 1 || rules[0].ID != clientID || rules[0].InstanceID != instanceID {
		t.Fatalf("got id=%v instance=%v", rules[0].ID, rules[0].InstanceID)
	}
}

func strPtr(s string) *string { return &s }
