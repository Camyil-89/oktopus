package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("DNS resolve to loopback (nip.io)", cmdAll, cmdLab, cmdRun)
}
