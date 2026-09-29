package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"oktopus/internal/benchfmt"
)

func runBench(args []string) int {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	pattern := fs.String("bench", "Benchmark", "шаблон -bench (как у go test)")
	count := fs.Int("count", 1, "число прогонов -count")
	benchtime := fs.String("benchtime", "", "длительность каждого бенчмарка (например 500ms); пусто = по умолчанию go test")
	integration := fs.Bool("integration", false, "включить медленные CONNECT/MITM бенчмарки (без -short)")
	extra := fs.String("args", "", "доп. аргументы go test (через пробел, в кавычках)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	pkgs := fs.Args()
	if len(pkgs) == 0 {
		pkgs = []string{
			"./internal/proxy/acl/test",
			"./internal/proxy/http/test",
			"./internal/proxy/https/test",
		}
	}

	if !*integration {
		fmt.Fprintln(os.Stderr, "bench: быстрый режим (-short): CONNECT/MITM бенчмарки пропущены; полный прогон: go run ./cmd bench -integration")
	} else {
		fmt.Fprintln(os.Stderr, "bench: полный режим (-integration): возможны долгие CONNECT/MITM бенчмарки")
	}

	var allRows []benchfmt.Row
	var meta benchfmt.Meta
	var outBuf bytes.Buffer

	for i, pkg := range pkgs {
		fmt.Fprintf(os.Stderr, "bench: [%d/%d] %s\n", i+1, len(pkgs), pkg)
		fmt.Fprintln(os.Stderr, "bench:   запуск go test …")

		testArgs := []string{
			"test",
			"-bench=" + *pattern,
			"-benchmem",
			"-run=DoesNotExist",
			fmt.Sprintf("-count=%d", *count),
		}
		if *benchtime != "" {
			testArgs = append(testArgs, "-benchtime="+*benchtime)
		}
		if !*integration {
			testArgs = append(testArgs, "-short")
		}
		if s := strings.TrimSpace(*extra); s != "" {
			testArgs = append(testArgs, strings.Fields(s)...)
		}
		testArgs = append(testArgs, pkg)

		start := time.Now()
		out, err := runGoTestWithProgress(testArgs)
		elapsed := time.Since(start).Round(time.Millisecond)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bench:   ошибка за %s: %v\n", elapsed, err)
			if len(out) > 0 {
				os.Stderr.Write(out)
			}
			return 1
		}
		fmt.Fprintf(os.Stderr, "bench:   пакет завершён за %s\n", elapsed)

		outBuf.Write(out)
		m, rows := benchfmt.Parse(out)
		if meta.GOOS == "" {
			meta = m
		}
		allRows = append(allRows, rows...)
	}

	if len(allRows) == 0 {
		fmt.Fprintln(os.Stderr, "bench: не найдено строк бенчмарка (проверьте -bench и пакеты)")
		os.Stdout.Write(outBuf.Bytes())
		return 1
	}

	fmt.Fprintln(os.Stderr, "bench: сводная таблица:")
	benchfmt.WriteTable(os.Stdout, meta, allRows)
	return 0
}

func runGoTestWithProgress(testArgs []string) ([]byte, error) {
	cmd := exec.Command("go", testArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		teeLines(stderr, &out, true)
	}()
	go func() {
		defer wg.Done()
		teeLines(stdout, &out, false)
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return out.Bytes(), err
	}
	return out.Bytes(), nil
}

func teeLines(r io.Reader, collect *bytes.Buffer, logBuild bool) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		collect.WriteString(line)
		collect.WriteByte('\n')
		if name := benchfmt.BenchLineName(line); name != "" {
			fmt.Fprintf(os.Stderr, "bench:   · %s\n", name)
			continue
		}
		if !logBuild {
			continue
		}
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "ok ") || strings.HasPrefix(trim, "FAIL") {
			fmt.Fprintf(os.Stderr, "bench:   | %s\n", trim)
		}
	}
}
