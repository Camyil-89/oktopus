package inspectlog

import (
	"fmt"
	"regexp"
	"strings"
)

// WellKnownKeys — типичные ключи ctx:log (загрузки файлов и эвристики).
var WellKnownKeys = []string{
	"upload_method",
	"upload_content_type",
	"upload_content_length",
	"upload_filenames",
	"upload_header_has_filename",
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func isAllowed(field string) bool {
	return keyPattern.MatchString(field)
}

// ValidateFieldKey проверяет ключ inspect_log для фильтров отчёта.
func ValidateFieldKey(field string) error {
	field = strings.TrimSpace(field)
	if !keyPattern.MatchString(field) {
		return fmt.Errorf("invalid inspect log field")
	}
	if !isAllowed(field) {
		return fmt.Errorf("unsupported inspect log field")
	}
	return nil
}

// GroupByFieldName — имя group_by в report spec для inspect ctx:log поля.
func GroupByFieldName(field string) string {
	return "inspect_log." + field
}

// ParseGroupByField возвращает ключ inspect log или пустую строку.
func ParseGroupByField(name string) (string, bool) {
	const prefix = "inspect_log."
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(name, prefix)
	if key == "" || !isAllowed(key) {
		return "", false
	}
	return key, true
}
