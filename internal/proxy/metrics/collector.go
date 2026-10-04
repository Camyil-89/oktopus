package metrics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"oktopus/internal/proxy/bytecount"
)

const window = 5 * time.Minute

const seriesBucketSec = int64(10)
const seriesBucketCount = 30

// TrafficBucket — агрегат запросов за интервал (для графика на дашборде).
type TrafficBucket struct {
	T         int64 `json:"t"`
	Requests  int64 `json:"requests"`
	Allow     int64 `json:"allow"`
	Deny      int64 `json:"deny"`
	BytesUp        int64 `json:"bytes_up"`
	BytesDown      int64 `json:"bytes_down"`
	BytesUpAllow   int64 `json:"bytes_up_allow"`
	BytesUpDeny    int64 `json:"bytes_up_deny"`
	BytesDownAllow int64 `json:"bytes_down_allow"`
	BytesDownDeny  int64 `json:"bytes_down_deny"`
}

// Snapshot — трафик за скользящие 5 минут; avg/p95/p99 длительностей — по последним recentLatencyN решениям.
type Snapshot struct {
	RequestsPerSecAvg5m      float64         `json:"requests_per_sec_avg_5m"`
	RequestsPerSecNow5s      float64         `json:"requests_per_sec_now_5s"`
	Requests1m               int64           `json:"requests_1m"`
	Allowed5m                int64           `json:"allowed_5m"`
	Denied5m                 int64           `json:"denied_5m"`
	DecideDurationUsAvg5m    float64           `json:"decide_duration_us_avg_5m"`
	DecideDurationUsP955m    int64             `json:"decide_duration_us_p95_5m"`
	DecideDurationUsP995m    int64             `json:"decide_duration_us_p99_5m"`
	DecideDurationUsMax5m    int64             `json:"decide_duration_us_max_5m"`
	DecideBreakdown5m        DecideBreakdown5m `json:"decide_breakdown_5m"`
	InspectDurationUsAvg5m   float64           `json:"inspect_duration_us_avg_5m"`
	InspectDurationUsP955m   int64             `json:"inspect_duration_us_p95_5m"`
	InspectDurationUsP995m   int64             `json:"inspect_duration_us_p99_5m"`
	InspectBreakdown5m       InspectBreakdown5m `json:"inspect_breakdown_5m"`
	PolicyDurationUsAvg5m    float64           `json:"policy_duration_us_avg_5m"`
	PolicyDurationUsP955m    int64             `json:"policy_duration_us_p95_5m"`
	PolicyDurationUsP995m    int64             `json:"policy_duration_us_p99_5m"`
	UniqueUsers5m            int             `json:"unique_users_5m"`
	UniqueSources5m          int             `json:"unique_sources_5m"`
	ActiveConnections        int             `json:"active_connections"`
	ActiveWebSocketConnections int           `json:"active_websocket_connections"`
	AccessLogQueuePending    int             `json:"access_log_queue_pending"`
	BytesTotalUpAllow        int64           `json:"bytes_total_up_allow"`
	BytesTotalUpDeny         int64           `json:"bytes_total_up_deny"`
	BytesTotalDownAllow      int64           `json:"bytes_total_down_allow"`
	BytesTotalDownDeny       int64           `json:"bytes_total_down_deny"`
	Buckets10s               []TrafficBucket `json:"buckets_10s"`
}

type secondBucket struct {
	requests     int64
	allow        int64
	deny         int64
	lastUser     string
	lastSourceIP string
	users        map[string]struct{}
	sources      map[string]struct{}
}

// Collector считает метрики на hot path (Observe) и отдаёт снимок для API.
type Collector struct {
	mu           sync.Mutex
	bySec        map[int64]*secondBucket
	bytesBySec   map[int64]byteTotals
	pruneSec     int64
	decideRecent decideRing
	inspectRecent inspectRing
	policyRecent nsRing
}

var defaultCollector = NewCollector()

func NewCollector() *Collector {
	return &Collector{
		bySec:      make(map[int64]*secondBucket),
		bytesBySec: make(map[int64]byteTotals),
	}
}

// RecordDecision регистрирует решение на этом экземпляре (не на process-wide defaultCollector).
func (c *Collector) RecordDecision(allow bool, spend time.Duration, parts DecideParts, user, source string) {
	c.observe(allow, spend, parts, user, source, false)
}

// TrafficSnapshot — снимок метрик этого экземпляра.
func (c *Collector) TrafficSnapshot() Snapshot {
	return c.snapshot()
}

// Percentile95 — 95-й перцентиль (для unit-тестов и утилит).
func Percentile95(values []int64) int64 {
	return percentile95(values)
}

// ObserveDecision регистрирует одно ACL-решение (CONNECT или HTTP).
func ObserveDecision(ctx context.Context, allow bool, spend time.Duration, parts DecideParts, user, source string) {
	collectorFromContext(ctx).observe(allow, spend, parts, user, source, false)
}

// ObserveDecisionWithPolicy — ACL + policy total за один захват mutex (CONNECT hot path).
func ObserveDecisionWithPolicy(ctx context.Context, allow bool, spend time.Duration, parts DecideParts, user, source string) {
	collectorFromContext(ctx).observe(allow, spend, parts, user, source, true)
}

// ObserveInspect регистрирует один проход Lua-инспекции (HTTP после ACL allow).
func ObserveInspect(ctx context.Context, spend time.Duration, parts InspectParts) {
	collectorFromContext(ctx).observeInspect(spend, parts)
}

// ObservePolicyTotal — полное время политики на запрос: ACL (+ inspect на HTTP allow в MITM).
func ObservePolicyTotal(ctx context.Context, spend time.Duration) {
	collectorFromContext(ctx).observePolicy(spend)
}

func (c *Collector) observe(allow bool, spend time.Duration, parts DecideParts, user, source string, withPolicy bool) {
	ns := spend.Nanoseconds()
	if ns < 0 {
		ns = 0
	}
	pns := parts.toNanos()
	now := time.Now()
	sec := now.Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	b := c.bySec[sec]
	if b == nil {
		b = &secondBucket{}
		c.bySec[sec] = b
	}
	b.requests++
	if allow {
		b.allow++
	} else {
		b.deny++
	}
	c.decideRecent.push(decideSample{ns: ns, parts: pns})
	if withPolicy {
		c.policyRecent.push(ns)
	}
	if user != "" && b.lastUser != user {
		b.lastUser = user
		if b.users == nil {
			b.users = make(map[string]struct{})
		}
		b.users[user] = struct{}{}
	}
	if ip := SourceIP(source); ip != "" && b.lastSourceIP != ip {
		b.lastSourceIP = ip
		if b.sources == nil {
			b.sources = make(map[string]struct{})
		}
		b.sources[ip] = struct{}{}
	}

	c.maybePruneLocked(now)
}

func (c *Collector) observeInspect(spend time.Duration, parts InspectParts) {
	ns := spend.Nanoseconds()
	if ns < 0 {
		ns = 0
	}
	pns := parts.toNanos()
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.inspectRecent.push(inspectSample{ns: ns, parts: pns})
	c.maybePruneLocked(now)
}

func (c *Collector) observePolicy(spend time.Duration) {
	ns := spend.Nanoseconds()
	if ns < 0 {
		ns = 0
	}
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.policyRecent.push(ns)
	c.maybePruneLocked(now)
}

func (c *Collector) maybePruneLocked(now time.Time) {
	sec := now.Unix()
	if c.pruneSec == sec {
		return
	}
	c.pruneSec = sec
	c.pruneLocked(now)
}

func (c *Collector) pruneLocked(now time.Time) {
	cutoffSec := now.Add(-window).Unix()
	for k := range c.bySec {
		if k < cutoffSec {
			delete(c.bySec, k)
		}
	}
	for k := range c.bytesBySec {
		if k < cutoffSec {
			delete(c.bytesBySec, k)
		}
	}
}

func (c *Collector) snapshot() Snapshot {
	now := time.Now()
	bytecount.Rotate(now.Unix())
	cutoffSec := now.Add(-window).Unix()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(now)

	var (
		totalReq     int64
		totalAllow   int64
		totalDeny    int64
		last5SecReq  int64
		last60SecReq int64
		users        = make(map[string]struct{})
		sources      = make(map[string]struct{})
	)

	nowSec := now.Unix()
	for sec, b := range c.bySec {
		if sec < cutoffSec {
			continue
		}
		totalReq += b.requests
		totalAllow += b.allow
		totalDeny += b.deny
		if nowSec-sec < 5 {
			last5SecReq += b.requests
		}
		if nowSec-sec < 60 {
			last60SecReq += b.requests
		}
		for u := range b.users {
			users[u] = struct{}{}
		}
		for s := range b.sources {
			sources[s] = struct{}{}
		}
	}

	decideSamples := c.decideRecent.samples()
	var decideNs []int64
	for _, s := range decideSamples {
		decideNs = append(decideNs, s.ns)
	}
	avgUs, decideP95, decideP99, maxUs := statsFromNs(decideNs)
	avgBreakdown := breakdownFromDecideSamples(decideSamples)

	inspectSamples := c.inspectRecent.samples()
	var inspectNs []int64
	for _, s := range inspectSamples {
		inspectNs = append(inspectNs, s.ns)
	}
	avgInspectUs, inspectP95, inspectP99, _ := statsFromNs(inspectNs)
	avgInspectBreakdown := breakdownFromInspectSamples(inspectSamples)

	policyNs := c.policyRecent.values()
	avgPolicyUs, policyP95, policyP99, _ := statsFromNs(policyNs)

	upA, upD, downA, downD := bytecount.Totals()

	return Snapshot{
		RequestsPerSecAvg5m:   float64(totalReq) / window.Seconds(),
		RequestsPerSecNow5s:   float64(last5SecReq) / 5,
		Requests1m:            last60SecReq,
		Allowed5m:             totalAllow,
		Denied5m:              totalDeny,
		DecideDurationUsAvg5m: avgUs,
		DecideDurationUsP955m: decideP95,
		DecideDurationUsP995m: decideP99,
		DecideDurationUsMax5m: maxUs,
		DecideBreakdown5m:     avgBreakdown,
		InspectDurationUsAvg5m: avgInspectUs,
		InspectDurationUsP955m: inspectP95,
		InspectDurationUsP995m: inspectP99,
		InspectBreakdown5m:     avgInspectBreakdown,
		PolicyDurationUsAvg5m:  avgPolicyUs,
		PolicyDurationUsP955m:  policyP95,
		PolicyDurationUsP995m:  policyP99,
		UniqueUsers5m:          len(users),
		UniqueSources5m:       len(sources),
		BytesTotalUpAllow:     int64(upA),
		BytesTotalUpDeny:      int64(upD),
		BytesTotalDownAllow:   int64(downA),
		BytesTotalDownDeny:    int64(downD),
		Buckets10s:            c.buckets10sLocked(now),
	}
}

func (c *Collector) buckets10sLocked(now time.Time) []TrafficBucket {
	nowSec := now.Unix()
	endBucket := nowSec / seriesBucketSec
	startBucket := endBucket - int64(seriesBucketCount-1)

	out := make([]TrafficBucket, seriesBucketCount)
	for i := 0; i < seriesBucketCount; i++ {
		bucketIdx := startBucket + int64(i)
		out[i] = TrafficBucket{T: bucketIdx * seriesBucketSec}
	}

	for sec, b := range c.bySec {
		bucketIdx := sec / seriesBucketSec
		if bucketIdx < startBucket || bucketIdx > endBucket {
			continue
		}
		i := int(bucketIdx - startBucket)
		if i < 0 || i >= seriesBucketCount {
			continue
		}
		out[i].Requests += b.requests
		out[i].Allow += b.allow
		out[i].Deny += b.deny
	}
	c.mergeBytesIntoBucketsLocked(now, out)
	return out
}

func percentile95(values []int64) int64 {
	return percentileQuantile(values, 0.95)
}

func percentileQuantile(values []int64, q float64) int64 {
	if len(values) == 0 || q <= 0 {
		return 0
	}
	if q > 1 {
		q = 1
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	n := len(sorted)
	idx := int(math.Ceil(q*float64(n))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
