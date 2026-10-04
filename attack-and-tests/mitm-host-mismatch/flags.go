package main

import (
	"flag"

	"oktopus/attack-and-tests/setup"
)

type runOpts struct {
	proxyAddr    string
	caFile       string
	connectHost  string
	tlsSNI       string
	internalURL  string
	internalHost string
	mode         string
	proxyUser    string
	proxyPass    string
	runControl   bool
	connectMode  string // mitm | tunnel (прогон all)
	api          setup.APIFlags
	skipSetup    bool
}

func parseRunFlags(args []string) (runOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := runOpts{
		proxyAddr:    setup.TestInstanceDialAddr,
		caFile:       "config/ca.crt",
		connectHost:  "localhost:9443",
		tlsSNI:       "localhost",
		internalURL:  "https://127.0.0.1:9555/secret",
		internalHost: "127.0.0.1:9555",
		mode:         "absolute",
		runControl:   true,
		api:          setup.DefaultAPIFlags(),
	}
	fs.StringVar(&o.proxyAddr, "proxy", o.proxyAddr, "адрес oktopus proxy (при -skip-setup)")
	fs.StringVar(&o.caFile, "ca", o.caFile, "CA MITM прокси")
	fs.StringVar(&o.connectHost, "connect", o.connectHost, "CONNECT (разрешённый dstdomain)")
	fs.StringVar(&o.tlsSNI, "sni", o.tlsSNI, "TLS SNI внутри CONNECT")
	fs.StringVar(&o.internalURL, "evil-url", o.internalURL, "абсолютный URL внутри туннеля")
	fs.StringVar(&o.internalHost, "evil-host", o.internalHost, "Host для -mode relative")
	fs.StringVar(&o.mode, "mode", o.mode, "absolute | relative")
	fs.StringVar(&o.proxyUser, "proxy-user", o.proxyUser, "Proxy-Authorization user (при -skip-setup)")
	fs.StringVar(&o.proxyPass, "proxy-pass", o.proxyPass, "Proxy-Authorization password")
	fs.BoolVar(&o.runControl, "control", o.runControl, "прямой GET на internal (ожидаем deny)")
	fs.StringVar(&o.api.APIBase, "api", o.api.APIBase, "базовый URL API (go run ./cmd serve)")
	fs.StringVar(&o.api.APIUser, "api-user", o.api.APIUser, "логин UI")
	fs.StringVar(&o.api.APIPass, "api-pass", o.api.APIPass, "пароль UI")
	fs.BoolVar(&o.skipSetup, "skip-setup", false, "не трогать API; вручную proxy/ACL/auth")
	if err := fs.Parse(args); err != nil {
		return runOpts{}, err
	}
	return o, nil
}
