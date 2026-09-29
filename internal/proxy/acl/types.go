package acl

// RuleType — к какому полю запроса применяется шаблон (squid acl / matcher).
type RuleType string

const (
	RuleAll  RuleType = "ALL"
	RuleSNI  RuleType = "SNI"
	RulePath RuleType = "PATH"
	RuleSRC  RuleType = "SRC"
	RuleDST  RuleType = "DST"
	RulePORT RuleType = "PORT"
)

// Action — результат совпавшего правила.
type Action string

const (
	ActionAllow Action = "ALLOW"
	ActionDeny  Action = "DENY"
)
