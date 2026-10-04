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
	httpHost     string
	httpPath     string
	internalURL  string
	proxyUser    string
	proxyPass    string
	runControl   bool
	connectMode  string
	api          setup.APIFlags
	skipSetup    bool
}

func parseRunFlags(args []string) (runOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := runOpts{
		proxyAddr:   "127.0.0.1:8080",
		caFile:      "config/ca.crt",
		connectHost: "localhost:9443",
		tlsSNI:      "internal.blocked",
		httpHost:    "localhost",
		httpPath:    "/secret",
		internalURL: "https://127.0.0.1:9555/secret",
		runControl:  true,
		api:         setup.DefaultAPIFlags(),
	}
	fs.StringVar(&o.proxyAddr, "proxy", o.proxyAddr, "адрес oktopus proxy (при -skip-setup)")
	fs.StringVar(&o.caFile, "ca", o.caFile, "CA MITM прокси")
	fs.StringVar(&o.connectHost, "connect", o.connectHost, "CONNECT host:port (разрешённый)")
	fs.StringVar(&o.tlsSNI, "sni", o.tlsSNI, "TLS ClientHello SNI (запрещённый dstdomain, не IP — иначе leaf MITM не сверить с CA)")
	fs.StringVar(&o.httpHost, "http-host", o.httpHost, "HTTP Host (разрешённый)")
	fs.StringVar(&o.httpPath, "path", o.httpPath, "путь внутри туннеля")
	fs.StringVar(&o.internalURL, "internal-url", o.internalURL, "control: прямой GET на internal")
	fs.StringVar(&o.proxyUser, "proxy-user", o.proxyUser, "Proxy-Authorization user")
	fs.StringVar(&o.proxyPass, "proxy-pass", o.proxyPass, "Proxy-Authorization password")
	fs.BoolVar(&o.runControl, "control", o.runControl, "прямой GET на internal (ожидаем deny)")
	fs.StringVar(&o.api.APIBase, "api", o.api.APIBase, "базовый URL API")
	fs.StringVar(&o.api.APIUser, "api-user", o.api.APIUser, "логин UI")
	fs.StringVar(&o.api.APIPass, "api-pass", o.api.APIPass, "пароль UI")
	fs.BoolVar(&o.skipSetup, "skip-setup", false, "не трогать API")
	if err := fs.Parse(args); err != nil {
		return runOpts{}, err
	}
	return o, nil
}
