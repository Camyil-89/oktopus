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

func (r *RulesRepository) List(ctx context.Context) ([]domain.Rule, error) {
	rows, err := r.q.ListProxyInspectRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy inspect rules: %w", err)
	}
	out := make([]domain.Rule, len(rows))
	for i := range rows {
		out[i] = toDomain(rows[i])
	}
	return out, nil
}

func (r *RulesRepository) ListSummary(ctx context.Context) ([]domain.RuleSummary, error) {
	rows, err := r.q.ListProxyInspectRulesSummary(ctx)
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

func (r *RulesRepository) Count(ctx context.Context) (int64, error) {
	n, err := r.q.CountProxyInspectRules(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres count proxy inspect rules: %w", err)
	}
	return n, nil
}

func (r *RulesRepository) ReplaceAll(ctx context.Context, rules []domain.Rule) error {
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
	if err := q.DeleteProxyInspectRulesExcept(ctx, keepIDs); err != nil {
		return fmt.Errorf("delete removed rules: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func upsertParams(r domain.Rule) store.UpsertProxyInspectRuleParams {
	return store.UpsertProxyInspectRuleParams{
		ID:        r.ID,
		Name:      r.Name,
		Script:    r.Script,
		Action:    r.Action,
		Enabled:   r.Enabled,
		SortOrder: int32(r.SortOrder),
	}
}

func toDomain(row store.ProxyInspectRule) domain.Rule {
	return domain.Rule{
		ID:        row.ID,
		Name:      row.Name,
		Script:    row.Script,
		Action:    row.Action,
		Enabled:   row.Enabled,
		SortOrder: int(row.SortOrder),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func toDomainSummary(row store.ListProxyInspectRulesSummaryRow) domain.RuleSummary {
	return domain.RuleSummary{
		ID:           row.ID,
		Name:         row.Name,
		Action:       row.Action,
		Enabled:      row.Enabled,
		SortOrder:    int(row.SortOrder),
		ScriptLength: int(row.ScriptLength),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
