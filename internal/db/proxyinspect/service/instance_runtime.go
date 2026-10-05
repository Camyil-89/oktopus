package service

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/proxy/inspect"
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

func (s *Service) ActiveRunner(instanceID uuid.UUID) *inspect.Runner {
	return s.runtimeFor(instanceID).activeRunner()
}

func (s *Service) BootstrapInstance(ctx context.Context, instanceID uuid.UUID) error {
	rules, err := s.repo.ListByInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	rt := s.runtimeFor(instanceID)
	rev := revisionFromRules(rules)
	rt.setConfigRevision(rev)
	runner, err := compileRunner(rules)
	if err != nil {
		rt.finishBuild(nil, rev, err)
		return err
	}
	rt.finishBuild(runner, rev, nil)
	return nil
}

func (s *Service) requestPublishInstance(instanceID uuid.UUID) {
	select {
	case s.publishWake <- instanceID:
	default:
	}
}
