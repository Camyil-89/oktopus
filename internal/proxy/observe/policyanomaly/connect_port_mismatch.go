package policyanomaly

const KindConnectPortMismatch = "connect_port_mismatch"

func init() {
	Register(KindConnectPortMismatch, detectConnectPortMismatch)
}

func detectConnectPortMismatch(in Input) *CheckResult {
	connectPort := connectPortFromInput(in)
	httpPort := httpPortFromInput(in)
	if connectPort == 0 || httpPort == 0 || connectPort == httpPort {
		return nil
	}
	return &CheckResult{
		Detect:      true,
		ConnectHost: normalizeHost(in.ConnectHostPort),
		ConnectPort: connectPort,
		HTTPPort:    httpPort,
		PolicyHost:  normalizeHost(in.PolicyHost),
	}
}
