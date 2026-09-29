package squid

import (
	"fmt"
	"strconv"
	"strings"

	"oktopus/internal/proxy/ratelimit"
)

// DelaySection — директивы delay_pools из конфига.
type DelaySection struct {
	PoolCount            int
	Class                map[int]int // pool -> 1|2|3
	Parameters           map[int]DelayParams
	InitialBucket        map[int]int64
	Access               []DelayAccessLine
}

// DelayParams — delay_parameters (aggregate + individual, как в Squid).
type DelayParams struct {
	LineNo      int
	AggRestore  int64
	AggMax      int64
	IndRestore  int64
	IndMax      int64
}

// DelayAccessLine — delay_access pool allow|deny …
type DelayAccessLine struct {
	LineNo  int
	Pool    int
	Allow   bool
	ACLs    []AccessClause
}

func parseDelayPools(fields []string, lineNo int, diags *[]Diagnostic) int {
	if len(fields) != 2 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_pools",
			Message: "delay_pools: нужно одно число",
		})
		return 0
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil || n < 0 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_pools",
			Message: "delay_pools: неотрицательное целое",
		})
		return 0
	}
	return n
}

func parseDelayClass(fields []string, lineNo int, diags *[]Diagnostic) (pool, class int) {
	if len(fields) != 3 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_class",
			Message: "delay_class: нужны pool и class",
		})
		return 0, 0
	}
	pool, err1 := strconv.Atoi(fields[1])
	class, err2 := strconv.Atoi(fields[2])
	if err1 != nil || err2 != nil || pool <= 0 || class < 1 || class > 3 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_class",
			Message: "delay_class: pool > 0, class 1..3",
		})
		return 0, 0
	}
	return pool, class
}

func parseDelayParameters(fields []string, lineNo int, diags *[]Diagnostic) (pool int, p DelayParams) {
	if len(fields) < 3 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_parameters",
			Message: "delay_parameters: нужны pool и restore/max",
		})
		return 0, p
	}
	pool, err := strconv.Atoi(fields[1])
	if err != nil || pool <= 0 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_parameters",
			Message: "delay_parameters: номер pool > 0",
		})
		return 0, p
	}
	parsed, err := parseDelayParameterRates(fields[2:])
	if err != nil {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_parameters",
			Message: err.Error(),
		})
		return 0, p
	}
	p = parsed
	p.LineNo = lineNo
	return pool, p
}

func parseDelayParameterRates(fields []string) (DelayParams, error) {
	var p DelayParams
	if len(fields) == 0 {
		return p, fmt.Errorf("delay_parameters: нужны restore/max")
	}
	if len(fields) == 1 {
		r, m, err := parseDelayPairSpec(fields[0])
		if err != nil {
			return p, err
		}
		p.IndRestore, p.IndMax = r, m
		p.AggRestore, p.AggMax = -1, -1
		return p, nil
	}
	if len(fields) == 2 && (isDelayPairSpec(fields[0]) || isDelayPairSpec(fields[1])) {
		ar, am, err := parseDelayPairSpec(fields[0])
		if err != nil {
			return p, err
		}
		p.AggRestore, p.AggMax = ar, am
		ir, im, err := parseDelayPairSpec(fields[1])
		if err != nil {
			return p, err
		}
		p.IndRestore, p.IndMax = ir, im
		return p, nil
	}
	if len(fields) >= 4 {
		ar, err := parseDelayRateToken(fields[0])
		if err != nil {
			return p, err
		}
		am, err := parseDelayRateToken(fields[1])
		if err != nil {
			return p, err
		}
		ir, err := parseDelayRateToken(fields[2])
		if err != nil {
			return p, err
		}
		im, err := parseDelayRateToken(fields[3])
		if err != nil {
			return p, err
		}
		p.AggRestore, p.AggMax = ar, am
		p.IndRestore, p.IndMax = ir, im
		return p, nil
	}
	r, err := parseDelayRateToken(fields[0])
	if err != nil {
		return p, err
	}
	m, err := parseDelayRateToken(fields[1])
	if err != nil {
		return p, err
	}
	p.IndRestore, p.IndMax = r, m
	p.AggRestore, p.AggMax = -1, -1
	return p, nil
}

func isDelayPairSpec(tok string) bool {
	tok = strings.TrimSpace(strings.ToLower(tok))
	return tok == "none" || tok == "-1" || strings.Contains(tok, "/")
}

func parseDelayPairSpec(tok string) (restore, max int64, err error) {
	tok = strings.TrimSpace(strings.ToLower(tok))
	switch tok {
	case "none", "-1", "-1/-1":
		return -1, -1, nil
	}
	if strings.Contains(tok, "/") {
		parts := strings.SplitN(tok, "/", 2)
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("delay_parameters: формат restore/max")
		}
		restore, err = parseDelayRateToken(parts[0])
		if err != nil {
			return 0, 0, err
		}
		max, err = parseDelayRateToken(parts[1])
		return restore, max, err
	}
	restore, err = parseDelayRateToken(tok)
	if err != nil {
		return 0, 0, err
	}
	return restore, restore, nil
}

func parseDelayInitial(fields []string, lineNo int, diags *[]Diagnostic) (pool int, size int64) {
	if len(fields) != 3 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_initial_bucket_size",
			Message: "delay_initial_bucket_size: нужны pool и размер",
		})
		return 0, 0
	}
	pool, err := strconv.Atoi(fields[1])
	if err != nil || pool <= 0 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_initial_bucket_size",
			Message: "номер pool > 0",
		})
		return 0, 0
	}
	size, err = parseDelaySizeToken(fields[2])
	if err != nil {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_initial_bucket_size",
			Message: err.Error(),
		})
		return 0, 0
	}
	return pool, size
}

func parseDelayAccess(fields []string, spans []fieldSpan, raw string, lineNo int, diags *[]Diagnostic) DelayAccessLine {
	var line DelayAccessLine
	if len(fields) < 3 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_access",
			Message: "delay_access: нужны pool и allow|deny",
		})
		return line
	}
	pool, err := strconv.Atoi(fields[1])
	if err != nil || pool <= 0 {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_access",
			Message: "delay_access: pool > 0",
		})
		return line
	}
	act := strings.ToLower(fields[2])
	if act != "allow" && act != "deny" {
		*diags = append(*diags, Diagnostic{
			Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_delay_access",
			Message: "allow или deny",
		})
		return line
	}
	var clauses []AccessClause
	for _, sp := range spans[3:] {
		c, err := parseAccessToken(sp.text)
		if err != nil {
			col, endCol := spanColumns(raw, sp)
			*diags = append(*diags, Diagnostic{
				Line: lineNo, Column: col, EndColumn: endCol, Severity: SeverityError,
				Code: "parse_delay_access", Message: err.Error(),
			})
			continue
		}
		col, endCol := spanColumns(raw, sp)
		c.Column = col
		c.EndColumn = endCol
		clauses = append(clauses, c)
	}
	line = DelayAccessLine{
		LineNo: lineNo,
		Pool:   pool,
		Allow:  act == "allow",
		ACLs:   clauses,
	}
	return line
}

func parseDelayRateToken(raw string) (int64, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0, fmt.Errorf("пустое значение скорости")
	}
	if raw == "-1" || raw == "none" {
		return -1, nil
	}
	return parseDelaySizeToken(raw)
}

func parseDelaySizeToken(raw string) (int64, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "-1" {
		return -1, nil
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(raw, "kb/s"):
		mult = 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "kb/s"))
	case strings.HasSuffix(raw, "mb/s"):
		mult = 1024 * 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "mb/s"))
	case strings.HasSuffix(raw, "gb/s"):
		mult = 1024 * 1024 * 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "gb/s"))
	case strings.HasSuffix(raw, "kb"):
		mult = 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "kb"))
	case strings.HasSuffix(raw, "mb"):
		mult = 1024 * 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "mb"))
	case strings.HasSuffix(raw, "gb"):
		mult = 1024 * 1024 * 1024
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "gb"))
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("неверное число %q", raw)
	}
	return n * mult, nil
}

func validateDelaySection(sec DelaySection, defined map[string]bool, diags *[]Diagnostic) {
	if sec.PoolCount == 0 {
		if len(sec.Access) > 0 || len(sec.Class) > 0 || len(sec.Parameters) > 0 {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "delay_pools",
				Message: "delay_access/delay_class без delay_pools",
			})
		}
		return
	}
	for pool := 1; pool <= sec.PoolCount; pool++ {
		if _, ok := sec.Class[pool]; !ok {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "delay_class",
				Message: fmt.Sprintf("нет delay_class для pool %d", pool),
			})
		}
		if _, ok := sec.Parameters[pool]; !ok {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "delay_parameters",
				Message: fmt.Sprintf("нет delay_parameters для pool %d", pool),
			})
		}
	}
	for pool := range sec.Class {
		if pool < 1 || pool > sec.PoolCount {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "delay_class",
				Message: fmt.Sprintf("delay_class: pool %d вне 1..%d", pool, sec.PoolCount),
			})
		}
	}
	for pool := range sec.Parameters {
		if pool < 1 || pool > sec.PoolCount {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "delay_parameters",
				Message: fmt.Sprintf("delay_parameters: pool %d вне 1..%d", pool, sec.PoolCount),
			})
		}
	}
	for _, line := range sec.Access {
		if line.Pool < 1 || line.Pool > sec.PoolCount {
			*diags = append(*diags, Diagnostic{
				Line: line.LineNo, Severity: SeverityError, Code: "delay_access",
				Message: fmt.Sprintf("delay_access: pool %d вне 1..%d", line.Pool, sec.PoolCount),
			})
		}
		if len(line.ACLs) == 0 {
			*diags = append(*diags, Diagnostic{
				Line: line.LineNo, Severity: SeverityError, Code: "delay_access",
				Message: "delay_access без acl",
			})
		}
		for _, cl := range line.ACLs {
			if !defined[cl.Name] {
				*diags = append(*diags, Diagnostic{
					Line: line.LineNo, Column: cl.Column, EndColumn: cl.EndColumn,
					Severity: SeverityError, Code: "unknown_acl",
					Message: fmt.Sprintf("неизвестный acl %q", cl.Name), ACLName: cl.Name,
				})
			}
		}
	}
}

func buildDelayRuntime(sec DelaySection) (*ratelimit.Runtime, error) {
	if sec.PoolCount <= 0 {
		return nil, nil
	}
	defs := make([]ratelimit.PoolDef, sec.PoolCount+1)
	for pool := 1; pool <= sec.PoolCount; pool++ {
		class := sec.Class[pool]
		if class == 0 {
			class = 1
		}
		params := sec.Parameters[pool]
		initial := sec.InitialBucket[pool]
		if initial == 0 && params.IndMax > 0 {
			initial = params.IndMax / 2
		}
		defs[pool] = ratelimit.PoolDef{
			Class:         class,
			AggRestoreBPS: params.AggRestore,
			AggMaxBytes:   params.AggMax,
			IndRestoreBPS: params.IndRestore,
			IndMaxBytes:   params.IndMax,
			Initial:       initial,
		}
	}
	return ratelimit.NewRuntime(defs), nil
}
