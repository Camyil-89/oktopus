package acl

// HTTPAccessCount — число строк http_access в engine.
func (e *Engine) HTTPAccessCount() int {
	if e == nil {
		return 0
	}
	return len(e.squidAccess)
}

// LogicalRuleCount — число строк http_access (логических шагов решения).
func (e *Engine) LogicalRuleCount() int {
	return e.HTTPAccessCount()
}

// PatternCount — число скомпилированных паттернов в acl matchers.
func (e *Engine) PatternCount() int {
	if e == nil {
		return 0
	}
	return e.squidPatternCount()
}

func (e *Engine) squidPatternCount() int {
	n := 0
	for _, line := range e.squidAccess {
		for _, cl := range line.Clauses {
			n += squidMatcherPatternCount(cl.Matcher)
		}
	}
	return n
}

func squidMatcherPatternCount(m SquidMatcher) int {
	switch x := m.(type) {
	case SquidMatchPatterns:
		return x.patternCount()
	default:
		return 0
	}
}

// SNIPatternStats — сколько SNI-строк попало в индекс vs regexp slow path.
type SNIPatternStats struct {
	Indexed int
	Regexp  int
}

// SNIPatternStats возвращает агрегат по dstdomain acl.
func (e *Engine) SNIPatternStats() SNIPatternStats {
	if e == nil {
		return SNIPatternStats{}
	}
	return e.squidSNIPatternStats()
}

func (e *Engine) squidSNIPatternStats() SNIPatternStats {
	var st SNIPatternStats
	for _, line := range e.squidAccess {
		for _, cl := range line.Clauses {
			if sp, ok := cl.Matcher.(SquidMatchPatterns); ok && sp.RuleType == RuleSNI {
				s := sp.SquidSNIPatternStats()
				st.Indexed += s.Indexed
				st.Regexp += s.Regexp
			}
		}
	}
	return st
}

// SlowRuleSlotCount — число dstdomain acl с regexp slow path.
func (e *Engine) SlowRuleSlotCount() int {
	if e == nil {
		return 0
	}
	return e.squidSlowACLCount()
}

func (e *Engine) squidSlowACLCount() int {
	n := 0
	for _, line := range e.squidAccess {
		for _, cl := range line.Clauses {
			if sp, ok := cl.Matcher.(SquidMatchPatterns); ok && sp.SquidSlowPatternCount() > 0 {
				n++
			}
		}
	}
	return n
}
