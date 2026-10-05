package repository

import (
	"context"
	"time"

	"oktopus/internal/db/proxyaccesslog/domain"
)

type ListFilter struct {
	InstanceID      string // UUID инстанса прокси (пусто — все)
	ID              string // UUID записи журнала
	User            string
	Source          string
	Destination     string
	URL             string
	SearchOnly      bool
	// ErrorKind: "" — не фильтровать; any — только ошибки (legacy, см. Segment).
	ErrorKind string
	// Segment: traffic | attacks | errors (пусто — без сегмента, кроме legacy error_kind).
	Segment string
	// AttackKind — подстрока kind в policy_anomaly (например host_sni_mismatch).
	AttackKind string
	// PolicyAnomalyQ — ILIKE по JSON policy_anomaly в KV.
	PolicyAnomalyQ string
	From            *time.Time
	To              *time.Time
	Action          int32 // -1 = любое, 0 deny, 1 allow
	DecisionRuleRef string
	InspectRuleID   string
}

type Page struct {
	Items []domain.Entry
	Total int64
	Page  int
	Size  int
}

type Repository interface {
	InsertBatch(ctx context.Context, entries []domain.Entry) error
	List(ctx context.Context, f ListFilter, page, pageSize int) (Page, error)
	DeleteBefore(ctx context.Context, before time.Time) error
	DeleteBetween(ctx context.Context, from, to time.Time) (int64, error)
	RunWidgetQuery(
		ctx context.Context,
		from, to time.Time,
		f ReportFilters,
		q WidgetQuery,
		widgetType string,
	) (WidgetAnalyticsResult, error)
}
