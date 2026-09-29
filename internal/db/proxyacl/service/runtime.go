package service

import (
	"sync"
	"time"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
)

const (
	CompileIdle     = "idle"
	CompileBuilding = "building"
	CompileReady    = "ready"
	CompileError    = "error"
)

// CompileStatusDTO — состояние сборки ACL для API.
type CompileStatusDTO struct {
	BuildStatus              string `json:"build_status"`
	BuildError               string `json:"build_error,omitempty"`
	BuildStartedAt           string `json:"build_started_at,omitempty"`
	BuildFinishedAt          string `json:"build_finished_at,omitempty"`
	LastCompileDurationMs    *int64 `json:"last_compile_duration_ms,omitempty"`
	ConfigRevision           string `json:"config_revision"`
	ActiveRevision           string `json:"active_revision"`
	RulesInSync              bool   `json:"rules_in_sync"`
	ActiveLogicalRules       int    `json:"active_logical_rules"`
	ActivePatterns           int    `json:"active_patterns"`
	ActiveSNIPatternsIndexed int    `json:"active_sni_patterns_indexed"`
	ActiveSNIPatternsRegexp  int    `json:"active_sni_patterns_regexp"`
	ActiveSlowRuleSlots      int    `json:"active_slow_rule_slots"`
	SNIRegexpReasonCounts    map[string]int `json:"sni_regexp_reason_counts,omitempty"`
	SNIRegexpSamples         []acl.SNIPatternRegexpSample `json:"sni_regexp_samples,omitempty"`
	EnabledRulesInDB         int `json:"enabled_rules_in_db"`
	CompileDiagnostics       []squid.Diagnostic `json:"compile_diagnostics,omitempty"`
}

type activeSnapshot struct {
	engine    *acl.Engine
	revision  string
	sniReport acl.SNIPatternIndexReport
}

type compileRuntime struct {
	mu sync.RWMutex

	configRevision string
	active         activeSnapshot

	buildStatus      string
	buildError       string
	buildStartedAt   time.Time
	buildFinishedAt       time.Time
	lastCompileDurationMs int64
	enabledRulesInDB      int
	lastDiagnostics       []squid.Diagnostic
}

func newCompileRuntime() *compileRuntime {
	return &compileRuntime{
		buildStatus: CompileIdle,
		active: activeSnapshot{
			engine:   acl.EmptyEngine(),
			revision: "",
		},
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
	return r.active.revision
}

func (r *compileRuntime) beginBuild() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buildStatus = CompileBuilding
	r.buildError = ""
	r.buildStartedAt = time.Now().UTC()
	r.buildFinishedAt = time.Time{}
}

func (r *compileRuntime) setEnabledRulesInDB(n int) {
	r.mu.Lock()
	r.enabledRulesInDB = n
	r.mu.Unlock()
}

func (r *compileRuntime) finishBuild(engine *acl.Engine, revision string, sniReport acl.SNIPatternIndexReport, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buildFinishedAt = time.Now().UTC()
	if !r.buildStartedAt.IsZero() {
		r.lastCompileDurationMs = r.buildFinishedAt.Sub(r.buildStartedAt).Milliseconds()
	}
	if err != nil {
		r.buildStatus = CompileError
		r.buildError = err.Error()
		r.lastDiagnostics = diagnosticsFromCompileErr(err)
		return
	}
	r.lastDiagnostics = nil
	if engine == nil {
		engine = acl.EmptyEngine()
	}
	r.active = activeSnapshot{
		engine:    engine,
		revision:  revision,
		sniReport: sniReport,
	}
	r.buildStatus = CompileReady
	r.buildError = ""
}

func (r *compileRuntime) markReadyIfSynced() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.configRevision != "" && r.configRevision == r.active.revision {
		r.buildStatus = CompileReady
		r.buildError = ""
	}
}

// isPublished — policy и engine совпали и последняя сборка без ошибки.
func (r *compileRuntime) isPublished() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.buildStatus == CompileReady &&
		r.configRevision == r.active.revision
}

func (r *compileRuntime) activeEngine() *acl.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active.engine == nil {
		return acl.EmptyEngine()
	}
	return r.active.engine
}

func compileDurationMs(started, finished time.Time) *int64 {
	if started.IsZero() || finished.IsZero() {
		return nil
	}
	d := finished.Sub(started)
	if d < 0 {
		d = 0
	}
	ms := d.Milliseconds()
	if ms == 0 && d > 0 {
		ms = 1
	}
	return &ms
}

func (r *compileRuntime) statusForAPI() CompileStatusDTO {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dto := CompileStatusDTO{
		BuildStatus:              r.buildStatus,
		BuildError:               r.buildError,
		ConfigRevision:           r.configRevision,
		ActiveRevision:           r.active.revision,
		RulesInSync:              r.configRevision != "" && r.configRevision == r.active.revision && r.buildStatus == CompileReady,
		ActiveLogicalRules:       r.active.engine.LogicalRuleCount(),
		ActivePatterns:           r.active.engine.PatternCount(),
		ActiveSNIPatternsIndexed: r.active.engine.SNIPatternStats().Indexed,
		ActiveSNIPatternsRegexp:  r.active.engine.SNIPatternStats().Regexp,
		ActiveSlowRuleSlots:      r.active.engine.SlowRuleSlotCount(),
		EnabledRulesInDB:         r.enabledRulesInDB,
		CompileDiagnostics:       r.lastDiagnostics,
	}
	if len(r.active.sniReport.RegexpReasonCounts) > 0 {
		dto.SNIRegexpReasonCounts = r.active.sniReport.RegexpReasonCounts
	}
	if len(r.active.sniReport.RegexpSamples) > 0 {
		dto.SNIRegexpSamples = r.active.sniReport.RegexpSamples
	}
	if !r.buildStartedAt.IsZero() {
		dto.BuildStartedAt = r.buildStartedAt.UTC().Format(time.RFC3339)
	}
	if !r.buildFinishedAt.IsZero() {
		dto.BuildFinishedAt = r.buildFinishedAt.UTC().Format(time.RFC3339)
		if ms := compileDurationMs(r.buildStartedAt, r.buildFinishedAt); ms != nil {
			dto.LastCompileDurationMs = ms
		}
	}
	if r.buildStatus == CompileIdle && r.configRevision == "" {
		dto.BuildStatus = CompileReady
	}
	return dto
}
