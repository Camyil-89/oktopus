package main

import "oktopus/attack-and-tests/poclib"

const (
	labAddr   = "127.0.0.1:9667"
	labMarker = "DNS_REBIND_HIT"
	// Публичный DNS: имя → 127.0.0.1 (класс rebinding через внешний резолвер).
	connectHost = "9667.127.0.0.1.sslip.io:9667"
)

func cmdLab(args []string) int {
	stop, _ := startLab()
	_ = stop
	select {}
}

func startLab() (func(), error) {
	return poclib.StartTCPLab(labAddr, labMarker)
}
