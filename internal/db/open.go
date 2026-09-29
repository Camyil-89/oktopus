package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	chdb "oktopus/internal/db/clickhouse"
	"oktopus/internal/db/infrastructure/postgres/store"
	proxyaccesslogch "oktopus/internal/db/proxyaccesslog/repository/clickhouse"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
	proxyaclpostgres "oktopus/internal/db/proxyacl/repository/postgres"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxyinspectpostgres "oktopus/internal/db/proxyinspect/repository/postgres"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"
	proxysettingspostgres "oktopus/internal/db/proxysettings/repository/postgres"
	proxysettingsservice "oktopus/internal/db/proxysettings/service"
	userpostgres "oktopus/internal/db/user/repository/postgres"
	userservice "oktopus/internal/db/user/service"
)

const defaultMigrationsDir = "db/migrations"

// Runtime подключение к БД и доменные сервисы.
type Runtime struct {
	Pool          *pgxpool.Pool
	ClickHouse    chdb.Closer
	Users         *userservice.Service
	ProxySettings *proxysettingsservice.Service
	ProxyACL       *proxyaclservice.Service
	ProxyInspect   *proxyinspectservice.Service
	ProxyAccessLog *proxyaccesslogservice.Service
}

// Open подключается к PostgreSQL, применяет миграции и гарантирует admin-пользователя.
func Open(ctx context.Context, cfg Config, migrationsDir string) (*Runtime, error) {
	if migrationsDir == "" {
		migrationsDir = defaultMigrationsDir
	}

	if err := RunMigrations(cfg.DatabaseURL, migrationsDir); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}

	q := store.New(pool)
	userSvc := userservice.New(userpostgres.NewUserRepository(q), cfg.AdminUsername)
	proxyRepo := proxysettingspostgres.NewSettingsRepository(q)
	aclRepo := proxyaclpostgres.NewRulesRepository(pool, q)
	aclSvc := proxyaclservice.New(aclRepo)
	inspectRepo := proxyinspectpostgres.NewRulesRepository(pool, q)
	inspectSvc := proxyinspectservice.New(inspectRepo)
	chConn, err := chdb.Open(ctx, chdb.Config{
		Addr:     cfg.ClickHouse.Addr,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}
	accessLogRepo := proxyaccesslogch.New(chConn)
	accessLogSvc := proxyaccesslogservice.New(accessLogRepo, proxyRepo, nil)
	proxySvc := proxysettingsservice.New(proxyRepo, aclSvc, nil)
	proxySvc.BindInspect(inspectSvc)

	if err := userSvc.EnsureAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		pool.Close()
		return nil, err
	}
	if err := proxySvc.EnsureDefaults(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := aclSvc.BootstrapCompile(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("acl bootstrap: %w", err)
	}
	if err := inspectSvc.BootstrapCompile(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("inspect bootstrap: %w", err)
	}
	aclSvc.StartListPoller()

	return &Runtime{
		Pool:           pool,
		ClickHouse:     chConn,
		Users:          userSvc,
		ProxySettings:  proxySvc,
		ProxyACL:       aclSvc,
		ProxyInspect:   inspectSvc,
		ProxyAccessLog: accessLogSvc,
	}, nil
}
