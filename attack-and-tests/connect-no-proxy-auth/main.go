package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("CONNECT without proxy-auth", cmdAll, cmdLab, cmdRun)
}
