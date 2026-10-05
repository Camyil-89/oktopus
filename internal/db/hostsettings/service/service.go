package service

import (
	"context"
	"errors"

	"oktopus/internal/apperr"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
	"oktopus/internal/db/hostsettings/domain"
	"oktopus/internal/db/hostsettings/repository"
	"oktopus/internal/id"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureDefaults(ctx context.Context) error {
	_, err := s.repo.Get(ctx)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		_, err = s.repo.Insert(ctx, domain.Default(id.MustNew()))
		return err
	default:
		return err
	}
}

func (s *Service) Get(ctx context.Context) (domain.Settings, error) {
	return s.repo.Get(ctx)
}

type UpdateInput struct {
	AccessLogRetentionDays *int32
}

func (s *Service) Update(ctx context.Context, patch UpdateInput) (domain.Settings, error) {
	cur, err := s.repo.Get(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	if patch.AccessLogRetentionDays != nil {
		cur.AccessLogRetentionDays = *patch.AccessLogRetentionDays
	}
	if err := proxyaccesslogservice.ValidateRetentionDays(cur.AccessLogRetentionDays); err != nil {
		return domain.Settings{}, err
	}
	updated, err := s.repo.Update(ctx, cur)
	if err != nil {
		return domain.Settings{}, apperr.Internal(apperr.UpdateProxySettingsFailed, err)
	}
	return updated, nil
}

func (s *Service) ReportsDashboard(ctx context.Context) ([]byte, error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	return st.ReportsDashboard, nil
}

func (s *Service) UpdateReportsDashboard(ctx context.Context, body []byte) ([]byte, error) {
	cur, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if body == nil {
		body = domain.DefaultReportsDashboard
	}
	updated, err := s.repo.UpdateReportsDashboard(ctx, cur.ID, body)
	if err != nil {
		return nil, apperr.Internal(apperr.UpdateReportsDashboardFailed, err)
	}
	return updated.ReportsDashboard, nil
}
