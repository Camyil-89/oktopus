package repository

// Segment журнала: traffic — без атак и ошибок; attacks — policy_anomaly; errors — системные/шлюз.
const (
	SegmentTraffic = "traffic"
	SegmentAttacks = "attacks"
	SegmentErrors  = "errors"
)

func ValidAccessLogSegment(s string) bool {
	switch s {
	case "", SegmentTraffic, SegmentAttacks, SegmentErrors:
		return true
	default:
		return false
	}
}
