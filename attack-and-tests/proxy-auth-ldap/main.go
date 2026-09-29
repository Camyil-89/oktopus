package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunProxyAuthSuite(aclrun.ProxyAuthLDAP(), os.Args[1:]))
}
