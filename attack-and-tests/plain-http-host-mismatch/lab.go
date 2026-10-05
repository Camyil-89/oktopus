package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const (
	allowedHTTP  = "127.0.0.1:9080"
	internalHTTP = "127.0.0.1:9081"
)

func cmdLab(args []string) int {
	log.SetFlags(0)
	stop, _ := startLab()
	_ = stop
	select {}
}

func startLab() (func(), error) {
	s1, err := poclib.StartHTTPLab(allowedHTTP, "ALLOWED_HTTP", "/")
	if err != nil {
		return nil, err
	}
	s2, err := poclib.StartHTTPLab(internalHTTP, poclib.SecretMarker, "/secret")
	if err != nil {
		s1()
		return nil, err
	}
	return func() { s1(); s2() }, nil
}
