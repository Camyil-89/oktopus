package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const (
	allowedListen  = "127.0.0.1:9443"
	forbiddenListen = "127.0.0.1:9778"
	allowedMarker  = "ALLOWED_PORT"
	forbiddenMarker = poclib.SecretMarker
)

func cmdLab(args []string) int {
	log.SetFlags(0)
	stop, err := startLab()
	if err != nil {
		return 1
	}
	_ = stop
	select {}
}

func startLab() (func(), error) {
	stop1, err := poclib.StartTLSLab(allowedListen, allowedMarker, "/")
	if err != nil {
		return nil, err
	}
	stop2, err := poclib.StartTLSLab(forbiddenListen, forbiddenMarker, "/secret")
	if err != nil {
		stop1()
		return nil, err
	}
	return func() { stop1(); stop2() }, nil
}
