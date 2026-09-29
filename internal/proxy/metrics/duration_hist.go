package metrics

import "math"

const numLatencyBuckets = 21

// Границы корзин (le) в наносекундах, как Prometheus histogram — кумулятивные верхние пределы.
// Корзина 0: только d == 0; корзина i>0: le[i-1] < d <= le[i].
var latencyLeNs = [numLatencyBuckets]int64{
	0,
	1_000,
	2_000,
	5_000,
	10_000,
	25_000,
	50_000,
	100_000,
	250_000,
	500_000,
	1_000_000,
	2_500_000,
	5_000_000,
	10_000_000,
	25_000_000,
	50_000_000,
	100_000_000,
	250_000_000,
	500_000_000,
	1_000_000_000,
	math.MaxInt64,
}

type durationHist [numLatencyBuckets]uint64

func (h *durationHist) addNs(ns int64) {
	if ns < 0 {
		ns = 0
	}
	h[latencyBucketIndex(ns)]++
}

// quantileUs — оценка перцентиля (Prometheus histogram_quantile: линейная интерполяция в корзине).
func (h *durationHist) quantileUs(q float64, total int64) int64 {
	if total <= 0 || q <= 0 {
		return 0
	}
	if q > 1 {
		q = 1
	}
	rank := q * float64(total)
	var cum float64
	for i, c := range h {
		if c == 0 {
			continue
		}
		count := float64(c)
		cumBefore := cum
		cum += count
		if cum < rank {
			continue
		}
		lower := int64(0)
		if i > 0 {
			lower = latencyLeNs[i-1]
		}
		upper := latencyLeNs[i]
		if upper <= lower {
			return nsToUsCeilP95(upper)
		}
		ratio := (rank - cumBefore) / count
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		estNs := float64(lower) + ratio*(float64(upper)-float64(lower))
		return nsToUsCeilP95(int64(math.Round(estNs)))
	}
	return 0
}

func latencyBucketIndex(ns int64) int {
	if ns <= 0 {
		return 0
	}
	for i := 1; i < len(latencyLeNs); i++ {
		if ns <= latencyLeNs[i] {
			return i
		}
	}
	return len(latencyLeNs) - 1
}

func meanNsToUsExact(sumNs, n int64) float64 {
	if n <= 0 {
		return 0
	}
	return float64(sumNs) / float64(n) / 1000.0
}

// MeanNsToUsExact — среднее sum/count в µs (float).
func MeanNsToUsExact(sumNs, n int64) float64 {
	return meanNsToUsExact(sumNs, n)
}
