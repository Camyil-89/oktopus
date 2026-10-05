package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyacl/domain"
	"oktopus/internal/db/proxyacl/repository"
	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/startup"
)

// Reloader перезагружает прокси после публикации ACL.
type Reloader interface {
	ReloadProxy(ctx context.Context) error
}

const (
	publishRetryInterval   = 3 * time.Second
	proxyReloadMinInterval = 5 * time.Second
)

type Service struct {
	repo        repository.RulesRepository
	reloader    Reloader
	runtimes    sync.Map // uuid.UUID -> *compileRuntime
	publishWake chan uuid.UUID
	listPoll    *listPollScheduler
	reloadMu    sync.Mutex
	lastReload  time.Time
}

func New(repo repository.RulesRepository) *Service {
	s := &Service{
		repo:        repo,
		publishWake: make(chan uuid.UUID, 64),
	}
	s.listPoll = newListPollScheduler(s)
	go s.runPublisher()
	return s
}

// StartListPoller запускает фоновый опрос remote-списков (после bootstrap ACL).
func (s *Service) StartListPoller() {
	go s.runListPoller()
}

func (s *Service) BindReloader(r Reloader) {
	s.reloader = r
}

// BootstrapCompile синхронно собирает ACL при старте.
func (s *Service) BootstrapCompile(ctx context.Context) error {
	ids, err := s.repo.ListPolicyInstanceIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.bootstrapInstance(ctx, id); err != nil {
			return err
		}
	}
	startup.MarkACLDBSynced()
	return nil
}

func (s *Service) CompileStatus(ctx context.Context, instanceID uuid.UUID) (CompileStatusDTO, error) {
	_ = ctx
	return s.CompileStatusFor(instanceID), nil
}

func (s *Service) runPublisher() {
	for instanceID := range s.publishWake {
		for {
			err := s.attemptPublish(instanceID)
			rt := s.runtimeFor(instanceID)
			if err == nil && rt.isPublished() {
				startup.MarkACLDBSynced()
				break
			}
			time.Sleep(publishRetryInterval)
		}
	}
}

func (s *Service) attemptPublish(instanceID uuid.UUID) error {
	ctx := context.Background()
	rt := s.runtimeFor(instanceID)
	prevRev := rt.getActiveRevision()
	pol, err := s.repo.GetPolicy(ctx, instanceID)
	if err != nil {
		return err
	}
	lists, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return err
	}
	rt.setEnabledRulesInDB(countSquidSources(lists, pol))
	src := instancePolicySource{s.repo, instanceID}
	refLists, err := referencedNamedLists(ctx, src, pol.ConfigText)
	if err != nil {
		return err
	}
	rev := revisionFromSquid(pol.ConfigText, refLists)
	rt.setConfigRevision(rev)

	if rev != prevRev {
		rt.beginBuild()
		engine, _, sniRep, err := compileSquidPolicy(pol.ConfigText, refLists)
		if err != nil {
			rt.finishBuild(nil, rev, acl.SNIPatternIndexReport{}, err)
			return err
		}
		rt.finishBuild(engine, rev, sniRep, nil)
	} else {
		rt.markReadyIfSynced()
	}

	if err := s.reloadProxyThrottled(ctx); err != nil {
		return fmt.Errorf("reload proxy: %w", err)
	}
	rt.markReadyIfSynced()
	return nil
}

func (s *Service) reloadProxyThrottled(ctx context.Context) error {
	if s.reloader == nil {
		return nil
	}
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	if wait := proxyReloadMinInterval - time.Since(s.lastReload); wait > 0 {
		time.Sleep(wait)
	}
	if err := s.reloader.ReloadProxy(ctx); err != nil {
		return err
	}
	s.lastReload = time.Now()
	return nil
}

func (s *Service) GetPolicy(ctx context.Context, instanceID uuid.UUID) (domain.Policy, error) {
	return s.repo.GetPolicy(ctx, instanceID)
}

// AnalyzePolicy проверяет конфиг; listInputs == nil — lists из БД.
func (s *Service) AnalyzePolicy(ctx context.Context, configText string, listInputs []squid.ListInput) squid.AnalyzeResult {
	if listInputs != nil {
		return squid.AnalyzeValidate(configText, listInputs)
	}
	cfg, parseDiags := squid.ParsePolicyConfig(configText)
	if cfg == nil {
		return squid.AnalyzeResult{OK: false, Diagnostics: parseDiags}
	}
	names := squid.ListNamesToMergeFromDB(cfg)
	if len(names) == 0 {
		return squid.AnalyzeParsedConfig(cfg, nil, parseDiags, false)
	}
	dbLists, err := s.repo.ListNamedListsByNames(ctx, names)
	if err != nil {
		return squid.AnalyzeResult{
			OK: false,
			Diagnostics: []squid.Diagnostic{{
				Line: 0, Severity: squid.SeverityError, Code: "internal",
				Message: err.Error(),
			}},
		}
	}
	return squid.AnalyzeParsedConfig(cfg, domainListsToSquid(dbLists), parseDiags, false)
}

func (s *Service) SetPolicyAndPublish(ctx context.Context, instanceID uuid.UUID, configText string) (domain.Policy, error) {
	if err := validateSquidCompile(ctx, &policyValidateRepo{s: s, instanceID: instanceID, draftPolicy: configText, hasPolicy: true}); err != nil {
		return domain.Policy{}, err
	}
	pol, err := s.repo.SetPolicy(ctx, instanceID, configText)
	if err != nil {
		return domain.Policy{}, err
	}
	refLists, _ := referencedNamedLists(ctx, instancePolicySource{s.repo, instanceID}, pol.ConfigText)
	s.runtimeFor(instanceID).setConfigRevision(revisionFromSquid(pol.ConfigText, refLists))
	s.requestPublishInstance(instanceID)
	return pol, nil
}

func (s *Service) ListNamedLists(ctx context.Context) ([]domain.NamedList, error) {
	return s.repo.ListNamedLists(ctx)
}

func (s *Service) ListNamedListsSummary(ctx context.Context) ([]domain.NamedListSummary, error) {
	return s.repo.ListNamedListsSummary(ctx)
}

func (s *Service) GetNamedList(ctx context.Context, id uuid.UUID) (domain.NamedList, error) {
	return s.repo.GetNamedList(ctx, id)
}

func (s *Service) SyncNamedListsAndPublish(ctx context.Context, inputs []SyncNamedListInput) ([]domain.NamedList, error) {
	lists, err := normalizeNamedLists(inputs)
	if err != nil {
		return nil, err
	}
	lists, err = s.preserveRemoteListBodies(ctx, lists)
	if err != nil {
		return nil, err
	}
	lists, err = s.ensureRemoteListBodies(ctx, lists)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListPolicyInstanceIDs(ctx)
	if err != nil {
		return nil, err
	}
	for _, instanceID := range ids {
		if err := validateSquidCompile(ctx, &policyValidateRepo{s: s, instanceID: instanceID, draftLists: lists, hasLists: true}); err != nil {
			return nil, err
		}
	}
	if err := s.repo.ReplaceNamedLists(ctx, lists); err != nil {
		return nil, err
	}
	out, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range out {
		if l.SourceMode == domain.ListSourceModeRemote && s.listPoll != nil {
			s.listPoll.scheduleSoon(l.ID)
		}
	}
	s.requestPublishAllInstances(ctx)
	return out, nil
}

// policyValidateRepo подставляет черновик policy/lists для проверки компиляции.
type policyValidateRepo struct {
	s           *Service
	instanceID  uuid.UUID
	draftPolicy string
	draftLists  []domain.NamedList
	hasPolicy   bool
	hasLists    bool
}

func (p *policyValidateRepo) GetPolicy(ctx context.Context) (domain.Policy, error) {
	if p.hasPolicy {
		return domain.Policy{ConfigText: p.draftPolicy}, nil
	}
	return p.s.repo.GetPolicy(ctx, p.instanceID)
}

func (p *policyValidateRepo) ListNamedListsByNames(ctx context.Context, names []string) ([]domain.NamedList, error) {
	if p.hasLists {
		return filterNamedListsByNames(p.draftLists, names), nil
	}
	return p.s.repo.ListNamedListsByNames(ctx, names)
}

// EvaluateInput — проверка запроса против опубликованной Squid-политики.
type EvaluateInput struct {
	SNI      string
	Path     string
	SrcIP    string
	DstIP    string
	DstPort  int
	Username string
	Groups   []string
}

// EvaluateStepDTO — шаг цепочки для API.
type EvaluateStepDTO struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
}

// EvaluateResultDTO — итог проверки.
type EvaluateResultDTO struct {
	Allowed bool              `json:"allowed"`
	Steps   []EvaluateStepDTO `json:"steps"`
}

func (s *Service) Evaluate(_ context.Context, instanceID uuid.UUID, in EvaluateInput) EvaluateResultDTO {
	engine := s.ActiveEngine(instanceID)
	id := auth.Identity{
		Username: strings.TrimSpace(in.Username),
		Groups:   in.Groups,
	}
	ex := engine.Explain(id, acl.RequestFields{
		SNI:     in.SNI,
		Path:    in.Path,
		SrcIP:   net.ParseIP(strings.TrimSpace(in.SrcIP)),
		DstIP:   net.ParseIP(strings.TrimSpace(in.DstIP)),
		DstPort: in.DstPort,
	})
	steps := make([]EvaluateStepDTO, 0, len(ex.Steps))
	for _, step := range ex.Steps {
		steps = append(steps, EvaluateStepDTO{
			Kind:    string(step.Kind),
			Message: step.Message,
		})
	}
	return EvaluateResultDTO{Allowed: ex.Allowed, Steps: steps}
}
