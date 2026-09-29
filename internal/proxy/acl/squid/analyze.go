package squid

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"oktopus/internal/proxy/acl"
)

// Analyze разбирает конфиг, проверяет ссылки на acl и собирает Engine.
func Analyze(configText string, lists []ListInput) AnalyzeResult {
	return analyzeConfig(configText, lists, true)
}

// AnalyzeValidate — проверка для API без сборки Engine и SNI index.
func AnalyzeValidate(configText string, lists []ListInput) AnalyzeResult {
	return analyzeConfig(configText, lists, false)
}

func analyzeConfig(configText string, lists []ListInput, buildEngine bool) AnalyzeResult {
	cfg, diags := ParsePolicyConfig(configText)
	return analyzeParsedConfig(cfg, lists, diags, buildEngine)
}

// AnalyzeParsedConfig проверяет или компилирует уже разобранный конфиг.
func AnalyzeParsedConfig(cfg *Config, lists []ListInput, parseDiags []Diagnostic, buildEngine bool) AnalyzeResult {
	return analyzeParsedConfig(cfg, lists, parseDiags, buildEngine)
}

func analyzeParsedConfig(cfg *Config, lists []ListInput, diags []Diagnostic, buildEngine bool) AnalyzeResult {
	if cfg != nil {
		mergeDBListsCollect(cfg, lists, &diags)
	}
	if cfg == nil {
		return AnalyzeResult{OK: false, Diagnostics: diags}
	}

	var named map[string]*compiledACL
	var acc *acl.SNIIndexCompileAcc
	if buildEngine {
		acl.BeginCompileLabels()
		acc = acl.NewSNIIndexCompileAcc()
		named = make(map[string]*compiledACL, len(cfg.ACLs))
		var inline []Line
		var listACLs []Line
		for _, def := range cfg.ACLs {
			if def.ACLListBody != "" {
				listACLs = append(listACLs, def)
			} else {
				inline = append(inline, def)
			}
		}
		for _, def := range inline {
			ca, err := compileACL(def, acc)
			if err != nil {
				diags = append(diags, aclCompileDiagnostic(def, err))
				continue
			}
			named[def.ACLName] = ca
		}
		if len(listACLs) > 0 {
			workers := 4
			if workers > len(listACLs) {
				workers = len(listACLs)
			}
			jobs := make(chan Line)
			var wg sync.WaitGroup
			var mu sync.Mutex
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for def := range jobs {
						ca, err := compileACL(def, acc)
						mu.Lock()
						if err != nil {
							diags = append(diags, aclCompileDiagnostic(def, err))
						} else {
							named[def.ACLName] = ca
						}
						mu.Unlock()
					}
				}()
			}
			for _, def := range listACLs {
				jobs <- def
			}
			close(jobs)
			wg.Wait()
		}
		acl.EndCompileLabels()
	} else {
		for _, def := range cfg.ACLs {
			if err := validateACLPatterns(def); err != nil {
				diags = append(diags, aclCompileDiagnostic(def, err))
			}
		}
	}

	collectAccessReferenceDiagnostics(cfg, &diags)
	validateDelaySection(cfg.Delay, cfgDefinedSet(cfg), &diags)

	if hasErrors(diags) {
		return AnalyzeResult{
			OK:          false,
			DefinedACLs: definedACLNames(cfg),
			Diagnostics: diags,
		}
	}

	if !buildEngine {
		if _, err := buildDelayRuntime(cfg.Delay); err != nil {
			diags = append(diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "compile", Message: err.Error(),
			})
			return AnalyzeResult{
				OK:          false,
				DefinedACLs: definedACLNames(cfg),
				Diagnostics: diags,
			}
		}
		return AnalyzeResult{
			OK:          true,
			DefinedACLs: definedACLNames(cfg),
			Diagnostics: diags,
		}
	}

	engine, err := assembleEngineFromNamed(cfg, named)
	if err != nil {
		diags = append(diags, Diagnostic{
			Line: 0, Severity: SeverityError, Code: "compile", Message: err.Error(),
		})
		return AnalyzeResult{
			OK:          false,
			DefinedACLs: definedACLNames(cfg),
			Diagnostics: diags,
		}
	}

	if len(cfg.HTTPAccess) == 0 {
		engine = acl.EmptyEngine()
	}

	var sniReport acl.SNIPatternIndexReport
	if acc != nil {
		sniReport = acc.Report()
	}

	return AnalyzeResult{
		OK:             true,
		DefinedACLs:    definedACLNames(cfg),
		Diagnostics:    diags,
		Engine:         engine,
		SNIIndexReport: sniReport,
	}
}

func aclCompileDiagnostic(def Line, err error) Diagnostic {
	d := Diagnostic{
		Line: def.LineNo, Column: 1, Severity: SeverityError,
		Code: "invalid_acl", ACLName: def.ACLName,
	}
	if def.LineNo <= 0 {
		d.Line = 0
		d.Code = "list"
		d.Message = fmt.Sprintf("list %q: %v", def.ACLName, err)
	} else {
		d.Message = fmt.Sprintf("acl %q: %v", def.ACLName, err)
	}
	return d
}

func cfgDefinedSet(cfg *Config) map[string]bool {
	defined := make(map[string]bool, len(cfg.DefinitionAt))
	for name := range cfg.DefinitionAt {
		defined[name] = true
	}
	return defined
}

func collectAccessReferenceDiagnostics(cfg *Config, diags *[]Diagnostic) {
	defined := cfgDefinedSet(cfg)
	for _, line := range cfg.HTTPAccess {
		if len(line.AccessACLs) == 0 {
			*diags = append(*diags, Diagnostic{
				Line: line.LineNo, Column: 1, Severity: SeverityError,
				Code: "empty_http_access", Message: "http_access без ссылок на acl",
			})
			continue
		}
		for _, cl := range line.AccessACLs {
			if !defined[cl.Name] {
				*diags = append(*diags, Diagnostic{
					Line: line.LineNo, Column: cl.Column, EndColumn: cl.EndColumn,
					Severity: SeverityError, Code: "unknown_acl",
					Message: fmt.Sprintf("неизвестный acl %q", cl.Name), ACLName: cl.Name,
				})
			}
		}
	}
	for _, line := range cfg.SSLVerify {
		if len(line.AccessACLs) == 0 {
			*diags = append(*diags, Diagnostic{
				Line: line.LineNo, Column: 1, Severity: SeverityError,
				Code: "empty_ssl_verify", Message: "ssl_verify без ссылок на acl",
			})
			continue
		}
		for _, cl := range line.AccessACLs {
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

func parseConfigCollect(text string, diags *[]Diagnostic) *Config {
	cfg := &Config{DefinitionAt: make(map[string]int)}
	text = NormalizePolicyText(text)
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		lineNo := i + 1
		if isPolicyCommentOrEmpty(raw) {
			continue
		}
		line := normalizePolicyLine(raw)
		spans := splitFieldSpans(line)
		if len(spans) == 0 {
			continue
		}
		kw := strings.ToLower(spans[0].text)
		switch kw {
		case "acl":
			if len(spans) < 3 {
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_acl",
					Message: "acl: нужны имя и тип",
				})
				continue
			}
			name := spans[1].text
			typ, err := parseACLType(spans[2].text)
			if err != nil {
				col, _ := spanColumns(raw, spans[2])
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: col, Severity: SeverityError, Code: "parse_acl",
					Message: err.Error(),
				})
				continue
			}
			vals := make([]string, 0, len(spans)-3)
			for _, sp := range spans[3:] {
				vals = append(vals, sp.text)
			}
			if typ == ACLTypeAll && len(vals) > 0 {
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_acl",
					Message: "acl all не принимает значения",
				})
				continue
			}
			if typ != ACLTypeAll && len(vals) == 0 {
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_acl",
					Message: fmt.Sprintf("acl %s: нет значений", typ),
				})
				continue
			}
			if _, dup := cfg.DefinitionAt[name]; dup {
				col, endCol := spanColumns(raw, spans[1])
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: col, EndColumn: endCol, Severity: SeverityError,
					Code: "duplicate_acl", Message: fmt.Sprintf("acl %q уже определён", name), ACLName: name,
				})
				continue
			}
			idx := len(cfg.ACLs)
			cfg.DefinitionAt[name] = idx
			cfg.ACLs = append(cfg.ACLs, Line{
				Kind: "acl", LineNo: lineNo, ACLName: name, ACLType: typ, ACLValues: vals,
			})
		case "http_access":
			if len(spans) < 2 {
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_http_access",
					Message: "http_access: нужно allow или deny",
				})
				continue
			}
			act := strings.ToLower(spans[1].text)
			if act != "allow" && act != "deny" {
				col, _ := spanColumns(raw, spans[1])
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: col, Severity: SeverityError, Code: "parse_http_access",
					Message: "действие allow или deny",
				})
				continue
			}
			var clauses []AccessClause
			for _, sp := range spans[2:] {
				c, err := parseAccessToken(sp.text)
				if err != nil {
					col, endCol := spanColumns(raw, sp)
					*diags = append(*diags, Diagnostic{
						Line: lineNo, Column: col, EndColumn: endCol, Severity: SeverityError,
						Code: "parse_http_access", Message: err.Error(),
					})
					continue
				}
				col, endCol := spanColumns(raw, sp)
				c.Column = col
				c.EndColumn = endCol
				clauses = append(clauses, c)
			}
			cfg.HTTPAccess = append(cfg.HTTPAccess, Line{
				Kind: "http_access", LineNo: lineNo, AccessAct: act, AccessACLs: clauses,
			})
		case "ssl_verify":
			if len(spans) < 2 {
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: 1, Severity: SeverityError, Code: "parse_ssl_verify",
					Message: "ssl_verify: нужно require или skip",
				})
				continue
			}
			act := strings.ToLower(spans[1].text)
			if act != "require" && act != "skip" {
				col, _ := spanColumns(raw, spans[1])
				*diags = append(*diags, Diagnostic{
					Line: lineNo, Column: col, Severity: SeverityError, Code: "parse_ssl_verify",
					Message: "действие require или skip",
				})
				continue
			}
			var clauses []AccessClause
			for _, sp := range spans[2:] {
				c, err := parseAccessToken(sp.text)
				if err != nil {
					col, endCol := spanColumns(raw, sp)
					*diags = append(*diags, Diagnostic{
						Line: lineNo, Column: col, EndColumn: endCol, Severity: SeverityError,
						Code: "parse_ssl_verify", Message: err.Error(),
					})
					continue
				}
				col, endCol := spanColumns(raw, sp)
				c.Column = col
				c.EndColumn = endCol
				clauses = append(clauses, c)
			}
			cfg.SSLVerify = append(cfg.SSLVerify, Line{
				Kind: "ssl_verify", LineNo: lineNo, AccessAct: act, AccessACLs: clauses,
			})
		case "delay_pools":
			initDelayMaps(cfg)
			fields := spanTexts(spans)
			cfg.Delay.PoolCount = parseDelayPools(fields, lineNo, diags)
		case "delay_class":
			initDelayMaps(cfg)
			fields := spanTexts(spans)
			pool, class := parseDelayClass(fields, lineNo, diags)
			if pool > 0 {
				cfg.Delay.Class[pool] = class
			}
		case "delay_parameters":
			initDelayMaps(cfg)
			fields := spanTexts(spans)
			pool, p := parseDelayParameters(fields, lineNo, diags)
			if pool > 0 {
				cfg.Delay.Parameters[pool] = p
			}
		case "delay_initial_bucket_size":
			initDelayMaps(cfg)
			fields := spanTexts(spans)
			pool, size := parseDelayInitial(fields, lineNo, diags)
			if pool > 0 {
				cfg.Delay.InitialBucket[pool] = size
			}
		case "delay_access":
			initDelayMaps(cfg)
			fields := spanTexts(spans)
			line := parseDelayAccess(fields, spans, raw, lineNo, diags)
			if line.Pool > 0 {
				cfg.Delay.Access = append(cfg.Delay.Access, line)
			}
		default:
			col, endCol := spanColumns(raw, spans[0])
			*diags = append(*diags, Diagnostic{
				Line: lineNo, Column: col, EndColumn: endCol, Severity: SeverityError, Code: "unknown_directive",
				Message: fmt.Sprintf("неизвестная директива %q (ожидается acl, http_access, ssl_verify или delay_*)", spans[0].text),
			})
		}
	}
	return cfg
}

func mergeDBListsCollect(cfg *Config, lists []ListInput, diags *[]Diagnostic) {
	for _, l := range lists {
		name := strings.TrimSpace(l.Name)
		if name == "" {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "list", Message: "list: пустое имя",
			})
			continue
		}
		typ, err := listTypeToACL(l.ListType)
		if err != nil {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "list",
				Message: fmt.Sprintf("list %q: %v", name, err), ACLName: name,
			})
			continue
		}
		if !acl.HasPatternLines(l.Body) {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "list",
				Message: fmt.Sprintf("list %q: пустое содержимое", name), ACLName: name,
			})
			continue
		}
		if _, exists := cfg.DefinitionAt[name]; exists {
			*diags = append(*diags, Diagnostic{
				Line: 0, Severity: SeverityError, Code: "list",
				Message: fmt.Sprintf("list %q: имя занято acl в конфиге", name), ACLName: name,
			})
			continue
		}
		idx := len(cfg.ACLs)
		cfg.DefinitionAt[name] = idx
		cfg.ACLs = append(cfg.ACLs, Line{
			Kind: "acl", LineNo: 0, ACLName: name, ACLType: typ, ACLListBody: l.Body,
		})
	}
}

func assembleEngineFromNamed(cfg *Config, named map[string]*compiledACL) (*acl.Engine, error) {
	if len(cfg.HTTPAccess) == 0 {
		return acl.EmptyEngine(), nil
	}
	var access []acl.SquidHTTPAccess
	for _, line := range cfg.HTTPAccess {
		clauses := make([]acl.SquidAccessClause, 0, len(line.AccessACLs))
		for _, cl := range line.AccessACLs {
			m := named[cl.Name]
			if m == nil {
				continue
			}
			clauses = append(clauses, acl.SquidAccessClause{
				Matcher:  m.matcher,
				Negated:  cl.Negated,
				Priority: m.priority,
			})
		}
		sort.SliceStable(clauses, func(i, j int) bool {
			return clauses[i].Priority < clauses[j].Priority
		})
		act := acl.ActionDeny
		if line.AccessAct == "allow" {
			act = acl.ActionAllow
		}
		access = append(access, acl.SquidHTTPAccess{
			LineNo:  line.LineNo,
			Action:  act,
			RuleRef: FormatAccessDirective("http_access", line.AccessAct, line.AccessACLs),
			Clauses: clauses,
		})
	}
	var sslVerify []acl.SquidSSLVerify
	for _, line := range cfg.SSLVerify {
		clauses := make([]acl.SquidAccessClause, 0, len(line.AccessACLs))
		for _, cl := range line.AccessACLs {
			m := named[cl.Name]
			if m == nil {
				continue
			}
			clauses = append(clauses, acl.SquidAccessClause{
				Matcher:  m.matcher,
				Negated:  cl.Negated,
				Priority: m.priority,
			})
		}
		sort.SliceStable(clauses, func(i, j int) bool {
			return clauses[i].Priority < clauses[j].Priority
		})
		sslVerify = append(sslVerify, acl.SquidSSLVerify{
			LineNo:     line.LineNo,
			SkipVerify: line.AccessAct == "skip",
			RuleRef:    FormatAccessDirective("ssl_verify", line.AccessAct, line.AccessACLs),
			Clauses:    clauses,
		})
	}

	rateRT, err := buildDelayRuntime(cfg.Delay)
	if err != nil {
		return nil, err
	}
	var delayAccess []acl.SquidDelayAccess
	for _, line := range cfg.Delay.Access {
		clauses := make([]acl.SquidAccessClause, 0, len(line.ACLs))
		for _, cl := range line.ACLs {
			m := named[cl.Name]
			if m == nil {
				continue
			}
			clauses = append(clauses, acl.SquidAccessClause{
				Matcher:  m.matcher,
				Negated:  cl.Negated,
				Priority: m.priority,
			})
		}
		sort.SliceStable(clauses, func(i, j int) bool {
			return clauses[i].Priority < clauses[j].Priority
		})
		act := acl.ActionDeny
		if line.Allow {
			act = acl.ActionAllow
		}
		delayAccess = append(delayAccess, acl.SquidDelayAccess{
			LineNo:  line.LineNo,
			Pool:    line.Pool,
			Action:  act,
			Clauses: clauses,
		})
	}

	return acl.NewSquidEngine(access, sslVerify, delayAccess, rateRT), nil
}

func initDelayMaps(cfg *Config) {
	if cfg.Delay.Class == nil {
		cfg.Delay.Class = make(map[int]int)
		cfg.Delay.Parameters = make(map[int]DelayParams)
		cfg.Delay.InitialBucket = make(map[int]int64)
	}
}

func spanTexts(spans []fieldSpan) []string {
	out := make([]string, len(spans))
	for i, sp := range spans {
		out[i] = sp.text
	}
	return out
}
