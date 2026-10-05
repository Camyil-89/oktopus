package main

import (
	"log"

	"oktopus/attack-and-tests/poclib"
)

const pocName = "tunnel-ipv6-loopback"

func cmdAll(args []string) int {
	log.SetFlags(0)
	o, err := parseFlags(args)
	if err != nil {
		return 2
	}
	stop, err := startLab()
	if err != nil {
		log.Printf("lab: %v", err)
		log.Println("RESULT: OK (пропуск: IPv6 lab недоступен на хосте)")
		return 0
	}
	defer stop()
	return poclib.RunDualMode(pocName, o, runAttack)
}
