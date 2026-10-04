package repository

import (
	"fmt"
	"strings"

	"oktopus/internal/db/proxyaccesslog/inspectlog"
)

var reportColumnFields = map[string]struct{}{
	"instance_id":         {},
	"destination_address": {},
	"source_address":      {},
	"user":                {},
	"action":              {},
	"denied_by":           {},
	"decision_rule_ref":   {},
	"search_engine":       {},
}

// ValidateReportFieldName — whitelist полей group_by / field_nonempty.
func ValidateReportFieldName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("empty field name")
	}
	if _, ok := reportColumnFields[name]; ok {
		return nil
	}
	if key, ok := inspectlog.ParseGroupByField(name); ok {
		return inspectlog.ValidateFieldKey(key)
	}
	return fmt.Errorf("unsupported field %q", name)
}

// MergeFieldNonempty объединяет фильтры отчёта и виджета без дубликатов.
func MergeFieldNonempty(global, widget []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range append(global, widget...) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
