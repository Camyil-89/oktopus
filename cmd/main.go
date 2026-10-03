package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "load":
		os.Exit(runProxyLoad(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `oktopus

Использование:
  go run ./cmd <команда> [флаги]

Команды:
  serve    прокси + API (миграции БД при старте)
  load     нагрузочный GET через HTTP-прокси

Примеры:
  go run ./cmd serve
  go run ./cmd load -url=https://example.com/ -workers=10 -duration=10s

`)
}
