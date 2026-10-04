package main

import (
	"flag"

	"oktopus/attack-and-tests/setup"
)

type runOpts struct {
	proxyAddr      string
	bypassConnect  string
	controlConnect string
	proxyUser      string
	proxyPass      string
	runControl     bool
	connectMode    string
	api            setup.APIFlags
	skipSetup      bool
}

func parseRunFlags(args []string) (runOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := runOpts{
		proxyAddr:      setup.TestInstanceDialAddr,
		bypassConnect:  "localhost:9666",
		controlConnect: "127.0.0.1:9666",
		runControl:     true,
		api:            setup.DefaultAPIFlags(),
	}
	fs.StringVar(&o.proxyAddr, "proxy", o.proxyAddr, "адрес oktopus proxy (при -skip-setup)")
	fs.StringVar(&o.bypassConnect, "bypass-connect", o.bypassConnect, "CONNECT по разрешённому домену")
	fs.StringVar(&o.controlConnect, "control-connect", o.controlConnect, "CONNECT по литералу IP")
	fs.StringVar(&o.proxyUser, "proxy-user", o.proxyUser, "Proxy-Authorization user")
	fs.StringVar(&o.proxyPass, "proxy-pass", o.proxyPass, "Proxy-Authorization password")
	fs.BoolVar(&o.runControl, "control", o.runControl, "CONNECT на литерал IP")
	fs.StringVar(&o.api.APIBase, "api", o.api.APIBase, "базовый URL API")
	fs.StringVar(&o.api.APIUser, "api-user", o.api.APIUser, "логин UI")
	fs.StringVar(&o.api.APIPass, "api-pass", o.api.APIPass, "пароль UI")
	fs.BoolVar(&o.skipSetup, "skip-setup", false, "не трогать API")
	if err := fs.Parse(args); err != nil {
		return runOpts{}, err
	}
	return o, nil
}
