// PoC: TLS SNI разрешён, HTTP Host — запрещённый origin.
//
//	go run ./attack-and-tests/mitm-allowed-sni-evil-host all
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
	fmt.Fprintf(os.Stderr, `MITM: allowed TLS SNI, forbidden HTTP Host

  go run ./attack-and-tests/mitm-allowed-sni-evil-host all
  go run ./attack-and-tests/mitm-allowed-sni-evil-host lab
  go run ./attack-and-tests/mitm-allowed-sni-evil-host run [flags]

См. README.md
`)
}
