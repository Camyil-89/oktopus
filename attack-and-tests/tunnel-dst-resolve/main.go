// PoC: tunnel CONNECT по разрешённому домену, резолв в 127.0.0.0/8 — dst без DNS (vs Squid).
//
//	go run ./attack-and-tests/tunnel-dst-resolve all
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	arg0 := os.Args[1]
	if arg0 != "" && arg0[0] == '-' {
		os.Exit(cmdAll(os.Args[1:]))
	}
	switch arg0 {
	case "all":
		os.Exit(cmdAll(os.Args[2:]))
	case "lab":
		os.Exit(cmdLab(os.Args[2:]))
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Tunnel: dstdomain allow + dst deny при резолве в loopback

  go run ./attack-and-tests/tunnel-dst-resolve all

  API setup + lab + mitm/tunnel + RESULT OK/FAIL

`)
}
