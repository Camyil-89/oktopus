package policyanomaly

import "net"

const KindConnectLiteralIP = "connect_literal_ip"

func init() {
	Register(KindConnectLiteralIP, detectConnectLiteralIP)
}

func detectConnectLiteralIP(in Input) *CheckResult {
	ch := normalizeHost(in.ConnectHostPort)
	if ch == "" || net.ParseIP(ch) == nil {
		return nil
	}
	return &CheckResult{
		Detect:      true,
		ConnectHost: ch,
		ConnectPort: tracePort(in.ConnectHostPort),
		PolicyHost:  normalizeHost(in.PolicyHost),
	}
}
