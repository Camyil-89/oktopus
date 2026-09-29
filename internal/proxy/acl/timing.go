package acl

import "time"

// EngineTiming — разбивка времени внутри DecideWithRuleRef (только сопоставление правил).
type EngineTiming struct {
	Scope      time.Duration
	MatchSNI   time.Duration
	MatchSrcIP time.Duration
	MatchDstIP time.Duration
	MatchPort  time.Duration
	MatchSlow  time.Duration
}
