package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/proxyinspect/domain"
	"oktopus/internal/db/proxyinspect/repository"
)

type RulesRepository struct {
	pool *pgxpool.Pool
	q    *store.Queries
}

func NewRulesRepository(pool *pgxpool.Pool, q *store.Queries) repository.RulesRepository {
	return &RulesRepository{pool: pool, q: q}
}

func (r *RulesRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]domain.Rule, error) {
	rows, err := r.q.ListProxyInspectRulesByInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy inspect rules: %w", err)
	}
	out := make([]domain.Rule, len(rows))
	for i := range rows {
		out[i] = toDomain(rows[i])
	}
	return out, nil
}

func (r *RulesRepository) ListSummaryByInstance(ctx context.Context, instanceID uuid.UUID) ([]domain.RuleSummary, error) {
	rows, err := r.q.ListProxyInspectRulesSummaryByInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy inspect rules summary: %w", err)
	}
	out := make([]domain.RuleSummary, len(rows))
	for i := range rows {
		out[i] = toDomainSummary(rows[i])
	}
	return out, nil
}

func (r *RulesRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Rule, error) {
	row, err := r.q.GetProxyInspectRule(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rule{}, domain.ErrNotFound
		}
		return domain.Rule{}, fmt.Errorf("postgres get proxy inspect rule: %w", err)
	}
	return toDomain(row), nil
}

func (r *RulesRepository) CountByInstance(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	n, err := r.q.CountProxyInspectRulesByInstance(ctx, instanceID)
	if err != nil {
		return 0, fmt.Errorf("postgres count proxy inspect rules: %w", err)
	}
	return n, nil
}

func (r *RulesRepository) ListInstanceIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.q.ListProxyInspectRuleInstanceIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres list inspect instance ids: %w", err)
	}
	return rows, nil
}

func (r *RulesRepository) ReplaceAllForInstance(ctx context.Context, instanceID uuid.UUID, rules []domain.Rule) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	keepIDs := make([]uuid.UUID, 0, len(rules))
	for _, rule := range rules {
		if _, err := q.UpsertProxyInspectRule(ctx, upsertParams(rule)); err != nil {
			return fmt.Errorf("upsert rule %s: %w", rule.ID, err)
		}
		keepIDs = append(keepIDs, rule.ID)
	}
	if err := q.DeleteProxyInspectRulesByInstanceExcept(ctx, store.DeleteProxyInspectRulesByInstanceExceptParams{
		InstanceID: instanceID,
		KeepIds:    keepIDs,
	}); err != nil {
		return fmt.Errorf("delete removed rules: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func upsertParams(r domain.Rule) store.UpsertProxyInspectRuleParams {
	return store.UpsertProxyInspectRuleParams{
		ID:         r.ID,
		InstanceID: r.InstanceID,
		Name:       r.Name,
		Script:     r.Script,
		Action:     r.Action,
		Enabled:    r.Enabled,
		SortOrder:  int32(r.SortOrder),
	}
}

func toDomain(row store.ProxyInspectRule) domain.Rule {
	return domain.Rule{
		ID:         row.ID,
		InstanceID: row.InstanceID,
		Name:       row.Name,
		Script:     row.Script,
		Action:     row.Action,
		Enabled:    row.Enabled,
		SortOrder:  int(row.SortOrder),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func toDomainSummary(row store.ListProxyInspectRulesSummaryByInstanceRow) domain.RuleSummary {
	return domain.RuleSummary{
		ID:           row.ID,
		InstanceID:   row.InstanceID,
		Name:         row.Name,
		Action:       row.Action,
		Enabled:      row.Enabled,
		SortOrder:    int(row.SortOrder),
		ScriptLength: int(row.ScriptLength),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
