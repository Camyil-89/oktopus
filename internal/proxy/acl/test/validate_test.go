package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
)

func TestValidateRulePatternInvalidRegex(t *testing.T) {
	err := acl.ValidateRulePattern(acl.RuleSNI, "*")
	if err == nil {
		t.Fatal("expected error for bare *")
	}
}

func TestValidateRulePatternOK(t *testing.T) {
	if err := acl.ValidateRulePattern(acl.RuleSNI, "(?i)example\\.com"); err != nil {
		t.Fatal(err)
	}
}
