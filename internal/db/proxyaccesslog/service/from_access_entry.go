package service

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/domain"
	"oktopus/internal/id"
	"oktopus/internal/proxy/accesslog"
)

func fromAccessEntry(e accesslog.Entry) domain.Entry {
	var user *string
	if e.User != nil && *e.User != "" {
		user = e.User
	}
	at := e.At
	if at.IsZero() {
		at = time.Now()
	}
	var deniedBy *string
	if db := strings.TrimSpace(e.DeniedBy); db != "" {
		deniedBy = &db
	}
	logID := e.ID
	if logID == uuid.Nil {
		logID = id.MustNew()
	}
	return domain.Entry{
		ID:                 logID,
		CreatedAt:          at.UTC(),
		SourceAddress:      e.SourceAddress,
		DestinationAddress: e.DestinationAddress,
		UserName:           user,
		DecideDurationUs:   e.SpendTime.Microseconds(),
		FullURL:            e.FullURLRequest,
		Action:             int16(e.Action),
		InspectRuleID:      parseOptionalUUID(e.InspectRuleRef),
		DeniedBy:           deniedBy,
		DecisionRuleRef:    decisionRuleRefFromEntry(e),
		Extra:              buildExtraJSON(e),
	}
}

func decisionRuleRefFromEntry(e accesslog.Entry) string {
	if s := strings.TrimSpace(e.RuleRef); s != "" {
		return s
	}
	return strings.TrimSpace(e.ACLRuleRef)
}

func parseOptionalUUID(raw string) *uuid.UUID {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "system_") {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
