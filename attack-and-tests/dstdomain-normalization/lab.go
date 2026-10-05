package main

import "oktopus/attack-and-tests/poclib"

const labAddr = "127.0.0.1:9443"

func cmdLab(args []string) int {
	stop, _ := startLab()
	_ = stop
	select {}
}

func startLab() (func(), error) {
	return poclib.StartTLSLab(labAddr, "NORMALIZED_OK", "/")
}
