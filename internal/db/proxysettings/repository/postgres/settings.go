package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/proxysettings/domain"
	"oktopus/internal/db/proxysettings/repository"
)

type SettingsRepository struct {
	q *store.Queries
}

func NewSettingsRepository(q *store.Queries) repository.SettingsRepository {
	return &SettingsRepository{q: q}
}

func (r *SettingsRepository) Get(ctx context.Context) (domain.Settings, error) {
	row, err := r.q.GetProxySettings(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Settings{}, domain.ErrNotFound
		}
		return domain.Settings{}, fmt.Errorf("postgres get proxy settings: %w", err)
	}
	return toDomain(row), nil
}

func (r *SettingsRepository) Insert(ctx context.Context, s domain.Settings) (domain.Settings, error) {
	s = s.NormalizeCAPaths()
	row, err := r.q.InsertProxySettings(ctx, insertParams(s))
	if err != nil {
		return domain.Settings{}, fmt.Errorf("postgres insert proxy settings: %w", err)
	}
	return toDomain(row), nil
}

func (r *SettingsRepository) Update(ctx context.Context, s domain.Settings) (domain.Settings, error) {
	s = s.NormalizeCAPaths()
	row, err := r.q.UpdateProxySettings(ctx, updateParams(s))
	if err != nil {
		return domain.Settings{}, fmt.Errorf("postgres update proxy settings: %w", err)
	}
	return toDomain(row), nil
}

func (r *SettingsRepository) UpdateCAPaths(ctx context.Context, id uuid.UUID, certPath, keyPath string) (domain.Settings, error) {
	row, err := r.q.UpdateProxyCAPaths(ctx, store.UpdateProxyCAPathsParams{
		ID:         id,
		CaCertPath: domain.DefaultCACertPath,
		CaKeyPath:  domain.DefaultCAKeyPath,
	})
	if err != nil {
		return domain.Settings{}, fmt.Errorf("postgres update proxy ca paths: %w", err)
	}
	return toDomain(row), nil
}

func insertParams(s domain.Settings) store.InsertProxySettingsParams {
	return store.InsertProxySettingsParams{
		ID:                    s.ID,
		ProxyEnabled:          s.ProxyEnabled,
		Listen:                s.Listen,
		ConnectMode:           s.ConnectMode,
		CaCertPath:            s.CACertPath,
		CaKeyPath:             s.CAKeyPath,
		AuthEnabled:           s.AuthEnabled,
		AuthStaticUsers:       s.AuthStaticUsers,
		AuthRealm:             s.AuthRealm,
		AuthBackend:           s.AuthBackend,
		AuthCacheTtlMinutes:   s.AuthCacheTTLMinutes,
		LdapUrl:               s.LDAPURL,
		LdapBaseDn:            s.LDAPBaseDN,
		LdapBindDn:            s.LDAPBindDN,
		LdapBindPassword:         s.LDAPBindPassword,
		AccessLogRetentionDays:   s.AccessLogRetentionDays,
	}
}

func updateParams(s domain.Settings) store.UpdateProxySettingsParams {
	return store.UpdateProxySettingsParams{
		ID:                    s.ID,
		ProxyEnabled:          s.ProxyEnabled,
		Listen:                s.Listen,
		ConnectMode:           s.ConnectMode,
		CaCertPath:            s.CACertPath,
		CaKeyPath:             s.CAKeyPath,
		AuthEnabled:           s.AuthEnabled,
		AuthStaticUsers:       s.AuthStaticUsers,
		AuthRealm:             s.AuthRealm,
		AuthBackend:           s.AuthBackend,
		AuthCacheTtlMinutes:   s.AuthCacheTTLMinutes,
		LdapUrl:               s.LDAPURL,
		LdapBaseDn:            s.LDAPBaseDN,
		LdapBindDn:            s.LDAPBindDN,
		LdapBindPassword:         s.LDAPBindPassword,
		AccessLogRetentionDays:   s.AccessLogRetentionDays,
	}
}

func toDomain(row store.ProxySetting) domain.Settings {
	return domain.Settings{
		ID:                  row.ID,
		ProxyEnabled:        row.ProxyEnabled,
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
		LDAPBindPassword:       row.LdapBindPassword,
		AccessLogRetentionDays: row.AccessLogRetentionDays,
		CreatedAt:              row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}
