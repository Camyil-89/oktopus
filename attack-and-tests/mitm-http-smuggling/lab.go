package main

import "oktopus/attack-and-tests/poclib"

const (
	allowedListen  = "127.0.0.1:9443"
	internalListen = "127.0.0.1:9555"
)

func cmdLab(args []string) int {
	stop, _ := startLab()
	_ = stop
	select {}
}

func startLab() (func(), error) {
	s1, err := poclib.StartTLSLab(allowedListen, "ALLOWED", "/")
	if err != nil {
		return nil, err
	}
	s2, err := poclib.StartTLSLab(internalListen, poclib.SecretMarker, "/secret")
	if err != nil {
		s1()
		return nil, err
	}
	return func() { s1(); s2() }, nil
}
