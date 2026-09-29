package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyinspect/domain"
	"oktopus/internal/db/proxyinspect/repository"
	"oktopus/internal/id"
	"oktopus/internal/proxy/inspect"
)

type Reloader interface {
	ReloadProxy(ctx context.Context) error
}

const (
	CompileIdle     = "idle"
	CompileBuilding = "building"
	CompileReady    = "ready"
	CompileError    = "error"

	publishRetryInterval   = 3 * time.Second
	proxyReloadMinInterval = 5 * time.Second
)

type CompileStatusDTO struct {
	BuildStatus      string `json:"build_status"`
	BuildError       string `json:"build_error,omitempty"`
	BuildStartedAt   string `json:"build_started_at,omitempty"`
	BuildFinishedAt  string `json:"build_finished_at,omitempty"`
	ConfigRevision   string `json:"config_revision"`
	ActiveRevision   string `json:"active_revision"`
	RulesInSync      bool   `json:"rules_in_sync"`
	ActiveRules      int    `json:"active_rules"`
	EnabledRulesInDB int    `json:"enabled_rules_in_db"`
}

type Service struct {
	repo        repository.RulesRepository
	reloader    Reloader
	rt          *compileRuntime
	publishWake chan struct{}
	reloadMu    sync.Mutex
	lastReload  time.Time
}

func New(repo repository.RulesRepository) *Service {
	s := &Service{
		repo:        repo,
		rt:          newCompileRuntime(),
		publishWake: make(chan struct{}, 1),
	}
	go s.runPublisher()
	return s
}

func (s *Service) BindReloader(r Reloader) {
	s.reloader = r
}

func (s *Service) BootstrapCompile(ctx context.Context) error {
	rules, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	rev := revisionFromRules(rules)
	s.rt.setConfigRevision(rev)
	runner, err := compileRunner(rules)
	if err != nil {
		s.rt.finishBuild(nil, rev, err)
		return err
	}
	s.rt.finishBuild(runner, rev, nil)
	return nil
}

func (s *Service) ActiveRunner() *inspect.Runner {
	return s.rt.activeRunner()
}

func (s *Service) ListSummary(ctx context.Context) ([]domain.RuleSummary, error) {
	return s.repo.ListSummary(ctx)
}

func (s *Service) List(ctx context.Context) ([]domain.Rule, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetRule(ctx context.Context, ruleID uuid.UUID) (domain.Rule, error) {
	return s.repo.GetByID(ctx, ruleID)
}

func (s *Service) CompileStatus(ctx context.Context) (CompileStatusDTO, error) {
	rules, err := s.repo.List(ctx)
	if err != nil {
		return CompileStatusDTO{}, err
	}
	enabled := 0
	for _, r := range rules {
		if r.Enabled {
			enabled++
		}
	}
	return s.rt.status(enabled), nil
}

// ValidateScript проверяет Lua без сохранения.
func (s *Service) ValidateScript(_ context.Context, script string) error {
	return inspect.ValidateScript(script)
}

type SyncRuleInput struct {
	ID        *string
	Name      string
	Script    *string
	Action    int16
	Enabled   bool
	SortOrder int
}

func (s *Service) SyncAndPublish(ctx context.Context, inputs []SyncRuleInput) ([]domain.RuleSummary, error) {
	existing, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	existingScripts := make(map[uuid.UUID]string, len(existing))
	for _, r := range existing {
		existingScripts[r.ID] = r.Script
	}
	rules, err := normalizeSyncInputs(inputs, existingScripts)
	if err != nil {
		return nil, err
	}
	if err := validateRulesCompile(rules); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceAll(ctx, rules); err != nil {
		return nil, err
	}
	list, err := s.repo.ListSummary(ctx)
	if err != nil {
		return nil, err
	}
	s.rt.setConfigRevision(revisionFromRules(rules))
	s.requestPublish()
	return list, nil
}

func normalizeSyncInputs(inputs []SyncRuleInput, existingScripts map[uuid.UUID]string) ([]domain.Rule, error) {
	out := make([]domain.Rule, 0, len(inputs))
	for i, in := range inputs {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, fmt.Errorf("rule %d: name required", i+1)
		}
		var ruleID uuid.UUID
		if in.ID != nil && strings.TrimSpace(*in.ID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*in.ID))
			if err != nil {
				return nil, fmt.Errorf("rule %q: invalid id", name)
			}
			ruleID = parsed
		} else {
			ruleID = id.MustNew()
		}
		script := ""
		if in.Script != nil {
			script = *in.Script
		} else if prev, ok := existingScripts[ruleID]; ok {
			script = prev
		}
		if strings.TrimSpace(script) == "" {
			return nil, fmt.Errorf("rule %q: script required", name)
		}
		action := in.Action
		if action != domain.ActionDenyOnMatch && action != domain.ActionAllowOnMatch {
			return nil, fmt.Errorf("rule %q: invalid action", name)
		}
		out = append(out, domain.Rule{
			ID:        ruleID,
			Name:      name,
			Script:    script,
			Action:    action,
			Enabled:   in.Enabled,
			SortOrder: in.SortOrder,
		})
	}
	return out, nil
}

func validateRulesCompile(rules []domain.Rule) error {
	if len(rules) == 0 {
		return nil
	}
	inputs := make([]inspect.RuleInput, len(rules))
	for i, r := range rules {
		inputs[i] = inspect.RuleInput{
			ID:     r.ID,
			Name:   r.Name,
			Action: r.Action,
			Script: r.Script,
		}
	}
	_, err := inspect.BuildProgram(inputs)
	return err
}

func compileRunner(rules []domain.Rule) (*inspect.Runner, error) {
	inputs := make([]inspect.RuleInput, 0, len(rules))
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		inputs = append(inputs, inspect.RuleInput{
			ID:     r.ID,
			Name:   r.Name,
			Action: r.Action,
			Script: r.Script,
		})
	}
	prog, err := inspect.BuildProgram(inputs)
	if err != nil {
		return nil, err
	}
	return inspect.NewRunner(prog), nil
}

func revisionFromRules(rules []domain.Rule) string {
	h := sha256.New()
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		h.Write([]byte(r.Name))
		h.Write([]byte{0})
		h.Write([]byte(r.Script))
		h.Write([]byte{0})
		h.Write([]byte{byte(r.Action)})
		h.Write([]byte{0})
		var sortBuf [4]byte
		sortBuf[0] = byte(r.SortOrder >> 24)
		sortBuf[1] = byte(r.SortOrder >> 16)
		sortBuf[2] = byte(r.SortOrder >> 8)
		sortBuf[3] = byte(r.SortOrder)
		h.Write(sortBuf[:])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
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
			if err == nil && s.rt.getConfigRevision() == s.rt.getActiveRevision() {
				break
			}
			time.Sleep(publishRetryInterval)
		}
	}
}

func (s *Service) attemptPublish() error {
	ctx := context.Background()
	prevRev := s.rt.getActiveRevision()
	s.rt.beginBuild()
	rules, err := s.repo.List(ctx)
	if err != nil {
		s.rt.finishBuild(nil, "", err)
		return err
	}
	rev := revisionFromRules(rules)
	runner, err := compileRunner(rules)
	if err != nil {
		s.rt.finishBuild(nil, rev, err)
		return err
	}
	s.rt.finishBuild(runner, rev, nil)
	if rev == prevRev {
		return nil
	}
	return s.reloadProxyThrottled(ctx)
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
		s.rt.setReloadError(fmt.Errorf("reload proxy: %w", err).Error())
		return err
	}
	s.lastReload = time.Now()
	return nil
}
