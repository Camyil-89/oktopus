package metrics

import "time"

// InspectParts — длительности одного прохода inspect middleware (исходящий HTTP после ACL allow).
type InspectParts struct {
	Prepare time.Duration
	Body    time.Duration
	Eval    time.Duration
}

func (p InspectParts) toNanos() inspectPartsNs {
	return inspectPartsNs{
		prepare: p.Prepare.Nanoseconds(),
		body:    p.Body.Nanoseconds(),
		eval:    p.Eval.Nanoseconds(),
	}
}

type inspectPartsNs struct {
	prepare int64
	body    int64
	eval    int64
}

func (p inspectPartsNs) add(dst *inspectPartsNs) {
	dst.prepare += p.prepare
	dst.body += p.body
	dst.eval += p.eval
}

// InspectBreakdown5m — средние и p95 по компонентам inspect (последние recentLatencyN проходов).
type InspectBreakdown5m struct {
	PrepareUsAvg int64 `json:"prepare_us_avg_5m"`
	BodyUsAvg    int64 `json:"body_us_avg_5m"`
	EvalUsAvg    int64 `json:"eval_us_avg_5m"`

	PrepareUsP95 int64 `json:"prepare_us_p95_5m,omitempty"`
	BodyUsP95    int64 `json:"body_us_p95_5m,omitempty"`
	EvalUsP95    int64 `json:"eval_us_p95_5m,omitempty"`
}

type inspectSample struct {
	ns    int64
	parts inspectPartsNs
}

func inspectBreakdownFromSums(sum inspectPartsNs, n int64) InspectBreakdown5m {
	if n <= 0 {
		return InspectBreakdown5m{}
	}
	return InspectBreakdown5m{
		PrepareUsAvg: meanNsToUsRound(sum.prepare, n),
		BodyUsAvg:    meanNsToUsRound(sum.body, n),
		EvalUsAvg:    meanNsToUsRound(sum.eval, n),
	}
}

func inspectBreakdownP95(samples []inspectSample) InspectBreakdown5m {
	if len(samples) == 0 {
		return InspectBreakdown5m{}
	}
	var prepare, body, eval []int64
	for _, s := range samples {
		prepare = append(prepare, s.parts.prepare)
		body = append(body, s.parts.body)
		eval = append(eval, s.parts.eval)
	}
	return InspectBreakdown5m{
		PrepareUsP95: percentile95UsFromNs(prepare),
		BodyUsP95:    percentile95UsFromNs(body),
		EvalUsP95:    percentile95UsFromNs(eval),
	}
}
