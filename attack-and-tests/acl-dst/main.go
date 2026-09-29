package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunEvaluateSuite(aclrun.ACLDst(), os.Args[1:]))
}