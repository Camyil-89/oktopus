package extra

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var ruleIDKey = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// PolicyAnomalyKVRuleID — KV для extra.policy_anomaly (совпадает с accesslog.PolicyNoteInspectRuleID).
var PolicyAnomalyKVRuleID = uuid.MustParse("00000000-0000-4000-8000-000000000001")

// KVRow — одно поле ctx:log в области inspect-правила.
type KVRow struct {
	LogID         uuid.UUID
	InspectRuleID uuid.UUID
	FieldKey      string
	FieldValue    string
}

// Parsed — разбор extra для записи в ClickHouse.
type Parsed struct {
	SearchEngine string
	SearchQuery  string
	InspectError string
	KV           []KVRow
}

// ParseFromJSON читает extra, собранный buildExtraJSON.
func ParseFromJSON(logID uuid.UUID, raw []byte) Parsed {
	out := Parsed{}
	if len(raw) == 0 {
		return out
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return out
	}
	if v, ok := m["inspect_error"]; ok {
		out.InspectError = jsonString(v)
	}
	if v, ok := m["search"]; ok {
		var search map[string]json.RawMessage
		if json.Unmarshal(v, &search) == nil {
			out.SearchEngine = jsonString(search["engine"])
			out.SearchQuery = jsonString(search["query"])
		}
	}
	if v, ok := m["policy_anomaly"]; ok && len(v) > 0 {
		out.KV = append(out.KV, KVRow{
			LogID:         logID,
			InspectRuleID: PolicyAnomalyKVRuleID,
			FieldKey:      "policy_anomaly",
			FieldValue:    string(v),
		})
	}
	for key, payload := range m {
		if key == "inspect_error" || key == "search" || key == "policy_anomaly" || key == "gateway_error" {
			continue
		}
		if !ruleIDKey.MatchString(key) {
			continue
		}
		ruleID, err := uuid.Parse(key)
		if err != nil {
			continue
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal(payload, &obj) != nil {
			continue
		}
		for fieldKey, fieldRaw := range obj {
			fieldKey = strings.TrimSpace(fieldKey)
			if fieldKey == "" {
				continue
			}
			out.KV = append(out.KV, KVRow{
				LogID:         logID,
				InspectRuleID: ruleID,
				FieldKey:      fieldKey,
				FieldValue:    rawToStorageString(fieldRaw),
			})
		}
	}
	return out
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(raw))
}

func rawToStorageString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		if b {
			return "true"
		}
		return "false"
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return string(raw)
}

// BuildJSON собирает extra для API из колонок и KV.
func BuildJSON(searchEngine, searchQuery, inspectError string, kv []KVRow) []byte {
	m := make(map[string]any)
	if strings.TrimSpace(inspectError) != "" {
		m["inspect_error"] = inspectError
	}
	if strings.TrimSpace(searchEngine) != "" || strings.TrimSpace(searchQuery) != "" {
		m["search"] = map[string]any{
			"engine": searchEngine,
			"query":  searchQuery,
		}
	}
	byRule := make(map[string]map[string]any)
	for _, row := range kv {
		if row.InspectRuleID == PolicyAnomalyKVRuleID && row.FieldKey == "policy_anomaly" {
			m["policy_anomaly"] = decodeStoredValue(row.FieldValue)
			continue
		}
		rid := row.InspectRuleID.String()
		block, ok := byRule[rid]
		if !ok {
			block = make(map[string]any)
			byRule[rid] = block
		}
		block[row.FieldKey] = decodeStoredValue(row.FieldValue)
	}
	for rid, block := range byRule {
		m[rid] = block
	}
	if len(m) == 0 {
		return []byte("{}")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func decodeStoredValue(s string) any {
	s = strings.TrimSpace(s)
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
		var v any
		if json.Unmarshal([]byte(s), &v) == nil {
			return v
		}
	}
	return s
}
