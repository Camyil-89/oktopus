package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/apperr"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"
	"oktopus/internal/db/proxyinstances/domain"
	"oktopus/internal/db/proxyinstances/repository"
	"oktopus/internal/id"
	"oktopus/internal/pki"
	proxyconfig "oktopus/internal/proxy/config"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/inspect"
)

// FleetApplier применяет runtime всех инстансов.
type FleetApplier interface {
	ApplyFleet(ctx context.Context, runtimes []InstanceRuntime) error
	Stop()
}

// InstanceRuntime — один слушатель прокси.
type InstanceRuntime struct {
	InstanceID    uuid.UUID
	Name          string
	Config        proxyconfig.Config
	ACLEngine     *acl.Engine
	InspectRunner *inspect.Runner
}

type Service struct {
	repo    repository.Repository
	acl     *proxyaclservice.Service
	inspect *proxyinspectservice.Service
	applier FleetApplier
}

func New(repo repository.Repository, acl *proxyaclservice.Service, applier FleetApplier) *Service {
	return &Service{repo: repo, acl: acl, applier: applier}
}

func (s *Service) BindInspect(ins *proxyinspectservice.Service) {
	s.inspect = ins
}

func (s *Service) BindApplier(a FleetApplier) {
	s.applier = a
}

func (s *Service) List(ctx context.Context) ([]domain.Instance, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, instanceID uuid.UUID) (domain.Instance, error) {
	return s.repo.Get(ctx, instanceID)
}

type CreateInput struct {
	Name   string
	Listen string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Instance, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.Instance{}, apperr.New(apperr.InstanceNameRequired)
	}
	listen := strings.TrimSpace(in.Listen)
	if listen == "" {
		return domain.Instance{}, apperr.New(apperr.ListenRequired)
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return domain.Instance{}, err
	}
	sortOrder := len(list)
	instID := id.MustNew()
	inst := domain.DefaultInstance(instID, name, listen, sortOrder)
	if err := os.MkdirAll(domain.ConfigDir(instID), 0o755); err != nil {
		return domain.Instance{}, fmt.Errorf("mkdir config: %w", err)
	}
	inserted, err := s.repo.Insert(ctx, inst)
	if err != nil {
		return domain.Instance{}, mapInstanceRepoErr(err)
	}
	if err := s.repo.InsertACLPolicy(ctx, id.MustNew(), inserted.ID); err != nil {
		return domain.Instance{}, err
	}
	if s.acl != nil {
		if err := s.acl.BootstrapInstance(ctx, inserted.ID); err != nil {
			return domain.Instance{}, err
		}
	}
	if s.inspect != nil {
		if err := s.inspect.BootstrapInstance(ctx, inserted.ID); err != nil {
			return domain.Instance{}, err
		}
	}
	inserted, err = s.applyFleetAfter(ctx, inserted)
	return inserted, err
}

func (s *Service) applyFleetAfter(ctx context.Context, inst domain.Instance) (domain.Instance, error) {
	_, err := s.applyFleet(ctx)
	return inst, err
}

type UpdateInput struct {
	Name                *string
	Enabled             *bool
	Listen              *string
	ConnectMode         *string
	AuthEnabled         *bool
	AuthStaticUsers     *string
	AuthRealm           *string
	AuthBackend         *string
	AuthCacheTTLMinutes *int64
	LDAPURL             *string
	LDAPBaseDN          *string
	LDAPBindDN          *string
	LDAPBindPassword    *string
	SortOrder           *int
}

func (s *Service) Update(ctx context.Context, instanceID uuid.UUID, patch UpdateInput) (domain.Instance, error) {
	cur, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return domain.Instance{}, err
	}
	next := applyPatch(cur, patch)
	if err := validateInstance(next); err != nil {
		return domain.Instance{}, err
	}
	updated, err := s.repo.Update(ctx, next)
	if err != nil {
		return domain.Instance{}, mapInstanceRepoErr(err)
	}
	_, err = s.applyFleet(ctx)
	if err != nil {
		return domain.Instance{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, instanceID uuid.UUID) error {
	if err := s.repo.Delete(ctx, instanceID); err != nil {
		return err
	}
	_ = os.RemoveAll(domain.ConfigDir(instanceID))
	if s.acl != nil {
		s.acl.DropInstance(instanceID)
	}
	if s.inspect != nil {
		s.inspect.DropInstance(instanceID)
	}
	_, err := s.applyFleet(ctx)
	return err
}

type GenerateCAInput struct {
	KeyBits      int
	CommonName   string
	Organization string
	Country      string
	ValidDays    int
}

func (s *Service) GenerateCA(ctx context.Context, instanceID uuid.UUID, in GenerateCAInput) (domain.Instance, error) {
	if _, err := s.repo.Get(ctx, instanceID); err != nil {
		return domain.Instance{}, err
	}
	if in.KeyBits == 0 {
		in.KeyBits = 4096
	}
	if in.KeyBits < 2048 {
		return domain.Instance{}, apperr.New(apperr.KeyBitsTooSmall)
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
	outDir := domain.ConfigDir(instanceID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return domain.Instance{}, err
	}
	if err := pki.GenerateCA(pki.CAConfig{
		CommonName:   in.CommonName,
		Organization: in.Organization,
		Country:      in.Country,
		ValidDays:    in.ValidDays,
		KeyBits:      in.KeyBits,
		OutDir:       outDir,
		Force:        true,
	}); err != nil {
		return domain.Instance{}, err
	}
	cert, key := domain.DefaultCAPaths(instanceID)
	updated, err := s.repo.UpdateCAPaths(ctx, instanceID, cert, key)
	if err != nil {
		return domain.Instance{}, err
	}
	_, err = s.applyFleet(ctx)
	if err != nil {
		return domain.Instance{}, err
	}
	return updated, nil
}

func (s *Service) UploadCA(ctx context.Context, instanceID uuid.UUID, certReader, keyReader io.Reader) (domain.Instance, error) {
	if _, err := s.repo.Get(ctx, instanceID); err != nil {
		return domain.Instance{}, err
	}
	certPEM, err := io.ReadAll(certReader)
	if err != nil {
		return domain.Instance{}, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := io.ReadAll(keyReader)
	if err != nil {
		return domain.Instance{}, fmt.Errorf("read key: %w", err)
	}
	outDir := domain.ConfigDir(instanceID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return domain.Instance{}, err
	}
	if _, _, err := pki.SaveCAFiles(outDir, certPEM, keyPEM, true); err != nil {
		return domain.Instance{}, err
	}
	cert, key := domain.DefaultCAPaths(instanceID)
	updated, err := s.repo.UpdateCAPaths(ctx, instanceID, cert, key)
	if err != nil {
		return domain.Instance{}, err
	}
	_, err = s.applyFleet(ctx)
	if err != nil {
		return domain.Instance{}, err
	}
	return updated, nil
}

type CAStatusDTO struct {
	CertInstalled bool
	KeyInstalled  bool
	ValidUntil    *time.Time
}

func (s *Service) CAStatus(ctx context.Context, instanceID uuid.UUID) (CAStatusDTO, error) {
	st, err := s.repo.Get(ctx, instanceID)
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

func (s *Service) CACertPath(ctx context.Context, instanceID uuid.UUID) (string, error) {
	st, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(st.CACertPath); err != nil {
		return "", apperr.New(apperr.CertNotFound)
	}
	return st.CACertPath, nil
}

func (s *Service) CAKeyPath(ctx context.Context, instanceID uuid.UUID) (string, error) {
	st, err := s.repo.Get(ctx, instanceID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(st.CAKeyPath); err != nil {
		return "", apperr.New(apperr.KeyNotFound)
	}
	return st.CAKeyPath, nil
}

func (s *Service) LoadFleetRuntime(ctx context.Context) ([]InstanceRuntime, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []InstanceRuntime
	for _, inst := range list {
		if !inst.Enabled {
			continue
		}
		rt, err := s.runtimeForInstance(ctx, inst)
		if err != nil {
			return nil, err
		}
		out = append(out, rt)
	}
	return out, nil
}

func (s *Service) runtimeForInstance(ctx context.Context, inst domain.Instance) (InstanceRuntime, error) {
	var engine *acl.Engine
	if s.acl != nil {
		engine = s.acl.ActiveEngine(inst.ID)
	} else {
		engine = acl.EmptyEngine()
	}
	var inspectRunner *inspect.Runner
	if s.inspect != nil {
		inspectRunner = s.inspect.ActiveRunner(inst.ID)
	} else {
		inspectRunner = inspect.NewRunner(&inspect.Program{})
	}
	return InstanceRuntime{
		InstanceID:    inst.ID,
		Name:          inst.Name,
		Config:        ToProxyConfig(inst),
		ACLEngine:     engine,
		InspectRunner: inspectRunner,
	}, nil
}

func (s *Service) applyFleet(ctx context.Context) ([]domain.Instance, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if s.applier == nil {
		return list, nil
	}
	runtimes, err := s.LoadFleetRuntime(ctx)
	if err != nil {
		return nil, apperr.MapProxyApply(err)
	}
	if err := s.applier.ApplyFleet(ctx, runtimes); err != nil {
		return nil, apperr.MapProxyApply(err)
	}
	return list, nil
}

func (s *Service) ReloadFleet(ctx context.Context) error {
	_, err := s.applyFleet(ctx)
	return err
}

func ToProxyConfig(inst domain.Instance) proxyconfig.Config {
	mode := proxyconfig.ConnectMode(strings.ToLower(strings.TrimSpace(inst.ConnectMode)))
	var cacheTTL time.Duration
	if inst.AuthCacheTTLMinutes > 0 {
		cacheTTL = time.Duration(inst.AuthCacheTTLMinutes) * time.Minute
	}
	return proxyconfig.Config{
		Listen:  inst.Listen,
		Connect: mode,
		Auth: proxyconfig.AuthConfig{
			Enabled:     inst.AuthEnabled,
			Realm:       inst.AuthRealm,
			Backend:     inst.AuthBackend,
			StaticUsers: inst.AuthStaticUsers,
			CacheTTL:    cacheTTL,
			LDAP: proxyconfig.LDAPConfig{
				URL:          inst.LDAPURL,
				BaseDN:       inst.LDAPBaseDN,
				BindDN:       inst.LDAPBindDN,
				BindPassword: inst.LDAPBindPassword,
			},
		},
		CACertPath: inst.CACertPath,
		CAKeyPath:  inst.CAKeyPath,
	}.WithDefaults()
}

func applyPatch(cur domain.Instance, p UpdateInput) domain.Instance {
	if p.Name != nil {
		cur.Name = strings.TrimSpace(*p.Name)
	}
	if p.Enabled != nil {
		cur.Enabled = *p.Enabled
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
	if p.SortOrder != nil {
		cur.SortOrder = *p.SortOrder
	}
	return cur
}

func validateInstance(inst domain.Instance) error {
	mode := proxyconfig.ConnectMode(strings.ToLower(inst.ConnectMode))
	switch mode {
	case proxyconfig.ConnectTunnel, proxyconfig.ConnectMITM:
	default:
		return apperr.New(apperr.InvalidConnectMode)
	}
	if strings.TrimSpace(inst.Name) == "" {
		return apperr.New(apperr.InstanceNameRequired)
	}
	if inst.Listen == "" {
		return apperr.New(apperr.ListenRequired)
	}
	if inst.AuthEnabled {
		backend := strings.ToLower(strings.TrimSpace(inst.AuthBackend))
		if backend == "static" && strings.TrimSpace(inst.AuthStaticUsers) == "" {
			return apperr.New(apperr.AuthStaticUsersRequired)
		}
		if backend == "ldap" && inst.LDAPURL == "" {
			return apperr.New(apperr.LDAPURLRequired)
		}
	}
	return nil
}

func mapInstanceRepoErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrListenTaken):
		return apperr.New(apperr.ListenAddressInUse)
	case errors.Is(err, domain.ErrNameTaken):
		return apperr.New(apperr.InstanceNameTaken)
	default:
		return err
	}
}
