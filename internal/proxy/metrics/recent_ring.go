package metrics

const recentLatencyN = 1000

type nsRing struct {
	data []int64
	i    int
	n    int
}

func (r *nsRing) push(v int64) {
	if r.data == nil {
		r.data = make([]int64, recentLatencyN)
	}
	r.data[r.i] = v
	r.i = (r.i + 1) % recentLatencyN
	if r.n < recentLatencyN {
		r.n++
	}
}

func (r *nsRing) values() []int64 {
	if r.n == 0 {
		return nil
	}
	if r.n < recentLatencyN {
		out := make([]int64, r.n)
		copy(out, r.data[:r.n])
		return out
	}
	out := make([]int64, recentLatencyN)
	copy(out, r.data[r.i:])
	copy(out[recentLatencyN-r.i:], r.data[:r.i])
	return out
}

type decideRing struct {
	data []decideSample
	i    int
	n    int
}

func (r *decideRing) push(s decideSample) {
	if r.data == nil {
		r.data = make([]decideSample, recentLatencyN)
	}
	r.data[r.i] = s
	r.i = (r.i + 1) % recentLatencyN
	if r.n < recentLatencyN {
		r.n++
	}
}

func (r *decideRing) samples() []decideSample {
	if r.n == 0 {
		return nil
	}
	if r.n < recentLatencyN {
		out := make([]decideSample, r.n)
		copy(out, r.data[:r.n])
		return out
	}
	out := make([]decideSample, recentLatencyN)
	copy(out, r.data[r.i:])
	copy(out[recentLatencyN-r.i:], r.data[:r.i])
	return out
}

type inspectRing struct {
	data []inspectSample
	i    int
	n    int
}

func (r *inspectRing) push(s inspectSample) {
	if r.data == nil {
		r.data = make([]inspectSample, recentLatencyN)
	}
	r.data[r.i] = s
	r.i = (r.i + 1) % recentLatencyN
	if r.n < recentLatencyN {
		r.n++
	}
}

func (r *inspectRing) samples() []inspectSample {
	if r.n == 0 {
		return nil
	}
	if r.n < recentLatencyN {
		out := make([]inspectSample, r.n)
		copy(out, r.data[:r.n])
		return out
	}
	out := make([]inspectSample, recentLatencyN)
	copy(out, r.data[r.i:])
	copy(out[recentLatencyN-r.i:], r.data[:r.i])
	return out
}

func statsFromNs(vals []int64) (avg float64, p95, p99, maxUs int64) {
	if len(vals) == 0 {
		return 0, 0, 0, 0
	}
	var sum int64
	var maxNs int64
	for _, v := range vals {
		sum += v
		if v > maxNs {
			maxNs = v
		}
	}
	n := int64(len(vals))
	return meanNsToUsExact(sum, n), percentileUsFromNs(vals, 0.95), percentileUsFromNs(vals, 0.99), nsToUsCeilP95(maxNs)
}

func breakdownFromDecideSamples(samples []decideSample) DecideBreakdown5m {
	if len(samples) == 0 {
		return DecideBreakdown5m{}
	}
	var sum decidePartsNs
	for _, s := range samples {
		s.parts.add(&sum)
	}
	n := int64(len(samples))
	bd := breakdownFromSums(sum, n)
	p95 := breakdownP95(samples)
	bd.PrepareUsP95 = p95.PrepareUsP95
	bd.EngineUsP95 = p95.EngineUsP95
	bd.ScopeUsP95 = p95.ScopeUsP95
	bd.MatchSNIUsP95 = p95.MatchSNIUsP95
	bd.MatchSrcIPUsP95 = p95.MatchSrcIPUsP95
	bd.MatchDstIPUsP95 = p95.MatchDstIPUsP95
	bd.MatchSlowUsP95 = p95.MatchSlowUsP95
	return bd
}

func breakdownFromInspectSamples(samples []inspectSample) InspectBreakdown5m {
	if len(samples) == 0 {
		return InspectBreakdown5m{}
	}
	var sum inspectPartsNs
	for _, s := range samples {
		s.parts.add(&sum)
	}
	n := int64(len(samples))
	bd := inspectBreakdownFromSums(sum, n)
	p95 := inspectBreakdownP95(samples)
	bd.PrepareUsP95 = p95.PrepareUsP95
	bd.BodyUsP95 = p95.BodyUsP95
	bd.EvalUsP95 = p95.EvalUsP95
	return bd
}
