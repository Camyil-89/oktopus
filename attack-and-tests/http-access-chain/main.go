package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunEvaluateSuite(aclrun.HTTPAccessChain(), os.Args[1:]))
}