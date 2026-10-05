// Package policyanomaly — детекторы policy_anomaly (один файл = один kind + Register).
//
// JSON: policy_anomaly["<kind>"] = {"detect":true|false, …}. CH: ILIKE по field_value,
// attack_kind — policyanomaly.CHSearchKindDetected(kind), вкладка «Атаки» — CHSearchDetectTrue.
package policyanomaly

import "encoding/json"

// CheckResult — результат одного зарегистрированного детектора (ключ в Payload = kind).
type CheckResult struct {
	Detect         bool     `json:"detect"`
	ConnectHost    string   `json:"connect_host,omitempty"`
	TLSClientSNI   string   `json:"tls_client_sni,omitempty"`
	PolicyHost     string   `json:"policy_host,omitempty"`
	HTTPHostHeader string   `json:"http_host,omitempty"`
	URLHost        string   `json:"url_host,omitempty"`
	ConnectPort    int      `json:"connect_port,omitempty"`
	HTTPPort       int      `json:"http_port,omitempty"`
	ResolvedIPs    []string `json:"resolved_ips,omitempty"`
}

// Payload — JSON extra.policy_anomaly: ключ = kind из Register.
type Payload map[string]CheckResult

// AnyDetected — хотя бы один kind с detect=true (вкладка «Атаки» в CH).
func (p Payload) AnyDetected() bool {
	for _, r := range p {
		if r.Detect {
			return true
		}
	}
	return false
}

// DetectedKinds возвращает kind с detect=true (порядок registry).
func (p Payload) DetectedKinds() []string {
	var out []string
	for _, kind := range RegisteredKinds() {
		if r, ok := p[kind]; ok && r.Detect {
			out = append(out, kind)
		}
	}
	return out
}

// MarshalJSON фиксирует порядок kind как в registry (удобнее diff и CH).
func (p Payload) MarshalJSON() ([]byte, error) {
	if len(p) == 0 {
		return []byte("{}"), nil
	}
	ordered := make(map[string]CheckResult, len(p))
	for _, kind := range RegisteredKinds() {
		if r, ok := p[kind]; ok {
			ordered[kind] = r
		}
	}
	for kind, r := range p {
		if _, ok := ordered[kind]; !ok {
			ordered[kind] = r
		}
	}
	return json.Marshal(ordered)
}

// Input — сигналы с прокси (CONNECT, MITM, ACL enrich).
type Input struct {
	ConnectHostPort string
	TLSClientSNI    string
	PolicyHost      string
	HTTPHostHeader  string
	URLHost         string
	ConnectPort     int
	HTTPPort        int
	DstResolved     []string
}

// Detector — nil или Detect=false трактуется как «не сработало».
type Detector func(Input) *CheckResult

type registeredDetector struct {
	kind string
	fn   Detector
}

var registry []registeredDetector

// Register добавляет детектор (обычно init() в <kind>.go).
func Register(kind string, fn Detector) {
	if kind == "" || fn == nil {
		return
	}
	registry = append(registry, registeredDetector{kind: kind, fn: fn})
}

// RegisteredKinds — порядок init-регистрации.
func RegisteredKinds() []string {
	out := make([]string, 0, len(registry))
	for _, d := range registry {
		out = append(out, d.kind)
	}
	return out
}

// Evaluate запускает все детекторы; для каждого kind в Payload всегда detect true/false.
func Evaluate(in Input) Payload {
	out := make(Payload, len(registry))
	for _, d := range registry {
		hit := d.fn(in)
		if hit != nil && hit.Detect {
			r := *hit
			r.Detect = true
			out[d.kind] = r
			continue
		}
		out[d.kind] = CheckResult{Detect: false}
	}
	return out
}

// CHSearchDetectTrue — подстрока для positionCaseInsensitive (любая атака).
const CHSearchDetectTrue = `"detect":true`

// CHSearchKindDetected возвращает подстроку для фильтра attack_kind в CH.
func CHSearchKindDetected(kind string) string {
	return `"` + kind + `":{"detect":true`
}
