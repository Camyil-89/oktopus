package benchfmt

import (
	"bytes"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	sample := `goos: windows
goarch: amd64
pkg: oktopus/internal/proxy/http/test
cpu: Intel CPU
BenchmarkForwarderServe-6    17690    67348 ns/op    39062 B/op    76 allocs/op
BenchmarkCopyHeaders-6     3447692      346.3 ns/op      80 B/op     4 allocs/op
`
	meta, rows := Parse([]byte(sample))
	if meta.GOOS != "windows" || meta.Pkg != "oktopus/internal/proxy/http/test" {
		t.Fatalf("meta: %+v", meta)
	}
	if len(rows) != 2 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Name != "ForwarderServe" || rows[0].AllocsPerOp != 76 {
		t.Fatalf("row0: %+v", rows[0])
	}
	if rows[1].NsPerOp != 346.3 {
		t.Fatalf("row1 ns: %v", rows[1].NsPerOp)
	}

	var buf bytes.Buffer
	WriteTable(&buf, meta, rows)
	if !strings.Contains(buf.String(), "ForwarderServe") {
		t.Fatalf("table: %s", buf.String())
	}
}
