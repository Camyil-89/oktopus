package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/proxyinstances/domain"
	"oktopus/internal/db/proxyinstances/repository"
)

type InstanceRepository struct {
	q *store.Queries
}

func New(q *store.Queries) repository.Repository {
	return &InstanceRepository{q: q}
}

func (r *InstanceRepository) List(ctx context.Context) ([]domain.Instance, error) {
	rows, err := r.q.ListProxyInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres list proxy instances: %w", err)
	}
	out := make([]domain.Instance, len(rows))
	for i := range rows {
		out[i] = toDomain(rows[i])
	}
	return out, nil
}

func (r *InstanceRepository) Get(ctx context.Context, id uuid.UUID) (domain.Instance, error) {
	row, err := r.q.GetProxyInstance(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Instance{}, domain.ErrNotFound
		}
		return domain.Instance{}, fmt.Errorf("postgres get proxy instance: %w", err)
	}
	return toDomain(row), nil
}

func (r *InstanceRepository) Insert(ctx context.Context, inst domain.Instance) (domain.Instance, error) {
	row, err := r.q.InsertProxyInstance(ctx, insertParams(inst))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Instance{}, mapUnique(err)
		}
		return domain.Instance{}, fmt.Errorf("postgres insert proxy instance: %w", err)
	}
	return toDomain(row), nil
}

func (r *InstanceRepository) Update(ctx context.Context, inst domain.Instance) (domain.Instance, error) {
	row, err := r.q.UpdateProxyInstance(ctx, updateParams(inst))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Instance{}, domain.ErrNotFound
		}
		if isUniqueViolation(err) {
			return domain.Instance{}, mapUnique(err)
		}
		return domain.Instance{}, fmt.Errorf("postgres update proxy instance: %w", err)
	}
	return toDomain(row), nil
}

func (r *InstanceRepository) UpdateCAPaths(ctx context.Context, id uuid.UUID, certPath, keyPath string) (domain.Instance, error) {
	row, err := r.q.UpdateProxyInstanceCAPaths(ctx, store.UpdateProxyInstanceCAPathsParams{
		ID:         id,
		CaCertPath: certPath,
		CaKeyPath:  keyPath,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Instance{}, domain.ErrNotFound
		}
		return domain.Instance{}, fmt.Errorf("postgres update instance ca paths: %w", err)
	}
	return toDomain(row), nil
}

func (r *InstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteProxyInstance(ctx, id); err != nil {
		return fmt.Errorf("postgres delete proxy instance: %w", err)
	}
	return nil
}

func (r *InstanceRepository) InsertACLPolicy(ctx context.Context, policyID, instanceID uuid.UUID) error {
	_, err := r.q.InsertProxyACLPolicyForInstance(ctx, store.InsertProxyACLPolicyForInstanceParams{
		ID:         policyID,
		InstanceID: instanceID,
	})
	if err != nil {
		return fmt.Errorf("postgres insert acl policy for instance: %w", err)
	}
	return nil
}

func insertParams(inst domain.Instance) store.InsertProxyInstanceParams {
	return store.InsertProxyInstanceParams{
		ID:                  inst.ID,
		Name:                inst.Name,
		Enabled:             inst.Enabled,
		Listen:              inst.Listen,
		ConnectMode:         inst.ConnectMode,
		CaCertPath:          inst.CACertPath,
		CaKeyPath:           inst.CAKeyPath,
		AuthEnabled:         inst.AuthEnabled,
		AuthStaticUsers:     inst.AuthStaticUsers,
		AuthRealm:           inst.AuthRealm,
		AuthBackend:         inst.AuthBackend,
		AuthCacheTtlMinutes: inst.AuthCacheTTLMinutes,
		LdapUrl:             inst.LDAPURL,
		LdapBaseDn:          inst.LDAPBaseDN,
		LdapBindDn:          inst.LDAPBindDN,
		LdapBindPassword:    inst.LDAPBindPassword,
		SortOrder:           int32(inst.SortOrder),
	}
}

func updateParams(inst domain.Instance) store.UpdateProxyInstanceParams {
	return store.UpdateProxyInstanceParams{
		ID:                  inst.ID,
		Name:                inst.Name,
		Enabled:             inst.Enabled,
		Listen:              inst.Listen,
		ConnectMode:         inst.ConnectMode,
		CaCertPath:          inst.CACertPath,
		CaKeyPath:           inst.CAKeyPath,
		AuthEnabled:         inst.AuthEnabled,
		AuthStaticUsers:     inst.AuthStaticUsers,
		AuthRealm:           inst.AuthRealm,
		AuthBackend:         inst.AuthBackend,
		AuthCacheTtlMinutes: inst.AuthCacheTTLMinutes,
		LdapUrl:             inst.LDAPURL,
		LdapBaseDn:          inst.LDAPBaseDN,
		LdapBindDn:          inst.LDAPBindDN,
		LdapBindPassword:    inst.LDAPBindPassword,
		SortOrder:           int32(inst.SortOrder),
	}
}

func toDomain(row store.ProxyInstance) domain.Instance {
	return domain.Instance{
		ID:                  row.ID,
		Name:                row.Name,
		Enabled:             row.Enabled,
		Listen:              row.Listen,
		ConnectMode:         row.ConnectMode,
		CACertPath:          row.CaCertPath,
		CAKeyPath:           row.CaKeyPath,
		AuthEnabled:         row.AuthEnabled,
		AuthStaticUsers:     row.AuthStaticUsers,
		AuthRealm:           row.AuthRealm,
		AuthBackend:         row.AuthBackend,
		AuthCacheTTLMinutes: row.AuthCacheTtlMinutes,
		LDAPURL:             row.LdapUrl,
		LDAPBaseDN:          row.LdapBaseDn,
		LDAPBindDN:          row.LdapBindDn,
		LDAPBindPassword:    row.LdapBindPassword,
		SortOrder:           int(row.SortOrder),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func mapUnique(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "proxy_instances_listen_unique":
			return domain.ErrListenTaken
		case "proxy_instances_name_unique":
			return domain.ErrNameTaken
		}
	}
	return err
}
