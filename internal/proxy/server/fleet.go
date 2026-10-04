package server

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"

	proxyinstanceservice "oktopus/internal/db/proxyinstances/service"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/metrics"
)

// Fleet — несколько Manager (по одному на инстанс).
type Fleet struct {
	hooks     *hooks.Hooks
	log       *log.Logger
	accessRec accesslog.Recorder

	mu       sync.Mutex
	managers map[uuid.UUID]*Manager
}

func NewFleet(h *hooks.Hooks, logger *log.Logger) *Fleet {
	return &Fleet{
		hooks:    h,
		log:      logger,
		managers: make(map[uuid.UUID]*Manager),
	}
}

func (f *Fleet) SetAccessRecorder(rec accesslog.Recorder) {
	f.accessRec = rec
}

type FleetRuntimeLoader func(ctx context.Context) ([]proxyinstanceservice.InstanceRuntime, error)

func (f *Fleet) ApplyFleet(ctx context.Context, runtimes []proxyinstanceservice.InstanceRuntime) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	want := make(map[uuid.UUID]bool, len(runtimes))
	for _, rt := range runtimes {
		want[rt.InstanceID] = true
		m := f.managers[rt.InstanceID]
		if m == nil {
			m = NewManager(f.hooks, f.log)
			m.SetInstanceID(rt.InstanceID)
			if f.accessRec != nil {
				m.SetAccessRecorder(stampingRecorder{instanceID: rt.InstanceID, inner: f.accessRec})
			}
			f.managers[rt.InstanceID] = m
		}
		if err := m.Apply(ctx, rt.Config, rt.ACLEngine, rt.InspectRunner); err != nil {
			if f.log != nil {
				f.log.Printf("proxy: instance %s: %v", rt.InstanceID, err)
			}
			continue
		}
	}
	for id, m := range f.managers {
		if !want[id] {
			m.Stop()
			delete(f.managers, id)
			metrics.DefaultRegistry().Drop(id)
		}
	}
	return nil
}

func (f *Fleet) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.managers {
		m.Stop()
	}
	f.managers = make(map[uuid.UUID]*Manager)
}

func (f *Fleet) RunContextWithLoader(ctx context.Context, load FleetRuntimeLoader) error {
	for {
		if err := ctx.Err(); err != nil {
			f.Stop()
			return nil
		}
		runtimes, err := load(ctx)
		if err != nil {
			if err == ErrRuntimeDeferred {
				f.Stop()
				if !waitFor(ctx, proxyListenRetryInterval) {
					return nil
				}
				continue
			}
			return err
		}
		_ = f.ApplyFleet(ctx, runtimes)
		if !waitFor(ctx, proxyListenRetryInterval) {
			f.Stop()
			return nil
		}
	}
}

type InstanceStatus struct {
	InstanceID      uuid.UUID
	Listen          string
	Active          bool
	LastStartError  string
	Traffic         metrics.Snapshot
}

func (f *Fleet) InstanceStatuses() []InstanceStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]InstanceStatus, 0, len(f.managers))
	for id, m := range f.managers {
		out = append(out, InstanceStatus{
			InstanceID:     id,
			Listen:         m.ProxyListen(),
			Active:         m.ProxyActive(),
			LastStartError: m.ProxyLastStartError(),
			Traffic:        m.ProxyTraffic(),
		})
	}
	return out
}

func (f *Fleet) ProxyActive() bool {
	for _, st := range f.InstanceStatuses() {
		if st.Active {
			return true
		}
	}
	return false
}

func (f *Fleet) AggregateTraffic() metrics.Snapshot {
	statuses := f.InstanceStatuses()
	snaps := make([]metrics.Snapshot, 0, len(statuses))
	for _, st := range statuses {
		snaps = append(snaps, st.Traffic)
	}
	return metrics.AverageSnapshots(snaps)
}

type stampingRecorder struct {
	instanceID uuid.UUID
	inner      accesslog.Recorder
}

func (r stampingRecorder) Record(e accesslog.Entry) {
	e.InstanceID = r.instanceID
	r.inner.Record(e)
}
