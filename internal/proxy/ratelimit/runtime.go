package ratelimit

import (
	"net"
	"sync"
	"time"
)

const refillInterval = 100 * time.Millisecond

// Runtime хранит состояние bucket'ов для скомпилированных pool.
type Runtime struct {
	pools []poolState
	stop  chan struct{}
	wg    sync.WaitGroup
}

type poolState struct {
	def       PoolDef
	aggregate bucket
	byKey     sync.Map
}

func NewRuntime(defs []PoolDef) *Runtime {
	if len(defs) <= 1 {
		return nil
	}
	rt := &Runtime{
		pools: make([]poolState, len(defs)),
		stop:  make(chan struct{}),
	}
	hasAggRefill := false
	for i := 1; i < len(defs); i++ {
		rt.pools[i].def = defs[i]
		if defs[i].Class == 2 && defs[i].AggLimited() {
			hasAggRefill = true
		}
	}
	if hasAggRefill {
		rt.wg.Add(1)
		go rt.refillLoop()
	}
	return rt
}

func (rt *Runtime) refillLoop() {
	defer rt.wg.Done()
	t := time.NewTicker(refillInterval)
	defer t.Stop()
	for {
		select {
		case <-rt.stop:
			return
		case <-t.C:
			rt.refillClass2Agg(refillInterval)
		}
	}
}

func (rt *Runtime) refillClass2Agg(dt time.Duration) {
	for i := range rt.pools {
		ps := &rt.pools[i]
		if ps.def.Class == 2 && ps.def.AggLimited() {
			ps.aggregate.topUp(ps.def.aggLimits(), dt)
		}
	}
}

type Flow struct {
	rt    *Runtime
	pool  int
	srcIP net.IP
}

func (f *Flow) Acquire(n int) error {
	if f == nil || f.rt == nil || n <= 0 {
		return nil
	}
	return f.rt.acquire(f.pool, f.srcIP, int64(n))
}

func (rt *Runtime) Flow(pool int, srcIP net.IP) *Flow {
	if rt == nil || pool <= 0 || pool >= len(rt.pools) {
		return nil
	}
	def := rt.pools[pool].def
	if def.Unlimited() {
		return nil
	}
	return &Flow{rt: rt, pool: pool, srcIP: srcIP}
}

func (rt *Runtime) acquire(pool int, srcIP net.IP, n int64) error {
	if pool <= 0 || pool >= len(rt.pools) {
		return nil
	}
	ps := &rt.pools[pool]
	def := ps.def

	waitAgg := func(lim rateLimits, key string) error {
		if lim.unlimited() {
			return nil
		}
		if key == "" {
			return ps.aggregate.wait(n, lim)
		}
		b := ps.loadKey(key, lim.initial)
		return b.wait(n, lim)
	}

	switch def.Class {
	case 2:
		if err := waitAgg(def.aggLimits(), ""); err != nil {
			return err
		}
		if def.IndLimited() {
			return waitAgg(def.indLimits(), "i:"+ipBucketKey(srcIP))
		}
		return nil
	case 3:
		if def.AggLimited() {
			if err := waitAgg(def.aggLimits(), "n:"+networkBucketKey(srcIP)); err != nil {
				return err
			}
		}
		if def.IndLimited() {
			return waitAgg(def.indLimits(), "i:"+ipBucketKey(srcIP))
		}
		return nil
	default:
		return waitAgg(def.indLimits(), "i:"+ipBucketKey(srcIP))
	}
}

func (ps *poolState) loadKey(key string, initial int64) *bucket {
	if v, ok := ps.byKey.Load(key); ok {
		return v.(*bucket)
	}
	b := newBucket(initial)
	actual, _ := ps.byKey.LoadOrStore(key, b)
	return actual.(*bucket)
}

func ipBucketKey(ip net.IP) string {
	if ip == nil {
		return "unknown"
	}
	return ip.String()
}

func networkBucketKey(ip net.IP) string {
	if ip == nil {
		return "unknown"
	}
	ip = ip.To16()
	if ip == nil {
		return "unknown"
	}
	if v4 := ip.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String() + "/24"
	}
	return net.IP(append([]byte(nil), ip[0:8]...)).String() + "/64"
}
