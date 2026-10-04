package policyanomaly

const KindHostSNIMismatch = "host_sni_mismatch"

func init() {
	Register(KindHostSNIMismatch, detectHostSNIMismatch)
}

func detectHostSNIMismatch(in Input) *CheckResult {
	ch := normalizeHost(in.ConnectHostPort)
	th := normalizeHost(in.TLSClientSNI)
	ph := normalizeHost(in.PolicyHost)
	hh := normalizeHost(in.HTTPHostHeader)
	urlH := normalizeHost(in.URLHost)

	if ch == "" && th == "" && ph == "" && hh == "" {
		return nil
	}

	mismatch := false
	if hostsDiffer(th, ph) || hostsDiffer(th, ch) || hostsDiffer(ph, ch) {
		mismatch = true
	}
	if hostsDiffer(hh, ch) || hostsDiffer(hh, th) {
		mismatch = true
	}
	if hostsDiffer(hh, ph) {
		onlyURLHostSplit := urlH != "" && hh != "" && hostsDiffer(urlH, hh) && ph == urlH
		if !onlyURLHostSplit {
			mismatch = true
		}
	}
	if !mismatch {
		return nil
	}
	return &CheckResult{
		Detect:         true,
		ConnectHost:    ch,
		TLSClientSNI:   th,
		PolicyHost:     ph,
		HTTPHostHeader: hh,
	}
}
