package metrics

import "math"
// meanNsToUsRound — средняя в µs (целое, half-up), для breakdown-компонентов.
func meanNsToUsRound(sumNs, n int64) int64 {
	if n <= 0 {
		return 0
	}
	return int64(math.Round(float64(sumNs) / float64(n) / 1000.0))
}

// nsToUsCeilP95 — p95 в наносекундах → µs для UI (не терять суб-микросекундный хвост).
func nsToUsCeilP95(ns int64) int64 {
	if ns <= 0 {
		return 0
	}
	return (ns + 999) / 1000
}

func percentile95UsFromNs(values []int64) int64 {
	return percentileUsFromNs(values, 0.95)
}

// Percentile95UsFromNs — p95 длительностей в наносекундах, результат в µs.
func Percentile95UsFromNs(values []int64) int64 {
	return percentile95UsFromNs(values)
}

// MeanNsToUsRound — средняя в µs (целое, half-up).
func MeanNsToUsRound(sumNs, n int64) int64 {
	return meanNsToUsRound(sumNs, n)
}

func percentileUsFromNs(values []int64, q float64) int64 {
	return nsToUsCeilP95(percentileQuantile(values, q))
}
