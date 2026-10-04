package main

import (
	"flag"

	"oktopus/attack-and-tests/poclib"
	"oktopus/attack-and-tests/setup"
)

func parseFlags(args []string) (poclib.RunOpts, error) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	o := poclib.RunOpts{ProxyAddr: setup.TestInstanceDialAddr}
	o.API = setup.DefaultAPIFlags()
	fs.StringVar(&o.ProxyAddr, "proxy", o.ProxyAddr, "адрес прокси при -skip-setup")
	fs.StringVar(&o.API.APIBase, "api", o.API.APIBase, "API")
	fs.StringVar(&o.API.APIUser, "api-user", o.API.APIUser, "логин UI")
	fs.StringVar(&o.API.APIPass, "api-pass", o.API.APIPass, "пароль UI")
	fs.BoolVar(&o.SkipSetup, "skip-setup", false, "не трогать API")
	if err := fs.Parse(args); err != nil {
		return poclib.RunOpts{}, err
	}
	return o, nil
}
