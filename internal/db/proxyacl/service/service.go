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
	rt          *compileRuntime
	publishWake chan struct{}
	listPoll    *listPollScheduler
	reloadMu    sync.Mutex
	lastReload  time.Time
}

func New(repo repository.RulesRepository) *Service {
	s := &Service{
		repo:        repo,
		rt:          newCompileRuntime(),
		publishWake: make(chan struct{}, 1),
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
	pol, err := s.repo.GetPolicy(ctx)
	if err != nil {
		return err
	}
	lists, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return err
	}
	s.rt.setEnabledRulesInDB(countSquidSources(lists, pol))
	refLists, err := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	if err != nil {
		return err
	}
	rev := revisionFromSquid(pol.ConfigText, refLists)
	s.rt.setConfigRevision(rev)
	s.rt.beginBuild()
	engine, _, sniRep, err := compileSquidPolicy(pol.ConfigText, refLists)
	if err != nil {
		s.rt.finishBuild(nil, rev, acl.SNIPatternIndexReport{}, err)
		return err
	}
	s.rt.finishBuild(engine, rev, sniRep, nil)
	startup.MarkACLDBSynced()
	return nil
}

func (s *Service) ActiveEngine() *acl.Engine {
	return s.rt.activeEngine()
}

func (s *Service) CompileStatus(ctx context.Context) (CompileStatusDTO, error) {
	_ = ctx
	return s.rt.statusForAPI(), nil
}

func (s *Service) requestPublish() {
	select {
	case s.publishWake <- struct{}{}:
	default:
	}
}

func (s *Service) runPublisher() {
	for range s.publishWake {
		for {
			err := s.attemptPublish()
			if err == nil && s.rt.isPublished() {
				startup.MarkACLDBSynced()
				break
			}
			time.Sleep(publishRetryInterval)
		}
	}
}

func (s *Service) attemptPublish() error {
	ctx := context.Background()
	prevRev := s.rt.getActiveRevision()
	pol, err := s.repo.GetPolicy(ctx)
	if err != nil {
		return err
	}
	lists, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return err
	}
	s.rt.setEnabledRulesInDB(countSquidSources(lists, pol))
	refLists, err := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	if err != nil {
		return err
	}
	rev := revisionFromSquid(pol.ConfigText, refLists)
	s.rt.setConfigRevision(rev)

	if rev != prevRev {
		s.rt.beginBuild()
		engine, _, sniRep, err := compileSquidPolicy(pol.ConfigText, refLists)
		if err != nil {
			s.rt.finishBuild(nil, rev, acl.SNIPatternIndexReport{}, err)
			return err
		}
		s.rt.finishBuild(engine, rev, sniRep, nil)
	} else {
		s.rt.markReadyIfSynced()
	}

	if err := s.reloadProxyThrottled(ctx); err != nil {
		return fmt.Errorf("reload proxy: %w", err)
	}
	s.rt.markReadyIfSynced()
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

func (s *Service) GetPolicy(ctx context.Context) (domain.Policy, error) {
	return s.repo.GetPolicy(ctx)
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

func (s *Service) SetPolicyAndPublish(ctx context.Context, configText string) (domain.Policy, error) {
	if err := validateSquidCompile(ctx, &policyValidateRepo{s: s, draftPolicy: configText, hasPolicy: true}); err != nil {
		return domain.Policy{}, err
	}
	pol, err := s.repo.SetPolicy(ctx, configText)
	if err != nil {
		return domain.Policy{}, err
	}
	refLists, _ := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	s.rt.setConfigRevision(revisionFromSquid(pol.ConfigText, refLists))
	s.requestPublish()
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
	if err := validateSquidCompile(ctx, &policyValidateRepo{s: s, draftLists: lists, hasLists: true}); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceNamedLists(ctx, lists); err != nil {
		return nil, err
	}
	out, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return nil, err
	}
	pol, _ := s.repo.GetPolicy(ctx)
	refLists, _ := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	s.rt.setConfigRevision(revisionFromSquid(pol.ConfigText, refLists))
	for _, l := range out {
		if l.SourceMode == domain.ListSourceModeRemote && s.listPoll != nil {
			s.listPoll.scheduleSoon(l.ID)
		}
	}
	s.requestPublish()
	return out, nil
}

// policyValidateRepo подставляет черновик policy/lists для проверки компиляции.
type policyValidateRepo struct {
	s           *Service
	draftPolicy string
	draftLists  []domain.NamedList
	hasPolicy   bool
	hasLists    bool
}

func (p *policyValidateRepo) GetPolicy(ctx context.Context) (domain.Policy, error) {
	if p.hasPolicy {
		return domain.Policy{ConfigText: p.draftPolicy}, nil
	}
	return p.s.repo.GetPolicy(ctx)
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

func (s *Service) Evaluate(_ context.Context, in EvaluateInput) EvaluateResultDTO {
	engine := s.rt.activeEngine()
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
