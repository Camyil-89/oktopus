package service

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyacl/domain"
	"oktopus/internal/db/proxyacl/repository"
	"oktopus/internal/proxy/acl"
)

func (s *Service) runtimeFor(instanceID uuid.UUID) *compileRuntime {
	if v, ok := s.runtimes.Load(instanceID); ok {
		return v.(*compileRuntime)
	}
	rt := newCompileRuntime()
	actual, _ := s.runtimes.LoadOrStore(instanceID, rt)
	return actual.(*compileRuntime)
}

func (s *Service) DropInstance(instanceID uuid.UUID) {
	s.runtimes.Delete(instanceID)
}

func (s *Service) ActiveEngine(instanceID uuid.UUID) *acl.Engine {
	return s.runtimeFor(instanceID).activeEngine()
}

func (s *Service) bootstrapInstance(ctx context.Context, instanceID uuid.UUID) error {
	rt := s.runtimeFor(instanceID)
	pol, err := s.repo.GetPolicy(ctx, instanceID)
	if err != nil {
		return err
	}
	lists, err := s.repo.ListNamedLists(ctx)
	if err != nil {
		return err
	}
	rt.setEnabledRulesInDB(countSquidSources(lists, pol))
	refLists, err := referencedNamedLists(ctx, instancePolicySource{s.repo, instanceID}, pol.ConfigText)
	if err != nil {
		return err
	}
	rev := revisionFromSquid(pol.ConfigText, refLists)
	rt.setConfigRevision(rev)
	rt.beginBuild()
	engine, _, sniRep, err := compileSquidPolicy(pol.ConfigText, refLists)
	if err != nil {
		rt.finishBuild(nil, rev, acl.SNIPatternIndexReport{}, err)
		return err
	}
	rt.finishBuild(engine, rev, sniRep, nil)
	return nil
}

func (s *Service) BootstrapInstance(ctx context.Context, instanceID uuid.UUID) error {
	return s.bootstrapInstance(ctx, instanceID)
}

func (s *Service) CompileStatusFor(instanceID uuid.UUID) CompileStatusDTO {
	return s.runtimeFor(instanceID).statusForAPI()
}

func (s *Service) requestPublishInstance(instanceID uuid.UUID) {
	select {
	case s.publishWake <- instanceID:
	default:
	}
}

func (s *Service) requestPublishAllInstances(ctx context.Context) {
	ids, err := s.repo.ListPolicyInstanceIDs(ctx)
	if err != nil {
		return
	}
	for _, id := range ids {
		s.requestPublishInstance(id)
	}
}

type instancePolicySource struct {
	repo       repositoryWithPolicy
	instanceID uuid.UUID
}

type repositoryWithPolicy = repository.RulesRepository

func (p instancePolicySource) GetPolicy(ctx context.Context) (domain.Policy, error) {
	return p.repo.GetPolicy(ctx, p.instanceID)
}

func (p instancePolicySource) ListNamedListsByNames(ctx context.Context, names []string) ([]domain.NamedList, error) {
	return p.repo.ListNamedListsByNames(ctx, names)
}
