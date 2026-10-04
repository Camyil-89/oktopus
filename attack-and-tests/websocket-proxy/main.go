// Интеграция: WebSocket через oktopus (wss в MITM и ws в plain HTTP proxy).
//
//	go run ./attack-and-tests/websocket-proxy all
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
	fmt.Fprintf(os.Stderr, `WebSocket через прокси (lab + проверка)

  go run ./attack-and-tests/websocket-proxy all
      API setup + lab + wss/ws в mitm и tunnel + RESULT OK/FAIL

  go run ./attack-and-tests/websocket-proxy lab
  go run ./attack-and-tests/websocket-proxy run [flags]

См. README.md
`)
}
