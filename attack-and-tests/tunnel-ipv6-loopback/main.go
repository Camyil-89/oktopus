package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("Tunnel CONNECT [::1] bypass", cmdAll, cmdLab, cmdRun)
}
