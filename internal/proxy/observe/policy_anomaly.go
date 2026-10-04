package observe

import "oktopus/internal/proxy/observe/policyanomaly"

// PolicyAnomalyCheck — результат одного kind в extra.policy_anomaly.
type PolicyAnomalyCheck = policyanomaly.CheckResult

// PolicyAnomalyPayload — extra.policy_anomaly (ключ = kind).
type PolicyAnomalyPayload = policyanomaly.Payload

// PolicyAnomalyDetectInput — вход EvaluatePolicyAnomaly.
type PolicyAnomalyDetectInput = policyanomaly.Input

// PolicyNameMismatch — совместимость (kind + detect в map).
type PolicyNameMismatch = policyanomaly.CheckResult

const (
	PolicyAnomalyHostSNIMismatch     = policyanomaly.KindHostSNIMismatch
	PolicyAnomalyURLHostMismatch     = policyanomaly.KindURLHostMismatch
	PolicyAnomalyConnectPortMismatch = policyanomaly.KindConnectPortMismatch
	PolicyAnomalyConnectLiteralIP    = policyanomaly.KindConnectLiteralIP
	PolicyAnomalyDstResolvePrivate   = policyanomaly.KindDstResolvePrivate
)

// EvaluatePolicyAnomaly — все зарегистрированные kind с detect true/false.
func EvaluatePolicyAnomaly(in PolicyAnomalyDetectInput) PolicyAnomalyPayload {
	return policyanomaly.Evaluate(in)
}

// DetectPolicyAnomalies — alias EvaluatePolicyAnomaly.
func DetectPolicyAnomalies(in PolicyAnomalyDetectInput) PolicyAnomalyPayload {
	return EvaluatePolicyAnomaly(in)
}

// RegisteredPolicyAnomalyKinds — kind из registry.
func RegisteredPolicyAnomalyKinds() []string {
	return policyanomaly.RegisteredKinds()
}

// DetectPolicyNameMismatch — legacy; nil если host_sni_mismatch не detect.
func DetectPolicyNameMismatch(connectHostPort, tlsClientSNI, policyHost, httpHostHeader string) *PolicyNameMismatch {
	p := EvaluatePolicyAnomaly(PolicyAnomalyDetectInput{
		ConnectHostPort: connectHostPort,
		TLSClientSNI:    tlsClientSNI,
		PolicyHost:      policyHost,
		HTTPHostHeader:  httpHostHeader,
	})
	if r, ok := p[PolicyAnomalyHostSNIMismatch]; ok && r.Detect {
		return &r
	}
	return nil
}
