package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("HTTP url_regex through proxy", cmdAll, cmdLab, cmdRun)
}
