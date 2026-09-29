package ratelimit

type rateLimits struct {
	restoreBPS int64
	maxBytes   int64
	initial    int64
}

func (l rateLimits) unlimited() bool {
	return l.restoreBPS < 0 || l.maxBytes < 0
}

// PoolDef — параметры delay pool (Squid delay_class / delay_parameters).
type PoolDef struct {
	Class int // 1, 2 или 3
	// Aggregate (class 2/3): общий bucket; none/-1 = без лимита.
	AggRestoreBPS int64
	AggMaxBytes   int64
	// Individual: per-IP (class 1/2) или per-host в сети (class 3).
	IndRestoreBPS int64
	IndMaxBytes   int64
	Initial       int64 // delay_initial_bucket_size (individual burst)
}

func (d PoolDef) AggLimited() bool {
	return !rateLimits{d.AggRestoreBPS, d.AggMaxBytes, 0}.unlimited()
}

func (d PoolDef) IndLimited() bool {
	return !rateLimits{d.IndRestoreBPS, d.IndMaxBytes, d.initialInd()}.unlimited()
}

func (d PoolDef) initialInd() int64 {
	if d.Initial > 0 {
		return d.Initial
	}
	if d.IndMaxBytes > 0 {
		return d.IndMaxBytes / 2
	}
	return 0
}

func (d PoolDef) aggLimits() rateLimits {
	return rateLimits{d.AggRestoreBPS, d.AggMaxBytes, 0}
}

func (d PoolDef) indLimits() rateLimits {
	return rateLimits{d.IndRestoreBPS, d.IndMaxBytes, d.initialInd()}
}

// Unlimited — pool не ограничивает трафик.
func (d PoolDef) Unlimited() bool {
	switch d.Class {
	case 2, 3:
		return !d.AggLimited() && !d.IndLimited()
	default:
		return !d.IndLimited()
	}
}
