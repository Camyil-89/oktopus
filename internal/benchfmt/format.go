package benchfmt

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
)

// Row — одна строка результата go test -bench.
type Row struct {
	Pkg         string
	Name        string
	GOMAX       int
	Iterations  int
	NsPerOp     float64
	BytesPerOp  int
	AllocsPerOp int
}

// Meta — заголовок из вывода go test.
type Meta struct {
	GOOS string
	GOARCH string
	Pkg string
	CPU string
}

var benchLine = regexp.MustCompile(
	`^(Benchmark\w+)-(\d+)\s+(\d+)\s+([\d.]+)\s+ns/op(?:\s+([\d.]+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`,
)

var metaGOOS = regexp.MustCompile(`^goos:\s+(\S+)`)
var metaGOARCH = regexp.MustCompile(`^goarch:\s+(\S+)`)
var metaPkg = regexp.MustCompile(`^pkg:\s+(.+)`)
var metaCPU = regexp.MustCompile(`^cpu:\s+(.+)`)

// Parse разбирает stdout/stderr go test -bench.
func Parse(output []byte) (Meta, []Row) {
	var meta Meta
	var rows []Row
	var currentPkg string
	for line := range bytes.SplitSeq(output, []byte{'\n'}) {
		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}
		if m := metaGOOS.FindStringSubmatch(s); len(m) == 2 {
			meta.GOOS = m[1]
			continue
		}
		if m := metaGOARCH.FindStringSubmatch(s); len(m) == 2 {
			meta.GOARCH = m[1]
			continue
		}
		if m := metaPkg.FindStringSubmatch(s); len(m) == 2 {
			currentPkg = m[1]
			meta.Pkg = m[1]
			continue
		}
		if m := metaCPU.FindStringSubmatch(s); len(m) == 2 {
			meta.CPU = m[1]
			continue
		}
		m := benchLine.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		ns, _ := strconv.ParseFloat(m[4], 64)
		row := Row{
			Pkg:        currentPkg,
			Name:       strings.TrimPrefix(m[1], "Benchmark"),
			GOMAX:      atoi(m[2]),
			Iterations: atoi(m[3]),
			NsPerOp:    ns,
		}
		if m[5] != "" {
			row.BytesPerOp = int(atoiFloat(m[5]))
		}
		if m[6] != "" {
			row.AllocsPerOp = atoi(m[6])
		}
		rows = append(rows, row)
	}
	return meta, rows
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// WriteTable печатает метаданные и таблицу в w.
func WriteTable(w io.Writer, meta Meta, rows []Row) {
	if meta.Pkg != "" || meta.GOOS != "" {
		fmt.Fprintln(w, "Среда:")
		if meta.Pkg != "" {
			fmt.Fprintf(w, "  пакет:  %s\n", meta.Pkg)
		}
		if meta.GOOS != "" || meta.GOARCH != "" {
			fmt.Fprintf(w, "  os/arch: %s/%s\n", meta.GOOS, meta.GOARCH)
		}
		if meta.CPU != "" {
			fmt.Fprintf(w, "  cpu:    %s\n", meta.CPU)
		}
		fmt.Fprintln(w)
	}

	pkgs := uniquePkgOrder(rows)
	multi := len(pkgs) > 1
	for i, pkg := range pkgs {
		if multi {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "── %s ──\n", pkg)
		}
		writeBenchTable(w, filterRowsByPkg(rows, pkg))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "ВРЕМЯ/OP — среднее за одну итерацию цикла бенчмарка; ~RPS — грубая оценка 1e9/ns на одном потоке.")
}

func uniquePkgOrder(rows []Row) []string {
	seen := make(map[string]bool)
	var order []string
	for _, r := range rows {
		p := r.Pkg
		if p == "" {
			p = "."
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		order = append(order, p)
	}
	return order
}

func filterRowsByPkg(rows []Row, pkg string) []Row {
	var out []Row
	for _, r := range rows {
		p := r.Pkg
		if p == "" {
			p = "."
		}
		if p == pkg {
			out = append(out, r)
		}
	}
	return out
}

func writeBenchTable(w io.Writer, rows []Row) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "БЕНЧМАРК\tPROCS\tИТЕРАЦИЙ\tВРЕМЯ/OP\tПАМЯТЬ/OP\tАЛЛОК/OP\t~RPS")
	for _, r := range rows {
		rps := formatRPS(r.NsPerOp)
		_, _ = fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%d\t%s\n",
			r.Name,
			r.GOMAX,
			formatInt(r.Iterations),
			formatDuration(r.NsPerOp),
			formatBytes(r.BytesPerOp),
			r.AllocsPerOp,
			rps,
		)
	}
	_ = tw.Flush()
}

// BenchLineName возвращает имя бенчмарка из строки вывода go test или "".
func BenchLineName(line string) string {
	m := benchLine.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return ""
	}
	return strings.TrimPrefix(m[1], "Benchmark")
}

func formatRPS(nsPerOp float64) string {
	if nsPerOp <= 0 {
		return "—"
	}
	return formatInt64(int64(1e9 / nsPerOp))
}

func formatInt(n int) string {
	return formatInt64(int64(n))
}

func formatInt64(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(byte(c))
	}
	return b.String()
}

// ReportHumanDuration добавляет к выводу go test -bench колонку ms/op, µs/op или ns/op.
func ReportHumanDuration(b *testing.B) {
	if b == nil || b.N == 0 {
		return
	}
	ns := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	switch {
	case ns >= 1e6:
		b.ReportMetric(ns/1e6, "ms/op")
	case ns >= 1e3:
		b.ReportMetric(ns/1e3, "µs/op")
	default:
		b.ReportMetric(ns, "ns/op")
	}
}

func formatDuration(ns float64) string {
	switch {
	case ns >= 1e9:
		return fmt.Sprintf("%.2f s", ns/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.2f ms", ns/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.2f µs", ns/1e3)
	default:
		return fmt.Sprintf("%.0f ns", ns)
	}
}

func formatBytes(b int) string {
	if b == 0 {
		return "—"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	val := float64(b) / float64(div)
	suffix := []string{"KiB", "MiB", "GiB"}[exp]
	return fmt.Sprintf("%.1f %s", val, suffix)
}
