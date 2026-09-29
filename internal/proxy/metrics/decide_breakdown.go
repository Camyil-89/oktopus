package metrics

import "time"

// DecideParts — длительности одного ACL-решения (hot path hooks → engine).
type DecideParts struct {
	Prepare    time.Duration
	Engine     time.Duration
	Scope      time.Duration
	MatchSNI   time.Duration
	MatchSrcIP time.Duration
	MatchDstIP time.Duration
	MatchSlow  time.Duration
}

func (p DecideParts) toNanos() decidePartsNs {
	return decidePartsNs{
		prepare:    p.Prepare.Nanoseconds(),
		engine:     p.Engine.Nanoseconds(),
		scope:      p.Scope.Nanoseconds(),
		matchSNI:   p.MatchSNI.Nanoseconds(),
		matchSrcIP: p.MatchSrcIP.Nanoseconds(),
		matchDstIP: p.MatchDstIP.Nanoseconds(),
		matchSlow:  p.MatchSlow.Nanoseconds(),
	}
}

type decidePartsNs struct {
	prepare    int64
	engine     int64
	scope      int64
	matchSNI   int64
	matchSrcIP int64
	matchDstIP int64
	matchSlow  int64
}

func (p decidePartsNs) add(dst *decidePartsNs) {
	dst.prepare += p.prepare
	dst.engine += p.engine
	dst.scope += p.scope
	dst.matchSNI += p.matchSNI
	dst.matchSrcIP += p.matchSrcIP
	dst.matchDstIP += p.matchDstIP
	dst.matchSlow += p.matchSlow
}

// DecideBreakdown5m — средние и p95 по компонентам ACL (последние recentLatencyN решений).
type DecideBreakdown5m struct {
	PrepareUsAvg    int64 `json:"prepare_us_avg_5m"`
	EngineUsAvg     int64 `json:"engine_us_avg_5m"`
	ScopeUsAvg      int64 `json:"scope_us_avg_5m"`
	MatchSNIUsAvg   int64 `json:"match_sni_us_avg_5m"`
	MatchSrcIPUsAvg int64 `json:"match_src_ip_us_avg_5m"`
	MatchDstIPUsAvg int64 `json:"match_dst_ip_us_avg_5m"`
	MatchSlowUsAvg  int64 `json:"match_slow_us_avg_5m"`

	PrepareUsP95    int64 `json:"prepare_us_p95_5m,omitempty"`
	EngineUsP95     int64 `json:"engine_us_p95_5m,omitempty"`
	ScopeUsP95      int64 `json:"scope_us_p95_5m,omitempty"`
	MatchSNIUsP95   int64 `json:"match_sni_us_p95_5m,omitempty"`
	MatchSrcIPUsP95 int64 `json:"match_src_ip_us_p95_5m,omitempty"`
	MatchDstIPUsP95 int64 `json:"match_dst_ip_us_p95_5m,omitempty"`
	MatchSlowUsP95  int64 `json:"match_slow_us_p95_5m,omitempty"`
}

type decideSample struct {
	ns    int64
	parts decidePartsNs
}

func breakdownFromSums(sum decidePartsNs, n int64) DecideBreakdown5m {
	if n <= 0 {
		return DecideBreakdown5m{}
	}
	return DecideBreakdown5m{
		PrepareUsAvg:    meanNsToUsRound(sum.prepare, n),
		EngineUsAvg:     meanNsToUsRound(sum.engine, n),
		ScopeUsAvg:      meanNsToUsRound(sum.scope, n),
		MatchSNIUsAvg:   meanNsToUsRound(sum.matchSNI, n),
		MatchSrcIPUsAvg: meanNsToUsRound(sum.matchSrcIP, n),
		MatchDstIPUsAvg: meanNsToUsRound(sum.matchDstIP, n),
		MatchSlowUsAvg:  meanNsToUsRound(sum.matchSlow, n),
	}
}

func breakdownP95(samples []decideSample) DecideBreakdown5m {
	if len(samples) == 0 {
		return DecideBreakdown5m{}
	}
	var (
		prepare    []int64
		engine     []int64
		scope      []int64
		matchSNI   []int64
		matchSrcIP []int64
		matchDstIP []int64
		matchSlow  []int64
	)
	for _, s := range samples {
		prepare = append(prepare, s.parts.prepare)
		engine = append(engine, s.parts.engine)
		scope = append(scope, s.parts.scope)
		matchSNI = append(matchSNI, s.parts.matchSNI)
		matchSrcIP = append(matchSrcIP, s.parts.matchSrcIP)
		matchDstIP = append(matchDstIP, s.parts.matchDstIP)
		matchSlow = append(matchSlow, s.parts.matchSlow)
	}
	return DecideBreakdown5m{
		PrepareUsP95:    percentile95UsFromNs(prepare),
		EngineUsP95:     percentile95UsFromNs(engine),
		ScopeUsP95:      percentile95UsFromNs(scope),
		MatchSNIUsP95:   percentile95UsFromNs(matchSNI),
		MatchSrcIPUsP95: percentile95UsFromNs(matchSrcIP),
		MatchDstIPUsP95: percentile95UsFromNs(matchDstIP),
		MatchSlowUsP95:  percentile95UsFromNs(matchSlow),
	}
}
