package policyanomaly

const KindURLHostMismatch = "url_host_mismatch"

func init() {
	Register(KindURLHostMismatch, detectURLHostMismatch)
}

func detectURLHostMismatch(in Input) *CheckResult {
	urlH := normalizeHost(in.URLHost)
	hh := normalizeHost(in.HTTPHostHeader)
	if urlH == "" || hh == "" || !hostsDiffer(urlH, hh) {
		return nil
	}
	return &CheckResult{
		Detect:         true,
		URLHost:        urlH,
		HTTPHostHeader: hh,
		PolicyHost:     normalizeHost(in.PolicyHost),
	}
}
