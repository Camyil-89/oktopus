package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const pocName = "tunnel-connect-ip-bypass"

func cmdAll(args []string) int {
	log.SetFlags(0)
	o, err := parseFlags(args)
	if err != nil {
		return 2
	}
	stop, err := startLab()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	defer stop()
	return poclib.RunDualMode(pocName, o, runAttack)
}
