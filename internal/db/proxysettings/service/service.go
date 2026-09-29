package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"oktopus/internal/db/proxysettings/domain"
	"oktopus/internal/db/proxysettings/repository"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
	"oktopus/internal/id"
	"oktopus/internal/pki"
	proxyconfig "oktopus/internal/proxy/config"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/inspect"
)

const defaultCADir = "config"

// Applier применяет конфиг к работающему прокси (hot reload).
type Applier interface {
	Apply(ctx context.Context, cfg proxyconfig.Config, aclEngine *acl.Engine, inspectRunner *inspect.Runner) error
	Stop()
}

// CAStatusDTO состояние CA для API.
type CAStatusDTO struct {
	CertInstalled bool
	KeyInstalled  bool
	ValidUntil    *time.Time
}

// Service сценарии настроек прокси.
type Service struct {
	repo    repository.SettingsRepository
	acl     *proxyaclservice.Service
	inspect *proxyinspectservice.Service
	applier Applier
}

func New(repo repository.SettingsRepository, acl *proxyaclservice.Service, applier Applier) *Service {
	return &Service{repo: repo, acl: acl, applier: applier}
}

func (s *Service) BindInspect(ins *proxyinspectservice.Service) {
	s.inspect = ins
}

func (s *Service) BindApplier(a Applier) {
	s.applier = a
}

func (s *Service) EnsureDefaults(ctx context.Context) error {
	_, err := s.repo.Get(ctx)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		def := domain.DefaultSettings(id.MustNew())
		_, err = s.repo.Insert(ctx, def)
		return err
	default:
		return err
	}
}

func (s *Service) Get(ctx context.Context) (domain.Settings, error) {
	return s.repo.Get(ctx)
}

type UpdateInput struct {
	ProxyEnabled         *bool
	Listen               *string
	ConnectMode          *string
	AuthEnabled          *bool
	AuthStaticUsers      *string
	AuthRealm            *string
	AuthBackend          *string
	AuthCacheTTLMinutes  *int64
	LDAPURL              *string
	LDAPBaseDN           *string
	LDAPBindDN           *string
	LDAPBindPassword           *string
	AccessLogRetentionDays     *int32
}

func (s *Service) Update(ctx context.Context, patch UpdateInput) (domain.Settings, error) {
	cur, err := s.repo.Get(ctx)
	if err != nil {
		return domain.Settings{}, err
	}

	next := applyPatch(cur, patch).NormalizeCAPaths()
	if err := validateSettings(next); err != nil {
		return domain.Settings{}, err
	}

	updated, err := s.repo.Update(ctx, next)
	if err != nil {
		return domain.Settings{}, err
	}
	return s.applySettings(ctx, updated)
}

type GenerateCAInput struct {
	KeyBits      int
	CommonName   string
	Organization string
	Country      string
	ValidDays    int
}

func (s *Service) GenerateCA(ctx context.Context, in GenerateCAInput) (domain.Settings, error) {
	cur, err := s.repo.Get(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	if in.KeyBits == 0 {
		in.KeyBits = 4096
	}
	if in.KeyBits < 2048 {
		return domain.Settings{}, fmt.Errorf("key_bits must be >= 2048")
	}
	if in.CommonName == "" {
		in.CommonName = "Oktopus Proxy CA"
	}
	if in.Organization == "" {
		in.Organization = "Oktopus"
	}
	if in.Country == "" {
		in.Country = "RU"
	}
	if in.ValidDays <= 0 {
		in.ValidDays = 3650
	}

	if err := pki.GenerateCA(pki.CAConfig{
		CommonName:   in.CommonName,
		Organization: in.Organization,
		Country:      in.Country,
		ValidDays:    in.ValidDays,
		KeyBits:      in.KeyBits,
		OutDir:       defaultCADir,
		Force:        true,
	}); err != nil {
		return domain.Settings{}, err
	}

	updated, err := s.repo.UpdateCAPaths(ctx, cur.ID, domain.DefaultCACertPath, domain.DefaultCAKeyPath)
	if err != nil {
		return domain.Settings{}, err
	}
	return s.applySettings(ctx, updated)
}

func (s *Service) UploadCA(ctx context.Context, certReader, keyReader io.Reader) (domain.Settings, error) {
	cur, err := s.repo.Get(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	certPEM, err := io.ReadAll(certReader)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := io.ReadAll(keyReader)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("read key: %w", err)
	}

	if _, _, err := pki.SaveCAFiles(defaultCADir, certPEM, keyPEM, true); err != nil {
		return domain.Settings{}, err
	}

	updated, err := s.repo.UpdateCAPaths(ctx, cur.ID, domain.DefaultCACertPath, domain.DefaultCAKeyPath)
	if err != nil {
		return domain.Settings{}, err
	}
	return s.applySettings(ctx, updated)
}

func (s *Service) CAStatus(ctx context.Context) (CAStatusDTO, error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return CAStatusDTO{}, err
	}
	info, err := pki.InspectCAFiles(st.CACertPath, st.CAKeyPath)
	if err != nil {
		return CAStatusDTO{}, err
	}
	dto := CAStatusDTO{
		CertInstalled: info.CertExists,
		KeyInstalled:  info.KeyExists,
	}
	if info.HasCert {
		t := info.NotAfter.UTC()
		dto.ValidUntil = &t
	}
	return dto, nil
}

func (s *Service) CACertPath(ctx context.Context) (string, error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(st.CACertPath); err != nil {
		return "", fmt.Errorf("cert not found")
	}
	return st.CACertPath, nil
}

func (s *Service) CAKeyPath(ctx context.Context) (string, error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(st.CAKeyPath); err != nil {
		return "", fmt.Errorf("key not found")
	}
	return st.CAKeyPath, nil
}

// Runtime — конфиг прокси и скомпилированный ACL из БД.
type Runtime struct {
	Config          proxyconfig.Config
	ACLEngine       *acl.Engine
	InspectRunner   *inspect.Runner
}

func (s *Service) LoadProxyRuntime(ctx context.Context) (Runtime, error) {
	st, err := s.repo.Get(ctx)
	if err != nil {
		return Runtime{}, err
	}
	var engine *acl.Engine
	if s.acl != nil {
		engine = s.acl.ActiveEngine()
	} else {
		engine = acl.EmptyEngine()
	}
	var inspectRunner *inspect.Runner
	if s.inspect != nil {
		inspectRunner = s.inspect.ActiveRunner()
	} else {
		inspectRunner = inspect.NewRunner(&inspect.Program{})
	}
	return Runtime{
		Config:        ToProxyConfig(st),
		ACLEngine:     engine,
		InspectRunner: inspectRunner,
	}, nil
}

func (s *Service) LoadProxyConfig(ctx context.Context) (proxyconfig.Config, error) {
	rt, err := s.LoadProxyRuntime(ctx)
	if err != nil {
		return proxyconfig.Config{}, err
	}
	return rt.Config, nil
}

func (s *Service) applySettings(ctx context.Context, st domain.Settings) (domain.Settings, error) {
	if s.applier == nil {
		return st, nil
	}
	if !st.ProxyEnabled {
		s.applier.Stop()
		return st, nil
	}
	rt, err := s.LoadProxyRuntime(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	if err := s.applier.Apply(ctx, rt.Config, rt.ACLEngine, rt.InspectRunner); err != nil {
		return domain.Settings{}, fmt.Errorf("apply proxy config: %w", err)
	}
	return st, nil
}

func ToProxyConfig(st domain.Settings) proxyconfig.Config {
	st = st.NormalizeCAPaths()
	mode := proxyconfig.ConnectMode(strings.ToLower(strings.TrimSpace(st.ConnectMode)))

	var cacheTTL time.Duration
	if st.AuthCacheTTLMinutes > 0 {
		cacheTTL = time.Duration(st.AuthCacheTTLMinutes) * time.Minute
	}

	return proxyconfig.Config{
		Listen:  st.Listen,
		Connect: mode,
		Auth: proxyconfig.AuthConfig{
			Enabled:     st.AuthEnabled,
			Realm:       st.AuthRealm,
			Backend:     st.AuthBackend,
			StaticUsers: st.AuthStaticUsers,
			CacheTTL:    cacheTTL,
			LDAP: proxyconfig.LDAPConfig{
				URL:          st.LDAPURL,
				BaseDN:       st.LDAPBaseDN,
				BindDN:       st.LDAPBindDN,
				BindPassword: st.LDAPBindPassword,
			},
		},
		CACertPath: st.CACertPath,
		CAKeyPath:  st.CAKeyPath,
	}.WithDefaults()
}

func applyPatch(cur domain.Settings, p UpdateInput) domain.Settings {
	if p.ProxyEnabled != nil {
		cur.ProxyEnabled = *p.ProxyEnabled
	}
	if p.Listen != nil {
		cur.Listen = strings.TrimSpace(*p.Listen)
	}
	if p.ConnectMode != nil {
		cur.ConnectMode = strings.TrimSpace(*p.ConnectMode)
	}
	if p.AuthEnabled != nil {
		cur.AuthEnabled = *p.AuthEnabled
	}
	if p.AuthStaticUsers != nil {
		cur.AuthStaticUsers = *p.AuthStaticUsers
	}
	if p.AuthRealm != nil {
		cur.AuthRealm = strings.TrimSpace(*p.AuthRealm)
	}
	if p.AuthBackend != nil {
		cur.AuthBackend = strings.TrimSpace(*p.AuthBackend)
	}
	if p.AuthCacheTTLMinutes != nil {
		cur.AuthCacheTTLMinutes = *p.AuthCacheTTLMinutes
	}
	if p.LDAPURL != nil {
		cur.LDAPURL = strings.TrimSpace(*p.LDAPURL)
	}
	if p.LDAPBaseDN != nil {
		cur.LDAPBaseDN = strings.TrimSpace(*p.LDAPBaseDN)
	}
	if p.LDAPBindDN != nil {
		cur.LDAPBindDN = strings.TrimSpace(*p.LDAPBindDN)
	}
	if p.LDAPBindPassword != nil && strings.TrimSpace(*p.LDAPBindPassword) != "" {
		cur.LDAPBindPassword = *p.LDAPBindPassword
	}
	if p.AccessLogRetentionDays != nil {
		cur.AccessLogRetentionDays = *p.AccessLogRetentionDays
	}
	return cur
}

func validateSettings(st domain.Settings) error {
	mode := proxyconfig.ConnectMode(strings.ToLower(st.ConnectMode))
	switch mode {
	case proxyconfig.ConnectTunnel, proxyconfig.ConnectMITM:
	default:
		return fmt.Errorf("invalid connect_mode %q", st.ConnectMode)
	}
	if st.Listen == "" {
		return errors.New("listen is required")
	}
	if err := proxyaccesslogservice.ValidateRetentionDays(st.AccessLogRetentionDays); err != nil {
		return err
	}
	if st.AuthEnabled {
		backend := strings.ToLower(strings.TrimSpace(st.AuthBackend))
		if backend == "static" && strings.TrimSpace(st.AuthStaticUsers) == "" {
			return errors.New("auth_static_users required for static backend")
		}
		if backend == "ldap" && st.LDAPURL == "" {
			return errors.New("ldap_url required for ldap backend")
		}
	}
	return nil
}
