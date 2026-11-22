package squid_test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
	"oktopus/internal/proxy/ratelimit"
)

func TestCompileDelayPools(t *testing.T) {
	cfg := `
acl all all
acl staff src 10.0.0.0/8

delay_pools 2
delay_class 1 1
delay_class 2 2
delay_parameters 1 8000/16000
delay_parameters 2 64000/64000
delay_access 2 allow staff
delay_access 1 allow all

http_access allow all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := auth.Identity{}
	staff := acl.RequestFields{SNI: "x", DstPort: 443, SrcIP: net.ParseIP("10.1.2.3")}
	other := acl.RequestFields{SNI: "x", DstPort: 443, SrcIP: net.ParseIP("203.0.113.1")}

	fStaff := engine.DelayFlow(id, staff, acl.SquidPhaseHTTP)
	if fStaff == nil {
		t.Fatal("expected staff pool")
	}
	fOther := engine.DelayFlow(id, other, acl.SquidPhaseHTTP)
	if fOther == nil {
		t.Fatal("expected default pool for other")
	}
}

func TestDelayRateLimit(t *testing.T) {
	cfg := `
acl all all
delay_pools 1
delay_class 1 1
delay_parameters 1 10000/10000
delay_access 1 allow all
http_access allow all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := engine.DelayFlow(auth.Identity{}, acl.RequestFields{SNI: "x", DstPort: 443, SrcIP: net.ParseIP("127.0.0.1")}, acl.SquidPhaseHTTP)
	if f == nil {
		t.Fatal("expected flow")
	}
	start := time.Now()
	if err := f.Acquire(20000); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 800*time.Millisecond {
		t.Fatalf("expected ~1s throttle for 20KiB at 10KiB/s, got %v", d)
	}
}

func TestDelayACLListLimitsUploadAndDownload(t *testing.T) {
	tests := []struct {
		name     string
		transfer func(t *testing.T, f *ratelimit.Flow) time.Duration
	}{
		{name: "download", transfer: func(t *testing.T, f *ratelimit.Flow) time.Duration {
			payload := bytes.Repeat([]byte("d"), 4<<10)
			start := time.Now()
			got, err := io.ReadAll(f.Reader(bytes.NewReader(payload)))
			if err != nil {
				t.Fatalf("read limited download: %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("download body mismatch: got %d bytes, want %d", len(got), len(payload))
			}
			return time.Since(start)
		}},
		{name: "upload", transfer: func(t *testing.T, f *ratelimit.Flow) time.Duration {
			payload := bytes.Repeat([]byte("u"), 4<<10)
			var dst bytes.Buffer
			start := time.Now()
			n, err := f.Writer(&dst).Write(payload)
			if err != nil {
				t.Fatalf("write limited upload: %v", err)
			}
			if n != len(payload) || !bytes.Equal(dst.Bytes(), payload) {
				t.Fatalf("upload body mismatch: wrote %d bytes, got %d, want %d", n, dst.Len(), len(payload))
			}
			return time.Since(start)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := `
acl all all
acl throttled src 127.0.0.1
delay_pools 1
delay_class 1 1
delay_parameters 1 65536/1024
delay_access 1 allow throttled
http_access allow all
`
			engine, err := squid.Compile(cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			flow := engine.DelayFlow(auth.Identity{}, acl.RequestFields{
				SNI: "transfer.test", DstPort: 443, SrcIP: net.ParseIP("127.0.0.1"),
			}, acl.SquidPhaseHTTP)
			if flow == nil {
				t.Fatal("delay_access ACL list did not select a rate-limit pool")
			}
			if unmatched := engine.DelayFlow(auth.Identity{}, acl.RequestFields{
				SNI: "transfer.test", DstPort: 443, SrcIP: net.ParseIP("192.0.2.1"),
			}, acl.SquidPhaseHTTP); unmatched != nil {
				t.Fatal("non-matching ACL list unexpectedly selected a rate-limit pool")
			}

			elapsed := tt.transfer(t, flow)
			// delay_parameters gives a 512-byte initial burst; the remaining
			// 3.5 KiB at 64 KiB/s should take about 55 ms.
			if elapsed < 25*time.Millisecond {
				t.Fatalf("ACL-selected rate limit did not delay transfer enough: %v", elapsed)
			}
			if elapsed > time.Second {
				t.Fatalf("ACL-selected rate limit delayed 4 KiB unexpectedly long: %v", elapsed)
			}
		})
	}
}

func TestDelayParametersNoneIndividual(t *testing.T) {
	cfg := `
acl all all
delay_pools 1
delay_class 1 2
delay_parameters 1 none 1048576/2097152
delay_access 1 allow all
http_access allow all
`
	engine, err := squid.Compile(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := engine.DelayFlow(auth.Identity{}, acl.RequestFields{
		SNI: "x", DstPort: 443, SrcIP: net.ParseIP("192.168.1.10"),
	}, acl.SquidPhaseHTTP)
	if f == nil {
		t.Fatal("expected flow")
	}
}
