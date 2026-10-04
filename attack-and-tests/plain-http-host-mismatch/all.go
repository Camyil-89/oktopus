package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const pocName = "plain-http-host-mismatch"

func cmdAll(args []string) int {
	log.SetFlags(0)
	o, err := parseFlags(args)
	if err != nil {
		return 2
	}
	stop, err := startLab()
	if err != nil {
		return 1
	}
	defer stop()
	return poclib.RunDualMode(pocName, o, runAttack)
}
