package main

import (
	"context"

	"oktopus/internal/db/proxysettings/service"
	"oktopus/internal/proxy"
)

type proxyHotReloader struct {
	settings *service.Service
	mgr      *proxy.Manager
}

func (r *proxyHotReloader) ReloadProxy(ctx context.Context) error {
	st, err := r.settings.Get(ctx)
	if err != nil {
		return err
	}
	if !st.ProxyEnabled {
		r.mgr.Stop()
		return nil
	}
	rt, err := r.settings.LoadProxyRuntime(ctx)
	if err != nil {
		return err
	}
	return r.mgr.Apply(ctx, rt.Config, rt.ACLEngine, rt.InspectRunner)
}
