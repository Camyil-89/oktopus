package main

import (
	"flag"

	"oktopus/attack-and-tests/poclib"
	"oktopus/attack-and-tests/setup"
)

func parseFlags(args []string) (poclib.RunOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := poclib.RunOpts{ProxyAddr: "127.0.0.1:8080"}
	o.API = setup.DefaultAPIFlags()
	fs.StringVar(&o.ProxyAddr, "proxy", o.ProxyAddr, "прокси")
	fs.StringVar(&o.API.APIBase, "api", o.API.APIBase, "API")
	fs.StringVar(&o.API.APIUser, "api-user", o.API.APIUser, "логин")
	fs.StringVar(&o.API.APIPass, "api-pass", o.API.APIPass, "пароль")
	fs.BoolVar(&o.SkipSetup, "skip-setup", false, "ручной setup")
	return o, fs.Parse(args)
}
