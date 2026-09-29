package inspect

import (
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// MatchResult — итог проверки одного запроса.
type MatchResult struct {
	Matched  bool
	Deny     bool
	RuleID   string
	RuleName string
	// RuleLogs — данные ctx:log по id правила (пишутся в extra журнала).
	RuleLogs map[string]map[string]any
}

// Runner выполняет Lua-правила на исходящих запросах.
type Runner struct {
	program *Program
	pool    sync.Pool
}

func NewRunner(program *Program) *Runner {
	if program == nil {
		program = &Program{}
	}
	r := &Runner{program: program}
	r.pool.New = func() any {
		L := lua.NewState()
		if program.source != "" {
			_ = L.DoString(program.source)
		}
		return L
	}
	return r
}

func (r *Runner) Empty() bool {
	return r == nil || len(r.program.rules) == 0
}

func (r *Runner) RuleCount() int {
	if r == nil || r.program == nil {
		return 0
	}
	return len(r.program.rules)
}

// Eval прогоняет правила по порядку: DENY при совпадении — сразу выход;
// ALLOW при совпадении — продолжаем; если DENY не сработал — итог ALLOW.
func (r *Runner) Eval(c RequestContext) (MatchResult, error) {
	if r.Empty() {
		return MatchResult{}, nil
	}
	L, ok := r.pool.Get().(*lua.LState)
	if !ok {
		L = lua.NewState()
		_ = L.DoString(r.program.source)
	}
	defer r.pool.Put(L)

	var lastAllow MatchResult
	ruleLogs := make(map[string]map[string]any)

	for i, rule := range r.program.rules {
		fn := L.GetGlobal(fmtInspectFn(i))
		if fn.Type() != lua.LTFunction {
			continue
		}
		scratch := make(map[string]any)
		L.Push(fn)
		pushRequestContext(L, c, scratch)
		if err := L.PCall(1, 1, nil); err != nil {
			return MatchResult{RuleLogs: ruleLogs}, err
		}
		matched := false
		if L.GetTop() > 0 {
			matched = inspectReturnMatched(L, 1)
			L.Pop(1)
		}
		if len(scratch) > 0 {
			ruleLogs[rule.ID.String()] = scratch
		}
		if !matched {
			continue
		}
		if rule.Action == 0 {
			return MatchResult{
				Matched:  true,
				Deny:     true,
				RuleID:   rule.ID.String(),
				RuleName: rule.Name,
				RuleLogs: ruleLogs,
			}, nil
		}
		lastAllow = MatchResult{
			Matched:  true,
			Deny:     false,
			RuleID:   rule.ID.String(),
			RuleName: rule.Name,
			RuleLogs: ruleLogs,
		}
	}
	if lastAllow.Matched {
		return lastAllow, nil
	}
	if len(ruleLogs) > 0 {
		return MatchResult{RuleLogs: ruleLogs}, nil
	}
	return MatchResult{}, nil
}

func fmtInspectFn(i int) string {
	return "inspect_" + itoa(i)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func inspectReturnMatched(L *lua.LState, nRet int) bool {
	if nRet == 0 {
		return false
	}
	v := L.Get(-1)
	switch val := v.(type) {
	case lua.LBool:
		return val == lua.LTrue
	case lua.LString:
		s := stringsTrimLower(string(val))
		switch s {
		case "", "allow", "skip", "false":
			return false
		default:
			return true
		}
	case *lua.LTable:
		a := stringsTrimLower(tableStringField(val, "action"))
		switch a {
		case "", "allow", "skip":
			return false
		case "match", "log", "tag", "deny", "block":
			return true
		default:
			return true
		}
	default:
		return v != lua.LNil && v != lua.LFalse
	}
}

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
