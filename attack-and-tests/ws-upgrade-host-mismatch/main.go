package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("WebSocket upgrade host mismatch", cmdAll, cmdLab, cmdRun)
}
