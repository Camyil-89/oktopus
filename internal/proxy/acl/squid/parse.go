package squid

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"oktopus/internal/proxy/acl"
)

// ACLType — тип acl в конфиге (squid-совместимые имена).
type ACLType string

const (
	ACLTypeSrc       ACLType = "src"
	ACLTypeDst       ACLType = "dst"
	ACLTypeDstDomain ACLType = "dstdomain"
	ACLTypePort      ACLType = "port"
	ACLTypeURLRegex  ACLType = "url_regex"
	ACLTypeLDAPGroup ACLType = "ldap_group"
	ACLTypeAll       ACLType = "all"
)

// Line — одна директива конфига.
type Line struct {
	Kind       string // "acl" | "http_access"
	LineNo     int
	ACLName     string
	ACLType     ACLType
	ACLValues   []string   // inline значения из acl в конфиге
	ACLListBody string     // тело named list из БД (без материализации строк)
	AccessAct  string // allow | deny
	AccessACLs []AccessClause
}

// AccessClause — ссылка на acl в http_access.
type AccessClause struct {
	Name      string
	Negated   bool
	Column    int // 1-based в строке
	EndColumn int // 1-based, конец токена
}

// Config — разобранный конфиг (acl, http_access, ssl_verify, delay_pools).
type Config struct {
	ACLs         []Line
	HTTPAccess   []Line
	SSLVerify    []Line
	Delay        DelaySection
	DefinitionAt map[string]int // acl name -> index in ACLs slice
}

const utf8BOM = "\ufeff"

// NormalizePolicyText убирает BOM в начале файла.
func NormalizePolicyText(text string) string {
	return strings.TrimPrefix(text, utf8BOM)
}

func normalizePolicyLine(raw string) string {
	line := strings.TrimSpace(raw)
	line = strings.TrimPrefix(line, utf8BOM)
	return strings.TrimSpace(line)
}

func isPolicyCommentOrEmpty(raw string) bool {
	line := normalizePolicyLine(raw)
	return line == "" || strings.HasPrefix(line, "#")
}

// ListInput — именованный list из БД (до merge в acl).
type ListInput struct {
	Name     string
	ListType string // src | dstdomain | port
	Body     string
}

// ParseConfig разбирает текст политики (acl / http_access).
func ParseConfig(text string) (*Config, error) {
	cfg := &Config{DefinitionAt: make(map[string]int)}
	text = NormalizePolicyText(text)
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		lineNo := i + 1
		if isPolicyCommentOrEmpty(raw) {
			continue
		}
		line := normalizePolicyLine(raw)
		fields := splitFields(line)
		if len(fields) == 0 {
			continue
		}
		kw := strings.ToLower(fields[0])
		switch kw {
		case "acl":
			if len(fields) < 3 {
				return nil, fmt.Errorf("строка %d: acl: нужны имя и тип", lineNo)
			}
			name := fields[1]
			typ, err := parseACLType(fields[2])
			if err != nil {
				return nil, fmt.Errorf("строка %d: %w", lineNo, err)
			}
			vals := fields[3:]
			if typ == ACLTypeAll && len(vals) > 0 {
				return nil, fmt.Errorf("строка %d: acl all не принимает значения", lineNo)
			}
			if typ != ACLTypeAll && len(vals) == 0 {
				return nil, fmt.Errorf("строка %d: acl %s: нет значений", lineNo, typ)
			}
			if _, dup := cfg.DefinitionAt[name]; dup {
				return nil, fmt.Errorf("строка %d: acl %q уже определён", lineNo, name)
			}
			idx := len(cfg.ACLs)
			cfg.DefinitionAt[name] = idx
			cfg.ACLs = append(cfg.ACLs, Line{
				Kind:      "acl",
				LineNo:    lineNo,
				ACLName:   name,
				ACLType:   typ,
				ACLValues: vals,
			})
		case "http_access":
			if len(fields) < 2 {
				return nil, fmt.Errorf("строка %d: http_access: нужно allow или deny", lineNo)
			}
			act := strings.ToLower(fields[1])
			if act != "allow" && act != "deny" {
				return nil, fmt.Errorf("строка %d: http_access: действие allow или deny", lineNo)
			}
			var clauses []AccessClause
			for _, tok := range fields[2:] {
				c, err := parseAccessToken(tok)
				if err != nil {
					return nil, fmt.Errorf("строка %d: %w", lineNo, err)
				}
				clauses = append(clauses, c)
			}
			cfg.HTTPAccess = append(cfg.HTTPAccess, Line{
				Kind:       "http_access",
				LineNo:     lineNo,
				AccessAct:  act,
				AccessACLs: clauses,
			})
		case "ssl_verify":
			if len(fields) < 2 {
				return nil, fmt.Errorf("строка %d: ssl_verify: нужно require или skip", lineNo)
			}
			act := strings.ToLower(fields[1])
			if act != "require" && act != "skip" {
				return nil, fmt.Errorf("строка %d: ssl_verify: действие require или skip", lineNo)
			}
			var clauses []AccessClause
			for _, tok := range fields[2:] {
				c, err := parseAccessToken(tok)
				if err != nil {
					return nil, fmt.Errorf("строка %d: %w", lineNo, err)
				}
				clauses = append(clauses, c)
			}
			cfg.SSLVerify = append(cfg.SSLVerify, Line{
				Kind:       "ssl_verify",
				LineNo:     lineNo,
				AccessAct:  act,
				AccessACLs: clauses,
			})
		case "delay_pools":
			if cfg.Delay.Class == nil {
				cfg.Delay.Class = make(map[int]int)
				cfg.Delay.Parameters = make(map[int]DelayParams)
				cfg.Delay.InitialBucket = make(map[int]int64)
			}
			n, err := strconv.Atoi(fields[1])
			if len(fields) != 2 || err != nil || n < 0 {
				return nil, fmt.Errorf("строка %d: delay_pools: неотрицательное целое", lineNo)
			}
			cfg.Delay.PoolCount = n
		case "delay_class":
			if len(fields) != 3 {
				return nil, fmt.Errorf("строка %d: delay_class: нужны pool и class", lineNo)
			}
			pool, err1 := strconv.Atoi(fields[1])
			class, err2 := strconv.Atoi(fields[2])
			if err1 != nil || err2 != nil || pool <= 0 || class < 1 || class > 3 {
				return nil, fmt.Errorf("строка %d: delay_class: pool > 0, class 1..3", lineNo)
			}
			if cfg.Delay.Class == nil {
				cfg.Delay.Class = make(map[int]int)
			}
			cfg.Delay.Class[pool] = class
		case "delay_parameters":
			fieldsCopy := append([]string(nil), fields...)
			var diags []Diagnostic
			pool, p := parseDelayParameters(fieldsCopy, lineNo, &diags)
			if len(diags) > 0 {
				return nil, fmt.Errorf("строка %d: %s", lineNo, diags[0].Message)
			}
			if cfg.Delay.Parameters == nil {
				cfg.Delay.Parameters = make(map[int]DelayParams)
			}
			cfg.Delay.Parameters[pool] = p
		case "delay_initial_bucket_size":
			if len(fields) != 3 {
				return nil, fmt.Errorf("строка %d: delay_initial_bucket_size: нужны pool и размер", lineNo)
			}
			pool, err1 := strconv.Atoi(fields[1])
			size, err := parseDelaySizeToken(fields[2])
			if err1 != nil || pool <= 0 || err != nil {
				return nil, fmt.Errorf("строка %d: delay_initial_bucket_size: pool > 0 и размер", lineNo)
			}
			if cfg.Delay.InitialBucket == nil {
				cfg.Delay.InitialBucket = make(map[int]int64)
			}
			cfg.Delay.InitialBucket[pool] = size
		case "delay_access":
			if len(fields) < 3 {
				return nil, fmt.Errorf("строка %d: delay_access: нужны pool и allow|deny", lineNo)
			}
			pool, err := strconv.Atoi(fields[1])
			if err != nil || pool <= 0 {
				return nil, fmt.Errorf("строка %d: delay_access: pool > 0", lineNo)
			}
			act := strings.ToLower(fields[2])
			if act != "allow" && act != "deny" {
				return nil, fmt.Errorf("строка %d: delay_access: allow или deny", lineNo)
			}
			var clauses []AccessClause
			for _, tok := range fields[3:] {
				c, err := parseAccessToken(tok)
				if err != nil {
					return nil, fmt.Errorf("строка %d: %w", lineNo, err)
				}
				clauses = append(clauses, c)
			}
			cfg.Delay.Access = append(cfg.Delay.Access, DelayAccessLine{
				LineNo: lineNo, Pool: pool, Allow: act == "allow", ACLs: clauses,
			})
		default:
			return nil, fmt.Errorf("строка %d: неизвестная директива %q (ожидается acl, http_access, ssl_verify или delay_*)", lineNo, fields[0])
		}
	}
	return cfg, nil
}

// MergeDBLists добавляет acl из сущностей list (имя = имя list).
func MergeDBLists(cfg *Config, lists []ListInput) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	for _, l := range lists {
		name := strings.TrimSpace(l.Name)
		if name == "" {
			return fmt.Errorf("list: пустое имя")
		}
		typ, err := listTypeToACL(l.ListType)
		if err != nil {
			return fmt.Errorf("list %q: %w", name, err)
		}
		if !acl.HasPatternLines(l.Body) {
			return fmt.Errorf("list %q: пустое содержимое", name)
		}
		if _, exists := cfg.DefinitionAt[name]; exists {
			return fmt.Errorf("list %q: имя занято acl в конфиге", name)
		}
		idx := len(cfg.ACLs)
		cfg.DefinitionAt[name] = idx
		cfg.ACLs = append(cfg.ACLs, Line{
			Kind:        "acl",
			LineNo:      0,
			ACLName:     name,
			ACLType:     typ,
			ACLListBody: l.Body,
		})
	}
	return nil
}

func listTypeToACL(listType string) (ACLType, error) {
	switch strings.ToLower(strings.TrimSpace(listType)) {
	case "src":
		return ACLTypeSrc, nil
	case "dst":
		return ACLTypeDst, nil
	case "dstdomain", "sni":
		return ACLTypeDstDomain, nil
	case "port":
		return ACLTypePort, nil
	default:
		return "", fmt.Errorf("тип list должен быть src, dstdomain или port")
	}
}

func parseACLType(raw string) (ACLType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "src":
		return ACLTypeSrc, nil
	case "dst":
		return ACLTypeDst, nil
	case "dstdomain", "sni":
		return ACLTypeDstDomain, nil
	case "port":
		return ACLTypePort, nil
	case "url_regex":
		return ACLTypeURLRegex, nil
	case "ldap_group":
		return ACLTypeLDAPGroup, nil
	case "all":
		return ACLTypeAll, nil
	default:
		return "", fmt.Errorf("неподдерживаемый тип acl %q", raw)
	}
}

func parseAccessToken(tok string) (AccessClause, error) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return AccessClause{}, fmt.Errorf("пустая ссылка на acl")
	}
	neg := false
	if strings.HasPrefix(tok, "!") {
		neg = true
		tok = strings.TrimPrefix(tok, "!")
	}
	if tok == "" {
		return AccessClause{}, fmt.Errorf("пустое имя acl после !")
	}
	return AccessClause{Name: tok, Negated: neg}, nil
}

func splitFields(line string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && unicode.IsSpace(rune(c)) {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

