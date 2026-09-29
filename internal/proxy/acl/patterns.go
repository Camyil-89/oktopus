package acl

import (
	"fmt"
	"regexp"
	"strings"
)

func splitPatternLines(text string) []string {
	var out []string
	_ = ForEachPatternLine(text, func(line string) error {
		out = append(out, line)
		return nil
	})
	return out
}

// HasPatternLines — есть ли в тексте хотя бы одна непустая не-комментарий строка.
func HasPatternLines(text string) bool {
	found := false
	_ = ForEachPatternLine(text, func(line string) error {
		found = true
		return errStopPatternScan
	})
	return found
}

var errStopPatternScan = fmt.Errorf("stop")

// ForEachPatternLine обходит строки списка без материализации []string.
func ForEachPatternLine(text string, fn func(line string) error) error {
	any := false
	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		any = true
		if err := fn(line); err != nil {
			if err == errStopPatternScan {
				return nil
			}
			return err
		}
	}
	if !any {
		return fmt.Errorf("нет значений")
	}
	return nil
}

func compilePattern(rt RuleType, pat string, acc *sniIndexCompileAcc) (compiledPattern, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return compiledPattern{}, fmt.Errorf("пустой шаблон")
	}
	if rt == RuleSRC || rt == RuleDST {
		cp, err := parseIPPatternLine(pat)
		if err != nil {
			return compiledPattern{}, err
		}
		return cp, nil
	}
	if rt == RulePORT {
		cp, err := parsePortPatternLine(pat)
		if err != nil {
			return compiledPattern{}, err
		}
		return cp, nil
	}
	if rt == RuleSNI {
		if cp, ok := compileSNIFastPattern(pat); ok {
			return cp, nil
		}
		if acc != nil {
			acc.noteRegexp(pat, sniRegexpFallbackReason(pat))
		}
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return compiledPattern{}, fmt.Errorf("regex: %w", err)
	}
	return compiledPattern{re: re}, nil
}
