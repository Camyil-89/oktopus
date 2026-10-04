package inspect

import (
	"context"
	stdhttp "net/http"
	"time"

	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/proxy/observe"
)

// HTTPMiddleware — Lua-инспекция только после ACL allow (см. observe.ACLAllowPassed).
// enabled — false в режиме tunnel (без MITM исходящий HTTPS не разбирается).
func HTTPMiddleware(runner *Runner, rec accesslog.Recorder, enabled bool) hooks.HTTPRequestMiddleware {
	if !enabled {
		return nil
	}
	return func(ctx context.Context, req *stdhttp.Request) hooks.Decision {
		if req == nil || !observe.ACLAllowPassed(req.Context()) {
			return hooks.AllowDecision()
		}
		aclDec, _ := observe.ACLDecisionFrom(req.Context())

		record := func(out accesslog.InspectHTTPOutcome) {
			if rec == nil {
				return
			}
			if runner == nil || runner.Empty() {
				rec.Record(accesslog.FinalHTTPEntryAfterACL(ctx, req, aclDec.RuleRef, aclDec.Spend))
				return
			}
			rec.Record(accesslog.FinalHTTPEntryAfterInspect(ctx, req, aclDec.RuleRef, aclDec.Spend, out))
		}

		start := time.Now()
		if runner == nil || runner.Empty() {
			metrics.ObserveInspect(ctx, 0, metrics.InspectParts{})
			metrics.ObservePolicyTotal(ctx, aclDec.Spend)
			record(accesslog.InspectHTTPOutcome{})
			return hooks.AllowDecision()
		}

		hookCtx := ctx
		if req != nil {
			if c := req.Context(); c != nil {
				hookCtx = c
			}
		}
		rc := RequestContextFromHTTP(hookCtx, req)
		prepareEnd := time.Now()
		AttachBodyMeta(&rc, req)
		bodyEnd := time.Now()
		res, err := runner.Eval(rc)
		evalEnd := time.Now()

		inspectSpend := evalEnd.Sub(start)
		parts := metrics.InspectParts{
			Prepare: prepareEnd.Sub(start),
			Body:    bodyEnd.Sub(prepareEnd),
			Eval:    evalEnd.Sub(bodyEnd),
		}
		metrics.ObserveInspect(hookCtx, inspectSpend, parts)
		metrics.ObservePolicyTotal(hookCtx, aclDec.Spend+inspectSpend)

		out := accesslog.InspectHTTPOutcome{
			Spend:    inspectSpend,
			Matched:  res.Matched,
			Deny:     res.Deny,
			RuleID:   res.RuleID,
			Err:      err,
			RuleLogs: res.RuleLogs,
		}
		record(out)

		if err != nil {
			return hooks.DenyDecision()
		}
		if res.Matched && res.Deny {
			return hooks.DenyDecision()
		}
		return hooks.AllowDecision()
	}
}
