package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func runProxyLoad(args []string) int {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	urlFlag := fs.String("url", "", "URL для GET через прокси (обязательно)")
	workers := fs.Int("workers", 10, "число параллельных воркеров")
	proxyAddr := fs.String("proxy", "127.0.0.1:8080", "адрес HTTP-прокси") // 12.0.0.9:8080
	proxyAuth := fs.String("proxy-auth", "", "login:password (приоритет над proxy-user); пусто = собрать из proxy-user")
	proxyUser := fs.String("proxy-user", "user", "логин Proxy-Authorization (dev LDAP: user, user2)")
	proxyPass := fs.String("proxy-password", "user", "пароль для proxy-user")
	duration := fs.Duration("duration", 10*time.Second, "длительность нагрузки")
	ramp := fs.Duration("ramp", 2000*time.Millisecond, "растянуть старт воркеров (снижает отказы accept на прокси)")
	maxConnsFlag := fs.Int("max-conns", 5000, "лимит одновременных соединений к прокси/хосту (0 = min(workers,64); снижает connectex на Windows)")
	targetRPS := fs.Int("target-rps", 0, "глобальный лимит попыток/с (0 = без лимита; на Windows при больших workers лучше 800–1200)")
	pingSamples := fs.Int("ping-samples", 20, "после нагрузки: число последовательных GET для сравнения direct vs proxy (0 = пропустить)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "load: укажите -url, например:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd load -url=https://example.com/ -workers=10")
		fmt.Fprintln(os.Stderr, "  go run ./cmd load -url=https://example.com/ -proxy-user=user2 -proxy-password=user2")
		return 2
	}
	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "load: workers must be >= 1")
		return 2
	}
	if runtime.GOOS == "windows" && *targetRPS == 0 && *workers > 32 {
		log.Printf("load: на Windows при workers=%d без -target-rps возможны TIME_WAIT и исчерпание портов; рекомендуется -target-rps=1000 -max-conns=64", *workers)
	}

	proxyURL, err := url.Parse("http://" + *proxyAddr)
	if err != nil {
		log.Printf("load: proxy: %v", err)
		return 1
	}
	authCreds, err := resolveProxyCredentials(*proxyAuth, *proxyUser, *proxyPass)
	if err != nil {
		log.Printf("load: %v", err)
		return 2
	}
	if err := applyProxyAuth(proxyURL, authCreds); err != nil {
		log.Printf("load: %v", err)
		return 2
	}

	maxConns := loadMaxConns(*workers, *maxConnsFlag)

	client := newLoadHTTPClient(proxyURL, maxConns)

	probeDeny, probeErr := probeGET(client, *urlFlag)
	if probeErr != nil {
		log.Printf("load: probe через прокси %s: %v", *proxyAddr, probeErr)
		return 1
	}
	if probeDeny && *targetRPS == 0 {
		log.Printf("load: probe deny — каждый HTTPS-запрос = новое TCP к прокси; без -target-rps на Windows часто всплеск RPS и пауза (TIME_WAIT)")
	}

	pacer := newLoadPacer(*targetRPS)

	fmt.Fprintf(os.Stdout, "load: proxy=%s url=%s workers=%d max_conns=%d target_rps=%d duration=%s",
		*proxyAddr, *urlFlag, *workers, maxConns, *targetRPS, *duration)
	if authCreds != "" {
		user, _, _ := strings.Cut(authCreds, ":")
		fmt.Fprintf(os.Stdout, " proxy_auth_user=%q", user)
	}
	fmt.Fprintln(os.Stdout)

	ok, fail := doProxyLoad(client, *urlFlag, *workers, *duration, *ramp, pacer)
	fmt.Fprintf(os.Stdout, "load: done ok=%d fail=%d avg_rps=%.0f\n",
		ok, fail, float64(ok)/duration.Seconds())

	if ok == 0 {
		log.Printf("load: ни одного успешного запроса (fail=%d)", fail)
		return 1
	}
	if fail > 0 {
		log.Printf("load: предупреждение: fail=%d (часто connectex/TIME_WAIT на Windows — уменьшите workers или -max-conns)", fail)
	}
	if *pingSamples > 0 {
		runLoadPingCompare(*urlFlag, client, *pingSamples)
	}
	return 0
}

func newLoadHTTPClient(proxyURL *url.URL, maxConns int) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:          maxConns * 2,
		MaxIdleConnsPerHost:   maxConns,
		MaxConnsPerHost:       maxConns,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	if proxyURL != nil {
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: tr, Timeout: 30 * time.Second}
}

type loadLatencyStats struct {
	samples int
	fail    int
	min     time.Duration
	max     time.Duration
	avg     time.Duration
}

func runLoadPingCompare(resourceURL string, proxyClient *http.Client, samples int) {
	directClient := newLoadHTTPClient(nil, 1)
	fmt.Fprintf(os.Stdout, "load: ping (1 goroutine, n=%d, warmup=1 each path)\n", samples)
	direct := measureLoadLatency(directClient, resourceURL, samples)
	viaProxy := measureLoadLatency(proxyClient, resourceURL, samples)
	printLoadLatencyLine("direct", direct)
	printLoadLatencyLine("proxy", viaProxy)
	if direct.samples > 0 && viaProxy.samples > 0 {
		delta := viaProxy.avg - direct.avg
		fmt.Fprintf(os.Stdout, "load: ping delta avg=%s (proxy−direct)\n", delta.Round(time.Microsecond))
	}
}

func printLoadLatencyLine(label string, s loadLatencyStats) {
	if s.samples == 0 {
		fmt.Fprintf(os.Stdout, "load: ping %-6s fail=%d (нет успешных замеров)\n", label, s.fail)
		return
	}
	fmt.Fprintf(os.Stdout, "load: ping %-6s ok=%d fail=%d min=%s avg=%s max=%s\n",
		label, s.samples, s.fail,
		s.min.Round(time.Microsecond), s.avg.Round(time.Microsecond), s.max.Round(time.Microsecond))
}

func measureLoadLatency(client *http.Client, resourceURL string, samples int) loadLatencyStats {
	loadLatencyWarmup(client, resourceURL)
	var total time.Duration
	var st loadLatencyStats
	st.min = time.Hour
	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := client.Get(resourceURL)
		if err != nil {
			if loadErrIsProxyDeny(err) {
				d := time.Since(start)
				loadLatencyRecord(&st, &total, d)
				continue
			}
			st.fail++
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if !loadStatusOK(resp.StatusCode) {
			st.fail++
			continue
		}
		d := time.Since(start)
		loadLatencyRecord(&st, &total, d)
	}
	if st.samples > 0 {
		st.avg = total / time.Duration(st.samples)
	} else {
		st.min = 0
	}
	return st
}

func loadLatencyWarmup(client *http.Client, resourceURL string) {
	resp, err := client.Get(resourceURL)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func loadLatencyRecord(st *loadLatencyStats, total *time.Duration, d time.Duration) {
	st.samples++
	*total += d
	if d < st.min {
		st.min = d
	}
	if d > st.max {
		st.max = d
	}
}

func loadMaxConns(workers, flag int) int {
	if flag > 0 {
		return flag
	}
	if workers < 1 {
		return 1
	}
	if workers > 64 {
		return 64
	}
	return workers
}

// probeGET проверяет доступность прокси. denyACL=true — ожидаем 403/Forbidden (нагрузка всё равно идёт).
func probeGET(client *http.Client, resourceURL string) (denyACL bool, err error) {
	const maxAttempts = 25
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
		}
		resp, err := client.Get(resourceURL)
		if err != nil {
			if loadErrIsProxyDeny(err) {
				log.Printf("load: probe: deny (%v) — нагрузка продолжится для проверки ACL", err)
				return true, nil
			}
			if loadErrRetryable(err) {
				lastErr = err
				continue
			}
			return false, err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden {
			log.Printf("load: probe: 403 Forbidden (deny) — нагрузка продолжится для проверки ACL")
			return true, nil
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			if resp.StatusCode == http.StatusProxyAuthRequired {
				return false, fmt.Errorf("status 407: укажите -proxy-user/-proxy-password или -proxy-auth=login:password (LDAP dev: user:user)")
			}
			return false, fmt.Errorf("status %d", resp.StatusCode)
		}
		return false, nil
	}
	return false, lastErr
}

type loadPacer struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func newLoadPacer(targetRPS int) *loadPacer {
	if targetRPS <= 0 {
		return nil
	}
	return &loadPacer{interval: time.Second / time.Duration(targetRPS)}
}

func (p *loadPacer) Wait() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if now.Before(p.next) {
		time.Sleep(p.next.Sub(now))
		now = time.Now()
	}
	p.next = now.Add(p.interval)
}

// loadStatusOK — ответ прокси, пригодный для нагрузочного прогона (в т.ч. deny 403).
func loadStatusOK(code int) bool {
	if code >= 200 && code < 400 {
		return true
	}
	return code == http.StatusForbidden
}

// loadErrIsProxyDeny — deny ACL на CONNECT/HTTPS: net/http часто даёт err, а не resp 403.
func loadErrIsProxyDeny(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Forbidden") || strings.Contains(msg, "403")
}

// applyProxyAuth задаёт Basic для прокси (Go шлёт Proxy-Authorization на CONNECT и http://).
func applyProxyAuth(proxyURL *url.URL, auth string) error {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return nil
	}
	user, pass, ok := strings.Cut(auth, ":")
	if !ok || user == "" {
		return fmt.Errorf("proxy-auth: ожидается login:password, got %q", auth)
	}
	proxyURL.User = url.UserPassword(user, pass)
	return nil
}

// resolveProxyCredentials: proxy-auth имеет приоритет; иначе proxy-user:proxy-password.
// Пустые user и password (и proxy-auth) — без авторизации (прокси с -auth=false).
func resolveProxyCredentials(authFlag, user, password string) (string, error) {
	authFlag = strings.TrimSpace(authFlag)
	if authFlag != "" {
		return authFlag, nil
	}
	user = strings.TrimSpace(user)
	password = strings.TrimSpace(password)
	if user == "" && password == "" {
		return "", nil
	}
	if user == "" {
		return "", fmt.Errorf("proxy-user: логин не задан")
	}
	return user + ":" + password, nil
}

func doProxyLoad(client *http.Client, resourceURL string, workers int, duration time.Duration, ramp time.Duration, pacer *loadPacer) (ok, fail int64) {
	var okN, failN atomic.Int64
	deadline := time.Now().Add(duration)
	sec := int(duration.Seconds())
	if sec < 1 {
		sec = 1
	}

	var lastOk int64
	var statsWG sync.WaitGroup
	statsWG.Add(1)
	go func() {
		defer statsWG.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for tick := 1; tick <= sec; tick++ {
			<-ticker.C
			okNow := okN.Load()
			failNow := failN.Load()
			rps1s := okNow - lastOk
			lastOk = okNow
			fmt.Fprintf(os.Stdout, "[%d/%ds] ok=%d fail=%d rps_1s=%d\n",
				tick, sec, okNow, failNow, rps1s)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			if ramp > 0 && workers > 1 {
				time.Sleep(time.Duration(id) * ramp / time.Duration(workers))
			}
			for !proxyLoadOne(client, resourceURL, pacer) {
				if time.Now().After(deadline) {
					failN.Add(1)
					return
				}
			}
			for time.Now().Before(deadline) {
				if proxyLoadOne(client, resourceURL, pacer) {
					okN.Add(1)
				} else {
					failN.Add(1)
				}
			}
		}(i)
	}
	wg.Wait()
	statsWG.Wait()

	return okN.Load(), failN.Load()
}

func proxyLoadOne(client *http.Client, resourceURL string, pacer *loadPacer) bool {
	const maxAttempts = 12
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 25 * time.Millisecond)
		}
		ok, retry := proxyLoadOneAttempt(client, resourceURL, pacer)
		if ok {
			return true
		}
		if !retry {
			return false
		}
	}
	return false
}

func proxyLoadOneAttempt(client *http.Client, resourceURL string, pacer *loadPacer) (ok, retry bool) {
	pacer.Wait()
	resp, err := client.Get(resourceURL)
	if err != nil {
		if loadErrIsProxyDeny(err) {
			return true, false
		}
		return false, loadErrRetryable(err)
	}
	if !loadStatusOK(resp.StatusCode) {
		resp.Body.Close()
		if resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
			return false, true
		}
		if resp.StatusCode == http.StatusProxyAuthRequired {
			return false, false
		}
		return false, false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return true, false
}

func loadErrRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "actively refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporary failure") ||
		strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "Only one usage of each socket address") ||
		strings.Contains(msg, "proxyconnect") ||
		strings.Contains(msg, "no buffer space available")
}
