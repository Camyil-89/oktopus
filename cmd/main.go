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
	case "genca":
		os.Exit(runGenca(os.Args[2:]))
	case "proxy":
		os.Exit(runProxy(os.Args[2:]))
	case "bench":
		os.Exit(runBench(os.Args[2:]))
	case "load":
		os.Exit(runProxyLoad(os.Args[2:]))
	case "db":
		os.Exit(runDB(os.Args[2:]))
	case "api":
		os.Exit(runAPI(os.Args[2:]))
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `oktopus — прокси с MITM (в разработке)

Использование:
  oktopus <команда> [флаги]

Команды:
  genca    сгенерировать корневой CA (ca.crt, ca.key)
  proxy    HTTP forward-прокси (логирование http://)
  bench    бенчмарки go test в виде таблицы
  load     нагрузочный прогон GET через прокси
  db       PostgreSQL: миграции goose и bootstrap admin
  api      HTTP API (JWT в cookie, :8000)
  serve    прокси + API в одном процессе

Пример:
  go run ./cmd genca -out certs
  go run ./cmd proxy
  go run ./cmd proxy -auth=false
  go run ./cmd proxy -connect tunnel
  go run ./cmd proxy -connect mitm -ca-cert certs/ca.crt
  go run ./cmd bench
  go run ./cmd bench -integration
  go run ./cmd bench -count=5 -benchtime=500ms
  go run ./cmd load -url=https://example.com/ -workers=10 -duration=10s
  go run ./cmd load -url=https://example.com/ -proxy-user=user -proxy-password=user
  go run ./cmd load -url=https://example.com/ -proxy-auth=user2:user2
  go run ./cmd db up
  go run ./cmd api
  go run ./cmd serve

`)
}
