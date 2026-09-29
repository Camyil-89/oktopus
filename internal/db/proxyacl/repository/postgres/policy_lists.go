package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/proxyacl/domain"
)

func (r *RulesRepository) GetPolicy(ctx context.Context) (domain.Policy, error) {
	row, err := r.q.GetProxyACLPolicy(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Policy{}, nil
		}
		return domain.Policy{}, fmt.Errorf("postgres get proxy acl policy: %w", err)
	}
	return domain.Policy{ConfigText: row.ConfigText, UpdatedAt: row.UpdatedAt.Time}, nil
}

func (r *RulesRepository) SetPolicy(ctx context.Context, configText string) (domain.Policy, error) {
	row, err := r.q.UpsertProxyACLPolicy(ctx, configText)
	if err != nil {
		return domain.Policy{}, fmt.Errorf("postgres set proxy acl policy: %w", err)
	}
	return domain.Policy{ConfigText: row.ConfigText, UpdatedAt: row.UpdatedAt.Time}, nil
}

func (r *RulesRepository) ListNamedListsByNames(ctx context.Context, names []string) ([]domain.NamedList, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListProxyACLListsByNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy acl lists by names: %w", err)
	}
	out := make([]domain.NamedList, len(rows))
	for i, row := range rows {
		out[i] = toDomainList(row)
	}
	return out, nil
}

func (r *RulesRepository) GetNamedList(ctx context.Context, id uuid.UUID) (domain.NamedList, error) {
	row, err := r.q.GetProxyACLList(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NamedList{}, domain.ErrListNotFound
		}
		return domain.NamedList{}, fmt.Errorf("postgres get proxy acl list: %w", err)
	}
	return toDomainList(row), nil
}

func (r *RulesRepository) ListNamedListsSummary(ctx context.Context) ([]domain.NamedListSummary, error) {
	rows, err := r.q.ListProxyACLListsSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy acl lists summary: %w", err)
	}
	out := make([]domain.NamedListSummary, len(rows))
	for i, row := range rows {
		out[i] = domain.NamedListSummary{
			ID:                  row.ID,
			Name:                row.Name,
			ListType:            row.ListType,
			SourceMode:          row.SourceMode,
			SourceURL:           row.SourceUrl,
			PollIntervalMinutes: int(row.PollIntervalMinutes),
			BodyLineCount:       int(row.BodyLineCount),
			BodyPreview:         row.BodyPreview,
			CreatedAt:           row.CreatedAt.Time,
			UpdatedAt:           row.UpdatedAt.Time,
		}
	}
	return out, nil
}

func (r *RulesRepository) ListNamedLists(ctx context.Context) ([]domain.NamedList, error) {
	rows, err := r.q.ListProxyACLLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy acl lists: %w", err)
	}
	out := make([]domain.NamedList, len(rows))
	for i, row := range rows {
		out[i] = toDomainList(row)
	}
	return out, nil
}

func (r *RulesRepository) ReplaceNamedLists(ctx context.Context, lists []domain.NamedList) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	keepIDs := make([]uuid.UUID, 0, len(lists))
	for _, l := range lists {
		row, err := q.UpsertProxyACLList(ctx, store.UpsertProxyACLListParams{
			ID:                  l.ID,
			Name:                l.Name,
			ListType:            l.ListType,
			Body:                l.Body,
			SourceMode:          l.SourceMode,
			SourceUrl:           l.SourceURL,
			PollIntervalMinutes: int32(l.PollIntervalMinutes),
		})
		if err != nil {
			return fmt.Errorf("upsert list %s: %w", l.Name, err)
		}
		keepIDs = append(keepIDs, row.ID)
	}
	if err := q.DeleteProxyACLListsExcept(ctx, keepIDs); err != nil {
		return fmt.Errorf("delete lists except: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *RulesRepository) UpdateNamedListBody(ctx context.Context, id uuid.UUID, body string) (domain.NamedList, error) {
	row, err := r.q.UpdateProxyACLListBody(ctx, store.UpdateProxyACLListBodyParams{
		ID:   id,
		Body: body,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NamedList{}, domain.ErrListNotFound
		}
		return domain.NamedList{}, fmt.Errorf("postgres update proxy acl list body: %w", err)
	}
	return toDomainList(row), nil
}

func toDomainList(row store.ProxyAclList) domain.NamedList {
	return domain.NamedList{
		ID:                  row.ID,
		Name:                row.Name,
		ListType:            row.ListType,
		Body:                row.Body,
		SourceMode:          row.SourceMode,
		SourceURL:           row.SourceUrl,
		PollIntervalMinutes: int(row.PollIntervalMinutes),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}
