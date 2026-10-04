package observe

import (
	"strings"
)

// PolicyNameMismatch — расхождение CONNECT / TLS SNI / HTTP host для журнала.
type PolicyNameMismatch struct {
	Kind           string `json:"kind"`
	ConnectHost    string `json:"connect_host,omitempty"`
	TLSClientSNI   string `json:"tls_client_sni,omitempty"`
	PolicyHost     string `json:"policy_host,omitempty"`
	HTTPHostHeader string `json:"http_host,omitempty"`
}

const PolicyAnomalyHostSNIMismatch = "host_sni_mismatch"

// DetectPolicyNameMismatch помечает попытку несогласованных имён (MITM). nil — всё согласовано или данных нет.
func DetectPolicyNameMismatch(connectHostPort, tlsClientSNI, policyHost, httpHostHeader string) *PolicyNameMismatch {
	ch := normalizeTraceHost(connectHostPort)
	th := normalizeTraceHost(tlsClientSNI)
	ph := normalizeTraceHost(policyHost)
	hh := normalizeTraceHost(httpHostHeader)

	if ch == "" && th == "" && ph == "" && hh == "" {
		return nil
	}

	mismatch := false
	if hostsDiffer(th, ph) || hostsDiffer(th, ch) || hostsDiffer(ph, ch) || hostsDiffer(hh, ph) {
		mismatch = true
	}
	if !mismatch {
		return nil
	}
	return &PolicyNameMismatch{
		Kind:           PolicyAnomalyHostSNIMismatch,
		ConnectHost:    ch,
		TLSClientSNI:   th,
		PolicyHost:     ph,
		HTTPHostHeader: hh,
	}
}

func hostsDiffer(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return !strings.EqualFold(a, b)
}

func normalizeTraceHost(hostPort string) string {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}
	return hostOnly(hostPort)
}
