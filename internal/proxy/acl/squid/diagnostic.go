package squid

import (
	"sort"

	"oktopus/internal/proxy/acl"
)

// Severity — уровень диагностики.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic — ошибка или предупреждение с позицией в конфиге.
type Diagnostic struct {
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndColumn int    `json:"end_column,omitempty"`
	Severity  Severity `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	ACLName   string `json:"acl_name,omitempty"`
}

// AnalyzeResult — результат разбора и проверки конфига.
type AnalyzeResult struct {
	OK             bool                       `json:"ok"`
	DefinedACLs    []string                   `json:"defined_acls"`
	Diagnostics    []Diagnostic               `json:"diagnostics"`
	Engine         *acl.Engine                `json:"-"`
	SNIIndexReport acl.SNIPatternIndexReport  `json:"-"`
}

// CompileErrors — несколько диагностик при неуспешной компиляции.
type CompileErrors struct {
	Diagnostics []Diagnostic
}

func (e *CompileErrors) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "ошибка компиляции ACL"
	}
	return e.Diagnostics[0].Message
}

func hasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

func definedACLNames(cfg *Config) []string {
	if cfg == nil || len(cfg.DefinitionAt) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(cfg.DefinitionAt))
	for name := range cfg.DefinitionAt {
		out = append(out, name)
	}
	sortStrings(out)
	return out
}

func sortStrings(ss []string) {
	sort.Strings(ss)
}
