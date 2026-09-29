package inspectlog

import (
	"fmt"
	"strings"
)

// CHJoinAlias — алиас LEFT JOIN для поля inspect_kv (ключ уже проверен).
func CHJoinAlias(field string) string {
	return "ilkv_" + field
}

// CHJoinClause — LEFT JOIN с argMin(field_value, inspect_rule_id): эквивалент ORDER BY inspect_rule_id LIMIT 1.
// В запросе основная таблица должна иметь алиас al.
func CHJoinClause(field string) (string, error) {
	field = strings.TrimSpace(field)
	if err := ValidateFieldKey(field); err != nil {
		return "", err
	}
	alias := CHJoinAlias(field)
	return fmt.Sprintf(`LEFT JOIN (
  SELECT log_id, argMin(field_value, inspect_rule_id) AS field_value
  FROM proxy_access_log_inspect_kv
  WHERE field_key = '%s'
  GROUP BY log_id
) AS %s ON %s.log_id = al.id`, field, alias, alias), nil
}

// ValueExprCH — значение inspect_log поля; требует соответствующий CHJoinClause в FROM.
func ValueExprCH(field string) (string, error) {
	field = strings.TrimSpace(field)
	if err := ValidateFieldKey(field); err != nil {
		return "", err
	}
	alias := CHJoinAlias(field)
	return fmt.Sprintf("coalesce(%s.field_value, '')", alias), nil
}
