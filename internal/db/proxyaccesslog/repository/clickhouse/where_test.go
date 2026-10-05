package clickhouse_test

import (
	"strings"
	"testing"

	"oktopus/internal/db/proxyaccesslog/extra"
	"oktopus/internal/db/proxyaccesslog/repository"
	"oktopus/internal/db/proxyaccesslog/repository/clickhouse"
	"oktopus/internal/proxy/observe"
)

func TestListWhereSegmentTrafficExcludesAttackAndError(t *testing.T) {
	where, _ := clickhouse.ListWhereForTest(repository.ListFilter{Segment: repository.SegmentTraffic})
	if !strings.Contains(where, "NOT") {
		t.Fatalf("where: %s", where)
	}
	if !strings.Contains(where, "policy_anomaly") {
		t.Fatalf("expected attack subquery: %s", where)
	}
}

func TestListWhereSegmentAttacksWithKind(t *testing.T) {
	where, args := clickhouse.ListWhereForTest(repository.ListFilter{
		Segment:    repository.SegmentAttacks,
		AttackKind: observe.PolicyAnomalyHostSNIMismatch,
	})
	if !strings.Contains(where, "positionCaseInsensitive") {
		t.Fatalf("where: %s", where)
	}
	if len(args) < 1 || args[0] != extra.PolicyAnomalyKVRuleID {
		t.Fatalf("args: %v", args)
	}
}
