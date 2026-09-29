package main

import (
	"os"

	"oktopus/attack-and-tests/aclrun"
)

func main() {
	os.Exit(aclrun.RunProxyAuthSuite(aclrun.ProxyAuthStatic(), os.Args[1:]))
}
