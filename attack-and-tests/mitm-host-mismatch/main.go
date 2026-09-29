// Лаборатория PoC: MITM — ACL по CONNECT/SNI vs фактический HTTP Host.
//
// Одна команда (lab + атака):
//
//	go run ./attack-and-tests/mitm-host-mismatch all
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
	// go run . -proxy ...  → all
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
	fmt.Fprintf(os.Stderr, `MITM host mismatch PoC

  go run ./attack-and-tests/mitm-host-mismatch all
      API setup + lab + mitm/tunnel + RESULT OK/FAIL

  go run ./attack-and-tests/mitm-host-mismatch -api http://127.0.0.1:8000
      то же (all по умолчанию для флагов)

  go run ./attack-and-tests/mitm-host-mismatch lab
  go run ./attack-and-tests/mitm-host-mismatch run [flags]

См. README.md и acl.example.squid
`)
}
