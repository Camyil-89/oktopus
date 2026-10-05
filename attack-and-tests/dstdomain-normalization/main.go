package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("dstdomain case / trailing dot", cmdAll, cmdLab, cmdRun)
}
