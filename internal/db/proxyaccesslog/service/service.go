package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"oktopus/internal/db/proxyaccesslog/domain"
	"oktopus/internal/db/proxyaccesslog/repository"
	hostsettingsrepo "oktopus/internal/db/hostsettings/repository"
	"oktopus/internal/proxy/accesslog"
)

const (
	MinRetentionDays       = 1
	DefaultRetentionDays   = 3
	RetentionPurgeInterval = 15 * time.Minute
)

// RetentionUnlimited — значение access_log_retention_days в БД (0).
const RetentionUnlimited = 0

type Service struct {
	repo     repository.Repository
	settings hostsettingsrepo.Repository
	log      *log.Logger
}

func New(repo repository.Repository, settings hostsettingsrepo.Repository, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{repo: repo, settings: settings, log: logger}
}

func (s *Service) List(ctx context.Context, f repository.ListFilter, page, pageSize int) (repository.Page, error) {
	return s.repo.List(ctx, f, page, pageSize)
}

func (s *Service) DeleteBetween(ctx context.Context, from, to time.Time) (int64, error) {
	if to.Before(from) {
		return 0, fmt.Errorf("period end must not be before start")
	}
	return s.repo.DeleteBetween(ctx, from.UTC(), to.UTC())
}

func (s *Service) FlushAccessLog(ctx context.Context, batch []accesslog.Entry) error {
	if len(batch) == 0 {
		return nil
	}
	rows := make([]domain.Entry, 0, len(batch))
	for _, e := range batch {
		row, err := fromAccessEntry(e)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return s.repo.InsertBatch(ctx, rows)
}

func (s *Service) PurgeExpired(ctx context.Context) error {
	st, err := s.settings.Get(ctx)
	if err != nil {
		return err
	}
	days := st.AccessLogRetentionDays
	if days == RetentionUnlimited {
		return nil
	}
	if days < MinRetentionDays {
		days = MinRetentionDays
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	if err := s.repo.DeleteBefore(ctx, cutoff); err != nil {
		return err
	}
	return nil
}

// RunRetentionLoop раз в interval проверяет срок хранения и удаляет устаревшие записи.
func (s *Service) RunRetentionLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = RetentionPurgeInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.PurgeExpired(ctx); err != nil {
				s.log.Printf("proxy access log purge: %v", err)
			}
		}
	}
}

func ValidateRetentionDays(days int32) error {
	if days == RetentionUnlimited {
		return nil
	}
	if days < MinRetentionDays {
		return fmt.Errorf("access_log_retention_days must be >= %d or 0 (unlimited)", MinRetentionDays)
	}
	return nil
}
