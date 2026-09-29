package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunCompileSuite(aclrun.DelayAccess(), os.Args[1:]))
}