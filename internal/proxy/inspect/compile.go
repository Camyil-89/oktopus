package inspect

import (
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/google/uuid"
)

const maxScriptBytes = 64 * 1024

// CompiledRule — метаданные правила в активной программе.
type CompiledRule struct {
	ID     uuid.UUID
	Name   string
	Action int16 // 0 = deny on match, 1 = allow on match
}

// Program — скомпилированный набор Lua-правил.
type Program struct {
	source string
	rules  []CompiledRule
}

type ruleSource struct {
	ID     uuid.UUID
	Name   string
	Action int16
	Script string
}

// RuleInput — вход для компиляции из домена.
type RuleInput struct {
	ID     uuid.UUID
	Name   string
	Action int16
	Script string
}

// BuildProgram компилирует включённые правила в Program.
func BuildProgram(inputs []RuleInput) (*Program, error) {
	src := make([]ruleSource, len(inputs))
	for i := range inputs {
		src[i] = ruleSource(inputs[i])
	}
	return CompileProgram(src)
}

// CompileProgram проверяет скрипты и собирает Program.
func CompileProgram(rules []ruleSource) (*Program, error) {
	if len(rules) == 0 {
		return &Program{}, nil
	}
	compiled := make([]CompiledRule, 0, len(rules))
	var b strings.Builder
	for i, r := range rules {
		script := strings.TrimSpace(r.Script)
		if script == "" {
			return nil, fmt.Errorf("rule %q: empty script", r.Name)
		}
		if len(script) > maxScriptBytes {
			return nil, fmt.Errorf("rule %q: script too large (max %d bytes)", r.Name, maxScriptBytes)
		}
		if err := validateScript(script); err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		compiled = append(compiled, CompiledRule{
			ID:     r.ID,
			Name:   r.Name,
			Action: r.Action,
		})
		fmt.Fprintf(&b, "-- rule %d: %s\n", i, r.Name)
		fmt.Fprintf(&b, "inspect_%d = (function()\n", i)
		b.WriteString(script)
		b.WriteString("\nreturn inspect\nend)()\n\n")
	}
	full := b.String()
	if err := validateCombined(full); err != nil {
		return nil, err
	}
	return &Program{source: full, rules: compiled}, nil
}

func validateScript(script string) error {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(script); err != nil {
		return fmt.Errorf("lua compile: %w", err)
	}
	if fn := L.GetGlobal("inspect"); fn.Type() != lua.LTFunction {
		return fmt.Errorf("script must define function inspect(ctx)")
	}
	return nil
}

func validateCombined(full string) error {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(full); err != nil {
		return fmt.Errorf("lua compile combined: %w", err)
	}
	return nil
}

// ValidateScript проверяет один Lua-скрипт (как при сборке правила).
func ValidateScript(script string) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return fmt.Errorf("empty script")
	}
	if len(script) > maxScriptBytes {
		return fmt.Errorf("script too large (max %d bytes)", maxScriptBytes)
	}
	if err := validateScript(script); err != nil {
		return err
	}
	wrapped := fmt.Sprintf("inspect_0 = (function()\n%s\nreturn inspect\nend)()", script)
	return validateCombined(wrapped)
}
