package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("Plain HTTP absolute URI vs Host", cmdAll, cmdLab, cmdRun)
}
