package clickhouse

import (
	"strings"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/extra"
	"oktopus/internal/db/proxyaccesslog/repository"
	"oktopus/internal/proxy/observe/policyanomaly"
)

func listWhere(f repository.ListFilter) (string, []any) {
	parts := []string{"1 = 1"}
	args := []any{}

	addILIKE := func(col, val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		parts = append(parts, col+" ILIKE ?")
		args = append(args, "%"+val+"%")
	}

	addILIKE("user_name", f.User)
	addILIKE("source_address", f.Source)
	addILIKE("destination_address", f.Destination)
	addILIKE("full_url", f.URL)

	if f.SearchOnly {
		parts = append(parts, "search_engine != ''")
	}

	segment := strings.TrimSpace(f.Segment)
	if segment == "" && strings.TrimSpace(f.ErrorKind) == "any" {
		segment = repository.SegmentErrors
	}

	errPred := accessLogErrorPredicate()
	attackPred, attackArgs := policyAnomalyPredicate(f.AttackKind, f.PolicyAnomalyQ)

	switch segment {
	case repository.SegmentTraffic:
		parts = append(parts, "NOT ("+errPred+")")
		parts = append(parts, "NOT ("+attackPred+")")
		args = append(args, attackArgs...)
	case repository.SegmentAttacks:
		parts = append(parts, attackPred)
		args = append(args, attackArgs...)
	case repository.SegmentErrors:
		parts = append(parts, errPred)
	}

	if f.From != nil {
		parts = append(parts, "created_at >= ?")
		args = append(args, f.From.UTC())
	}
	if f.To != nil {
		parts = append(parts, "created_at <= ?")
		args = append(args, f.To.UTC())
	}
	if f.Action == 0 || f.Action == 1 {
		parts = append(parts, "action = ?")
		args = append(args, uint8(f.Action))
	}
	if inst := strings.TrimSpace(f.InstanceID); inst != "" {
		parsed, err := uuid.Parse(inst)
		if err == nil {
			parts = append(parts, "instance_id = ?")
			args = append(args, parsed)
		}
	}
	if id := strings.TrimSpace(f.ID); id != "" {
		parsed, err := uuid.Parse(id)
		if err == nil {
			parts = append(parts, "id = ?")
			args = append(args, parsed)
		}
	}
	if ref := strings.TrimSpace(f.DecisionRuleRef); ref != "" {
		parts = append(parts, "decision_rule_ref ILIKE ?")
		args = append(args, "%"+ref+"%")
	}
	if id := strings.TrimSpace(f.InspectRuleID); id != "" {
		parsed, err := uuid.Parse(id)
		if err == nil {
			parts = append(parts, "inspect_rule_id = ?")
			args = append(args, parsed)
		}
	}

	return strings.Join(parts, " AND "), args
}

func accessLogErrorPredicate() string {
	return `(
		length(trimBoth(inspect_error)) > 0
		OR decision_rule_ref IN ('system_auth_fail', 'system_inspect_error', 'system_gateway_error')
		OR coalesce(denied_by, '') = 'gateway'
	)`
}

func policyAnomalyPredicate(kind, contains string) (string, []any) {
	kind = strings.TrimSpace(kind)
	contains = strings.TrimSpace(contains)
	sub := []string{
		"inspect_rule_id = ?",
		"field_key = 'policy_anomaly'",
		"positionCaseInsensitive(field_value, ?) > 0",
	}
	args := []any{extra.PolicyAnomalyKVRuleID, policyanomaly.CHSearchDetectTrue}
	if kind != "" {
		sub = append(sub, "positionCaseInsensitive(field_value, ?) > 0")
		args = append(args, policyanomaly.CHSearchKindDetected(kind))
	}
	if contains != "" {
		sub = append(sub, "field_value ILIKE ?")
		args = append(args, "%"+contains+"%")
	}
	sql := "id IN (SELECT log_id FROM proxy_access_log_inspect_kv WHERE " + strings.Join(sub, " AND ") + ")"
	return sql, args
}
