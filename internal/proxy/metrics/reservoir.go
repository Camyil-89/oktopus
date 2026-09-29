package metrics

import "math/rand/v2"

// ReservoirAppend сохраняет до cap элементов равномерной выборкой из потока (streamIndex — число элементов в потоке, с 1).
func ReservoirAppend[T any](s []T, v T, cap int, streamIndex int64) []T {
	return reservoirAppend(s, v, cap, streamIndex)
}

func reservoirAppend[T any](s []T, v T, cap int, streamIndex int64) []T {
	if cap <= 0 {
		return s
	}
	if streamIndex <= int64(cap) {
		return append(s, v)
	}
	j := rand.Int64N(streamIndex)
	if j < int64(cap) {
		s[j] = v
	}
	return s
}
