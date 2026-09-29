package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunEvaluateSuite(aclrun.ACLDstDomain(), os.Args[1:]))
}