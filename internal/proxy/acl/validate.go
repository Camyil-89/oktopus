package acl

import (
	"fmt"
	"strings"
)

// ValidateRulePattern проверяет multiline-шаблон правила (как при компиляции ACL).
func ValidateRulePattern(ruleType RuleType, pattern string) error {
	rt := RuleType(strings.ToUpper(strings.TrimSpace(string(ruleType))))
	if rt != RuleAll && rt != RuleSNI && rt != RulePath && rt != RuleSRC && rt != RuleDST && rt != RulePORT {
		return fmt.Errorf("тип должен быть ALL, SNI, PATH, SRC, DST или PORT")
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("шаблон пустой")
	}
	var lines []string
	if strings.Contains(pattern, "\n") {
		lines = splitPatternLines(pattern)
	} else {
		lines = []string{pattern}
	}
	if len(lines) == 0 {
		return fmt.Errorf("шаблон пустой")
	}
	for i, line := range lines {
		_, err := compilePattern(rt, line, nil)
		if err != nil {
			return patternLineError(i, err)
		}
	}
	return nil
}

func patternLineError(lineIdx int, err error) error {
	msg := err.Error()
	const needle = "pattern "
	if idx := strings.Index(msg, needle); idx >= 0 {
		rest := msg[idx+len(needle):]
		if cut := strings.Index(rest, ": "); cut >= 0 {
			msg = rest[cut+2:]
		}
	}
	return fmt.Errorf("строка %d: %s", lineIdx+1, msg)
}
