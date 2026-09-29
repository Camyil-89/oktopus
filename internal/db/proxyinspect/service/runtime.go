package service

import (
	"sync"
	"time"

	"oktopus/internal/proxy/inspect"
)

type compileRuntime struct {
	mu sync.RWMutex

	configRevision string
	published      *inspect.Runner
	activeRevision string

	buildStatus     string
	buildError      string
	buildStartedAt  time.Time
	buildFinishedAt time.Time
	activeRuleCount int
}

func newCompileRuntime() *compileRuntime {
	return &compileRuntime{
		buildStatus: CompileReady,
		published:   inspect.NewRunner(&inspect.Program{}),
	}
}

func (r *compileRuntime) setConfigRevision(rev string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configRevision = rev
}

func (r *compileRuntime) getConfigRevision() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configRevision
}

func (r *compileRuntime) getActiveRevision() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeRevision
}

func (r *compileRuntime) beginBuild() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buildStatus = CompileBuilding
	r.buildError = ""
	r.buildStartedAt = time.Now().UTC()
	r.buildFinishedAt = time.Time{}
}

func (r *compileRuntime) finishBuild(runner *inspect.Runner, revision string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buildFinishedAt = time.Now().UTC()
	if err != nil {
		r.buildStatus = CompileError
		r.buildError = err.Error()
		return
	}
	if runner == nil {
		runner = inspect.NewRunner(&inspect.Program{})
	}
	r.published = runner
	r.activeRevision = revision
	if runner != nil {
		r.activeRuleCount = runner.RuleCount()
	} else {
		r.activeRuleCount = 0
	}
	r.buildStatus = CompileReady
	r.buildError = ""
}

func (r *compileRuntime) setReloadError(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buildStatus = CompileError
	r.buildError = msg
}

func (r *compileRuntime) activeRunner() *inspect.Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.published == nil {
		return inspect.NewRunner(&inspect.Program{})
	}
	return r.published
}

func (r *compileRuntime) status(enabledInDB int) CompileStatusDTO {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dto := CompileStatusDTO{
		BuildStatus:      r.buildStatus,
		BuildError:       r.buildError,
		ConfigRevision:   r.configRevision,
		ActiveRevision:   r.activeRevision,
		RulesInSync:      r.configRevision != "" && r.configRevision == r.activeRevision && r.buildStatus == CompileReady,
		ActiveRules:      r.activeRuleCount,
		EnabledRulesInDB: enabledInDB,
	}
	if !r.buildStartedAt.IsZero() {
		dto.BuildStartedAt = r.buildStartedAt.UTC().Format(time.RFC3339)
	}
	if !r.buildFinishedAt.IsZero() {
		dto.BuildFinishedAt = r.buildFinishedAt.UTC().Format(time.RFC3339)
	}
	if r.buildStatus == CompileIdle && r.configRevision == "" {
		dto.BuildStatus = CompileReady
	}
	return dto
}
