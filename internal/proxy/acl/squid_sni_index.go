package acl

import "fmt"

// SNIIndexCompileAcc — накопитель причин regexp при сборке squid dstdomain.
type SNIIndexCompileAcc struct {
	acc *sniIndexCompileAcc
}

// NewSNIIndexCompileAcc создаёт аккумулятор для отчёта SNI index vs regexp.
func NewSNIIndexCompileAcc() *SNIIndexCompileAcc {
	return &SNIIndexCompileAcc{acc: newSNIIndexCompileAcc()}
}

// Report возвращает итог компиляции SNI-строк.
func (a *SNIIndexCompileAcc) Report() SNIPatternIndexReport {
	if a == nil || a.acc == nil {
		return SNIPatternIndexReport{}
	}
	return a.acc.report()
}

// CompileSquidPatternLine компилирует строку шаблона для squid acl с учётом отчёта.
func CompileSquidPatternLine(rt RuleType, line string, acc *SNIIndexCompileAcc) (CompiledPatternPublic, error) {
	var inner *sniIndexCompileAcc
	if acc != nil {
		inner = acc.acc
	}
	cp, err := compilePattern(rt, line, inner)
	if err != nil {
		return CompiledPatternPublic{}, err
	}
	return CompiledPatternPublic{inner: cp, typ: rt}, nil
}

const squidSNIMemberRule = 0

func newSNIMembershipIndex() *sniIndex {
	return &sniIndex{
		exact: make(map[string]int),
		root:  domainNode{suffixRule: -1, wildRule: -1},
	}
}

func insertSNIMembershipPattern(idx **sniIndex, p compiledPattern) {
	if p.sniFast == sniFastNone {
		return
	}
	if *idx == nil {
		*idx = newSNIMembershipIndex()
	}
	switch p.sniFast {
	case sniFastExact:
		setEarliest((*idx).exact, p.sniHost, squidSNIMemberRule)
	case sniFastSuffix:
		(*idx).insertSuffix(p.sniHost, squidSNIMemberRule)
	case sniFastWildcardSuffix:
		(*idx).insertWildcard(p.sniHost, squidSNIMemberRule)
	}
}

// CompileSquidSNIMatcher собирает dstdomain acl потоково, без слайса всех паттернов.
func CompileSquidSNIMatcher(eachLine func(func(string) error) error, acc *SNIIndexCompileAcc) (SquidMatchPatterns, error) {
	var idx *sniIndex
	var slow []compiledPattern
	total, indexed := 0, 0
	lineNo := 0
	err := eachLine(func(line string) error {
		lineNo++
		total++
		cp, err := compilePattern(RuleSNI, line, accInner(acc))
		if err != nil {
			return fmt.Errorf("строка %d: %w", lineNo, err)
		}
		if cp.sniFast == sniFastNone {
			slow = append(slow, cp)
			return nil
		}
		indexed++
		insertSNIMembershipPattern(&idx, cp)
		return nil
	})
	if err != nil {
		return SquidMatchPatterns{}, err
	}
	if total == 0 {
		return SquidMatchPatterns{}, fmt.Errorf("нет значений")
	}
	return SquidMatchPatterns{
		RuleType:     RuleSNI,
		sniIndex:     idx,
		slowPatterns: slow,
		patternTotal: total,
		sniIndexed:   indexed,
		sniRegexp:    len(slow),
	}, nil
}

func accInner(acc *SNIIndexCompileAcc) *sniIndexCompileAcc {
	if acc == nil {
		return nil
	}
	return acc.acc
}

// buildSNIMembershipIndex — trie/map для OR по доменным литералам (все совпадения → memberRule).
func buildSNIMembershipIndex(patterns []compiledPattern) (*sniIndex, []compiledPattern) {
	var idx *sniIndex
	var slow []compiledPattern
	for _, p := range patterns {
		if p.sniFast == sniFastNone {
			slow = append(slow, p)
			continue
		}
		insertSNIMembershipPattern(&idx, p)
	}
	return idx, slow
}

func finalizeSquidSNIMatchPatterns(m *SquidMatchPatterns) {
	if m == nil || m.RuleType != RuleSNI {
		return
	}
	if m.patternTotal > 0 && m.sniIndex != nil && len(m.Patterns) == 0 {
		return
	}
	inners := make([]compiledPattern, 0, len(m.Patterns))
	for _, p := range m.Patterns {
		inners = append(inners, p.inner)
	}
	m.sniIndex, m.slowPatterns = buildSNIMembershipIndex(inners)
	m.patternTotal = len(inners)
	m.sniIndexed = 0
	m.sniRegexp = len(m.slowPatterns)
	for _, p := range inners {
		if p.sniFast != sniFastNone {
			m.sniIndexed++
		}
	}
	m.Patterns = nil
}

func (m *SquidMatchPatterns) recomputeSNICounters() {
	if m.RuleType != RuleSNI || m.patternTotal > 0 {
		return
	}
	m.sniIndexed = 0
	m.sniRegexp = 0
	for _, p := range m.Patterns {
		if p.inner.sniFast != sniFastNone {
			m.sniIndexed++
		} else if p.inner.re != nil {
			m.sniRegexp++
		}
	}
}

// SquidSNIPatternStats — indexed / regexp по одному dstdomain acl.
func (m *SquidMatchPatterns) SquidSNIPatternStats() SNIPatternStats {
	if m == nil || m.RuleType != RuleSNI {
		return SNIPatternStats{}
	}
	return SNIPatternStats{Indexed: m.sniIndexed, Regexp: m.sniRegexp}
}

// SquidSlowPatternCount — число regexp-строк в dstdomain acl (slow path).
func (m *SquidMatchPatterns) SquidSlowPatternCount() int {
	if m == nil || m.RuleType != RuleSNI {
		return 0
	}
	return len(m.slowPatterns)
}
