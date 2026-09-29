package squid_test

import (
	"net"
	"testing"
	"time"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
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
