package main

import "oktopus/attack-and-tests/poclib"

func main() {
	poclib.MainCLI("MITM HTTP smuggling (CL.TE probe)", cmdAll, cmdLab, cmdRun)
}
