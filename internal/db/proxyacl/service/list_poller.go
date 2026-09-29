package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyacl/domain"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/startup"
)

const (
	listPollTickInterval = time.Minute
	listPollErrorBackoff = 5 * time.Minute
)

type listPollScheduler struct {
	svc  *Service
	mu   sync.Mutex
	next map[uuid.UUID]time.Time
}

func newListPollScheduler(svc *Service) *listPollScheduler {
	return &listPollScheduler{
		svc:  svc,
		next: make(map[uuid.UUID]time.Time),
	}
}

func (p *listPollScheduler) scheduleSoon(id uuid.UUID) {
	p.mu.Lock()
	delete(p.next, id)
	p.mu.Unlock()
}

func (p *listPollScheduler) scheduleAfter(id uuid.UUID, d time.Duration) {
	p.mu.Lock()
	p.next[id] = time.Now().Add(d)
	p.mu.Unlock()
}

func (p *listPollScheduler) isDue(id uuid.UUID) bool {
	p.mu.Lock()
	next, ok := p.next[id]
	p.mu.Unlock()
	if !ok {
		return true
	}
	return !time.Now().Before(next)
}

func (p *listPollScheduler) prune(keep map[uuid.UUID]struct{}) {
	p.mu.Lock()
	for id := range p.next {
		if _, ok := keep[id]; !ok {
			delete(p.next, id)
		}
	}
	p.mu.Unlock()
}

func (s *Service) runListPoller() {
	ticker := time.NewTicker(listPollTickInterval)
	defer ticker.Stop()

	go func() {
		if err := startup.WaitRemoteListPollStartup(context.Background()); err != nil {
			return
		}
		if s.listPoll != nil {
			s.listPoll.pollAllRemoteOnStartup(context.Background())
		}
	}()

	for range ticker.C {
		if s.listPoll != nil {
			s.listPoll.tick(context.Background())
		}
	}
}

func (p *listPollScheduler) pollAllRemoteOnStartup(ctx context.Context) {
	summaries, err := p.svc.repo.ListNamedListsSummary(ctx)
	if err != nil {
		log.Printf("acl list poll: startup: %v", err)
		return
	}
	var remote int
	for _, sum := range summaries {
		if sum.SourceMode != domain.ListSourceModeRemote {
			continue
		}
		remote++
		p.pollSummaryList(ctx, sum, "startup")
	}
	if remote > 0 {
		log.Printf("acl list poll: startup: finished %d remote list(s)", remote)
	}
}

func (p *listPollScheduler) tick(ctx context.Context) {
	summaries, err := p.svc.repo.ListNamedListsSummary(ctx)
	if err != nil {
		return
	}
	keep := make(map[uuid.UUID]struct{})
	for _, sum := range summaries {
		if sum.SourceMode != domain.ListSourceModeRemote {
			continue
		}
		keep[sum.ID] = struct{}{}
		if !p.isDue(sum.ID) {
			continue
		}
		p.pollSummaryList(ctx, sum, "scheduled")
	}
	p.prune(keep)
}

func (p *listPollScheduler) pollSummaryList(ctx context.Context, sum domain.NamedListSummary, trigger string) {
	list, err := p.svc.repo.GetNamedList(ctx, sum.ID)
	if err != nil {
		log.Printf("acl list poll [%s]: %q: load failed: %v", trigger, sum.Name, err)
		p.scheduleAfter(sum.ID, listPollErrorBackoff)
		return
	}
	if err := p.pollOne(ctx, list, sum.SourceURL, trigger); err != nil {
		p.scheduleAfter(sum.ID, listPollErrorBackoff)
		return
	}
	p.scheduleAfter(sum.ID, remotePollInterval(sum.PollIntervalMinutes))
}

func (s *Service) pollRemoteList(ctx context.Context, list domain.NamedList, sourceURL string, trigger string) error {
	body, err := fetchListBodyFromURL(sourceURL)
	if err != nil {
		log.Printf("acl list poll [%s]: %q: fetch failed: %v", trigger, list.Name, err)
		return err
	}
	if body == "" {
		err := fmt.Errorf("remote source returned empty body")
		log.Printf("acl list poll [%s]: %q: %v", trigger, list.Name, err)
		return err
	}
	changed, published, err := s.applyRemoteListBody(ctx, list, body)
	if err != nil {
		log.Printf("acl list poll [%s]: %q: %v", trigger, list.Name, err)
		return err
	}
	if changed {
		if published {
			log.Printf("acl list poll [%s]: %q: updated, acl publish queued", trigger, list.Name)
		} else {
			log.Printf("acl list poll [%s]: %q: updated (acl already in sync)", trigger, list.Name)
		}
		return nil
	}
	log.Printf("acl list poll [%s]: %q: unchanged", trigger, list.Name)
	return nil
}

func (p *listPollScheduler) pollOne(ctx context.Context, l domain.NamedList, sourceURL string, trigger string) error {
	return p.svc.pollRemoteList(ctx, l, sourceURL, trigger)
}

func (s *Service) PollNamedListNow(ctx context.Context, id uuid.UUID, sourceURLOverride string) (domain.NamedList, error) {
	list, err := s.repo.GetNamedList(ctx, id)
	if err != nil {
		return domain.NamedList{}, err
	}
	if list.SourceMode != domain.ListSourceModeRemote {
		return domain.NamedList{}, fmt.Errorf("list %q: source_mode is not remote", list.Name)
	}
	sourceURL := strings.TrimSpace(list.SourceURL)
	if o := strings.TrimSpace(sourceURLOverride); o != "" {
		if err := validateRemoteSourceURL(o); err != nil {
			return domain.NamedList{}, fmt.Errorf("list %q: %w", list.Name, err)
		}
		sourceURL = o
	}
	if sourceURL == "" {
		return domain.NamedList{}, fmt.Errorf("list %q: source_url is required", list.Name)
	}
	if err := s.pollRemoteList(ctx, list, sourceURL, "manual"); err != nil {
		return domain.NamedList{}, fmt.Errorf("list %q: %w", list.Name, err)
	}
	if s.listPoll != nil {
		s.listPoll.scheduleAfter(list.ID, remotePollInterval(list.PollIntervalMinutes))
	}
	return s.repo.GetNamedList(ctx, id)
}

func remotePollInterval(minutes int) time.Duration {
	if minutes < 1 {
		minutes = 60
	}
	return time.Duration(minutes) * time.Minute
}

func (s *Service) applyRemoteListBody(ctx context.Context, list domain.NamedList, newBody string) (changed bool, published bool, err error) {
	canonicalNew := canonicalListBody(newBody)
	if canonicalNew == canonicalListBody(list.Body) {
		return false, false, nil
	}
	if err := validateSquidCompileWithListBody(ctx, s, list, canonicalNew); err != nil {
		return false, false, err
	}
	if _, err := s.repo.UpdateNamedListBody(ctx, list.ID, canonicalNew); err != nil {
		return false, false, err
	}
	pol, _ := s.repo.GetPolicy(ctx)
	refLists, _ := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	rev := revisionFromSquid(pol.ConfigText, refLists)
	s.rt.setConfigRevision(rev)
	if rev == s.rt.getActiveRevision() {
		return true, false, nil
	}
	s.requestPublish()
	return true, true, nil
}

func validateSquidCompileWithListBody(ctx context.Context, s *Service, list domain.NamedList, body string) error {
	pol, err := s.repo.GetPolicy(ctx)
	if err != nil {
		return err
	}
	cfg, _ := squid.ParsePolicyConfig(pol.ConfigText)
	if cfg == nil {
		return validateSquidCompile(ctx, &policyValidateRepo{s: s})
	}
	needed := false
	for _, name := range squid.ListNamesToMergeFromDB(cfg) {
		if strings.EqualFold(name, list.Name) {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}
	refLists, err := referencedNamedLists(ctx, s.repo, pol.ConfigText)
	if err != nil {
		return err
	}
	for i := range refLists {
		if refLists[i].ID == list.ID {
			refLists[i].Body = body
			break
		}
	}
	return validateSquidCompile(ctx, &policyValidateRepo{s: s, draftLists: refLists, hasLists: true})
}
