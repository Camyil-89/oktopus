// PoC: CONNECT/Host разрешён, TLS ClientHello SNI — запрещённый origin.
//
//	go run ./attack-and-tests/mitm-allowed-host-evil-sni all
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
	fmt.Fprintf(os.Stderr, `MITM: allowed Host, forbidden TLS SNI

  go run ./attack-and-tests/mitm-allowed-host-evil-sni all
  go run ./attack-and-tests/mitm-allowed-host-evil-sni lab
  go run ./attack-and-tests/mitm-allowed-host-evil-sni run [flags]

См. README.md
`)
}
