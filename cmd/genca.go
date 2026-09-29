package main

import (
	"flag"
	"fmt"
	"os"

	"oktopus/internal/pki"
)

func runGenca(args []string) int {
	fs := flag.NewFlagSet("genca", flag.ExitOnError)
	out := fs.String("out", "certs", "каталог для ca.crt и ca.key")
	cn := fs.String("cn", "Oktopus Proxy CA", "Common Name корневого CA")
	org := fs.String("org", "Oktopus", "Organization")
	country := fs.String("country", "RU", "страна (2 буквы)")
	days := fs.Int("days", 3650, "срок действия сертификата в днях")
	keyBits := fs.Int("key-bits", 4096, "размер RSA ключа (>= 2048)")
	force := fs.Bool("force", false, "перезаписать существующие файлы")

	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	err := pki.GenerateCA(pki.CAConfig{
		CommonName:   *cn,
		Organization: *org,
		Country:      *country,
		ValidDays:    *days,
		KeyBits:      *keyBits,
		OutDir:       *out,
		Force:        *force,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "genca: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Создано:\n  %s\n  %s\n", *out+"/ca.crt", *out+"/ca.key")
	fmt.Fprintln(os.Stdout, "Установите ca.crt в доверенные корневые ЦС на клиентах. ca.key храните только на сервере прокси.")
	return 0
}
