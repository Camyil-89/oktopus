package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const (
	labAddr   = "127.0.0.1:9777"
	labMarker = "IP_BYPASS_HIT"
)

func cmdLab(args []string) int {
	log.SetFlags(0)
	stop, err := startLab()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	log.Printf("lab: TCP %s", labAddr)
	_ = stop
	select {}
}

func startLab() (func(), error) {
	return poclib.StartTCPLab(labAddr, labMarker)
}
