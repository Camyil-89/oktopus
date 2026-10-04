package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	chdb "oktopus/internal/db/clickhouse"
	hostsettingspostgres "oktopus/internal/db/hostsettings/repository/postgres"
	hostsettingsservice "oktopus/internal/db/hostsettings/service"
	"oktopus/internal/db/infrastructure/postgres/store"
	proxyaccesslogch "oktopus/internal/db/proxyaccesslog/repository/clickhouse"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
	proxyaclpostgres "oktopus/internal/db/proxyacl/repository/postgres"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxyinspectpostgres "oktopus/internal/db/proxyinspect/repository/postgres"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"
	proxyinstancespostgres "oktopus/internal/db/proxyinstances/repository/postgres"
	proxyinstancesservice "oktopus/internal/db/proxyinstances/service"
	userpostgres "oktopus/internal/db/user/repository/postgres"
	userservice "oktopus/internal/db/user/service"
)

const defaultMigrationsDir = "db/migrations"

// Runtime подключение к БД и доменные сервисы.
type Runtime struct {
	Pool           *pgxpool.Pool
	ClickHouse     chdb.Closer
	Users          *userservice.Service
	HostSettings   *hostsettingsservice.Service
	ProxyInstances *proxyinstancesservice.Service
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
	hostRepo := hostsettingspostgres.New(q)
	hostSvc := hostsettingsservice.New(hostRepo)
	instanceRepo := proxyinstancespostgres.New(q)
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
	accessLogSvc := proxyaccesslogservice.New(accessLogRepo, hostRepo, nil)
	instanceSvc := proxyinstancesservice.New(instanceRepo, aclSvc, nil)
	instanceSvc.BindInspect(inspectSvc)

	if err := userSvc.EnsureAdmin(ctx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		pool.Close()
		return nil, err
	}
	if err := hostSvc.EnsureDefaults(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := aclSvc.BootstrapCompile(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("acl bootstrap: %w", err)
	}
	instances, err := instanceRepo.List(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}
	for _, inst := range instances {
		if err := inspectSvc.BootstrapInstance(ctx, inst.ID); err != nil {
			pool.Close()
			return nil, fmt.Errorf("inspect bootstrap instance %s: %w", inst.ID, err)
		}
	}
	aclSvc.StartListPoller()

	return &Runtime{
		Pool:           pool,
		ClickHouse:     chConn,
		Users:          userSvc,
		HostSettings:   hostSvc,
		ProxyInstances: instanceSvc,
		ProxyACL:       aclSvc,
		ProxyInspect:   inspectSvc,
		ProxyAccessLog: accessLogSvc,
	}, nil
}
