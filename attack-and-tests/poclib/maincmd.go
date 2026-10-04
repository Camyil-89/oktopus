package poclib

import (
	"fmt"
	"os"
)

// MainCLI — стандартные подкоманды all | lab | run.
func MainCLI(title string, allFn func([]string) int, labFn func([]string) int, runFn func([]string) int) {
	if len(os.Args) < 2 {
		usage(title)
		os.Exit(2)
	}
	arg0 := os.Args[1]
	if arg0 != "" && arg0[0] == '-' {
		os.Exit(allFn(os.Args[1:]))
	}
	switch arg0 {
	case "all":
		os.Exit(allFn(os.Args[2:]))
	case "lab":
		os.Exit(labFn(os.Args[2:]))
	case "run":
		os.Exit(runFn(os.Args[2:]))
	default:
		usage(title)
		os.Exit(2)
	}
}

func usage(title string) {
	fmt.Fprintf(os.Stderr, `%s

  go run ./attack-and-tests/<name> all
  go run ./attack-and-tests/<name> lab
  go run ./attack-and-tests/<name> run [flags]

`, title)
}
