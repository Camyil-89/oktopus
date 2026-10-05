package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunInspectSuite(aclrun.InspectAllowDeny(), os.Args[1:]))
}
