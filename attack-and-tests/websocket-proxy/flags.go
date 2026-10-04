package main

import (
	"flag"

	"oktopus/attack-and-tests/setup"
)

type runOpts struct {
	proxyAddr    string
	caFile       string
	wssConnect   string // CONNECT host:port (TLS origin)
	wssSNI       string
	wsHTTPURL    string // absolute http://… для explicit proxy
	proxyUser    string
	proxyPass    string
	connectMode  string // mitm | tunnel (пусто — mitm для run)
	api          setup.APIFlags
	skipSetup    bool
}

func parseRunFlags(args []string) (runOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := runOpts{
		proxyAddr:  setup.TestInstanceDialAddr,
		caFile:     "config/ca.crt",
		wssConnect: "localhost:9443",
		wssSNI:     "localhost",
		wsHTTPURL:  "http://localhost:9080/",
		api:        setup.DefaultAPIFlags(),
	}
	fs.StringVar(&o.proxyAddr, "proxy", o.proxyAddr, "адрес oktopus proxy (при -skip-setup)")
	fs.StringVar(&o.caFile, "ca", o.caFile, "CA MITM прокси")
	fs.StringVar(&o.wssConnect, "wss-connect", o.wssConnect, "CONNECT к TLS origin (wss)")
	fs.StringVar(&o.wssSNI, "wss-sni", o.wssSNI, "TLS ServerName внутри CONNECT")
	fs.StringVar(&o.wsHTTPURL, "ws-url", o.wsHTTPURL, "absolute URL для ws через HTTP proxy")
	fs.StringVar(&o.proxyUser, "proxy-user", o.proxyUser, "Proxy-Authorization user")
	fs.StringVar(&o.proxyPass, "proxy-pass", o.proxyPass, "Proxy-Authorization password")
	fs.StringVar(&o.api.APIBase, "api", o.api.APIBase, "базовый URL API")
	fs.StringVar(&o.api.APIUser, "api-user", o.api.APIUser, "логин UI")
	fs.StringVar(&o.api.APIPass, "api-pass", o.api.APIPass, "пароль UI")
	fs.BoolVar(&o.skipSetup, "skip-setup", false, "не трогать API")
	fs.StringVar(&o.connectMode, "connect-mode", "", "mitm или tunnel (run; по умолчанию mitm)")
	if err := fs.Parse(args); err != nil {
		return runOpts{}, err
	}
	return o, nil
}
