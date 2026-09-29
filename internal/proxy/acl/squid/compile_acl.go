package squid

import (
	"fmt"
	"strings"

	"oktopus/internal/proxy/acl"
)

type compiledACL struct {
	matcher  acl.SquidMatcher
	priority int
}

// validateACLPatterns проверяет строки шаблонов без сборки matcher/SNI index.
func validateACLPatterns(def Line) error {
	switch def.ACLType {
	case ACLTypeAll, ACLTypeLDAPGroup:
		return nil
	case ACLTypeSrc:
		return validatePatternLines(acl.RuleSRC, def)
	case ACLTypeDst:
		return validatePatternLines(acl.RuleDST, def)
	case ACLTypeDstDomain:
		return validatePatternLines(acl.RuleSNI, def)
	case ACLTypePort:
		return validatePatternLines(acl.RulePORT, def)
	case ACLTypeURLRegex:
		return validatePatternLines(acl.RuleAll, def)
	default:
		return fmt.Errorf("acl %q: внутренняя ошибка типа", def.ACLName)
	}
}

func validatePatternLines(rt acl.RuleType, def Line) error {
	return foreachPatternLine(def, func(i int, line string) error {
		_, err := acl.CompileSquidPatternLine(rt, line, nil)
		if err != nil {
			return fmt.Errorf("строка %d: %s", i, patternErrorText(err))
		}
		return nil
	})
}

func compileACL(def Line, acc *acl.SNIIndexCompileAcc) (*compiledACL, error) {
	switch def.ACLType {
	case ACLTypeAll:
		return &compiledACL{matcher: acl.SquidMatchAll{}, priority: acl.SquidPriorityAll}, nil
	case ACLTypeLDAPGroup:
		groups := append([]string(nil), def.ACLValues...)
		return &compiledACL{
			matcher:  acl.SquidMatchLDAPGroups{Groups: groups},
			priority: acl.SquidPriorityLDAP,
		}, nil
	case ACLTypeSrc:
		pats, err := compilePatternLinesFromDef(acl.RuleSRC, def, nil)
		if err != nil {
			return nil, fmt.Errorf("acl %q: %w", def.ACLName, err)
		}
		return &compiledACL{matcher: acl.NewSquidMatchPatterns(acl.RuleSRC, pats), priority: acl.SquidPrioritySrc}, nil
	case ACLTypeDst:
		pats, err := compilePatternLinesFromDef(acl.RuleDST, def, nil)
		if err != nil {
			return nil, fmt.Errorf("acl %q: %w", def.ACLName, err)
		}
		return &compiledACL{matcher: acl.NewSquidMatchPatterns(acl.RuleDST, pats), priority: acl.SquidPriorityDst}, nil
	case ACLTypeDstDomain:
		m, err := acl.CompileSquidSNIMatcher(func(yield func(string) error) error {
			return foreachPatternLine(def, func(_ int, line string) error {
				return yield(line)
			})
		}, acc)
		if err != nil {
			return nil, fmt.Errorf("acl %q: %w", def.ACLName, err)
		}
		return &compiledACL{matcher: m, priority: acl.SquidPriorityDstDomain}, nil
	case ACLTypePort:
		pats, err := compilePatternLinesFromDef(acl.RulePORT, def, nil)
		if err != nil {
			return nil, fmt.Errorf("acl %q: %w", def.ACLName, err)
		}
		return &compiledACL{matcher: acl.NewSquidMatchPatterns(acl.RulePORT, pats), priority: acl.SquidPriorityPort}, nil
	case ACLTypeURLRegex:
		pats, err := compilePatternLinesFromDef(acl.RuleAll, def, nil)
		if err != nil {
			return nil, fmt.Errorf("acl %q: %w", def.ACLName, err)
		}
		return &compiledACL{matcher: acl.NewSquidMatchPatterns(acl.RuleAll, pats), priority: acl.SquidPriorityURLRegex}, nil
	default:
		return nil, fmt.Errorf("acl %q: внутренняя ошибка типа", def.ACLName)
	}
}

func compilePatternLinesFromDef(rt acl.RuleType, def Line, acc *acl.SNIIndexCompileAcc) ([]acl.CompiledPatternPublic, error) {
	var out []acl.CompiledPatternPublic
	err := foreachPatternLine(def, func(i int, line string) error {
		cp, err := acl.CompileSquidPatternLine(rt, line, acc)
		if err != nil {
			return fmt.Errorf("строка %d: %s", i, patternErrorText(err))
		}
		out = append(out, cp)
		return nil
	})
	return out, err
}

func patternErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.LastIndex(msg, ": "); idx >= 0 {
		return msg[idx+2:]
	}
	return msg
}
