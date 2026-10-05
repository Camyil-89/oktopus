package view

import "net/http"

func Register(
	mux *http.ServeMux,
	pages *Handler,
	host *HostSettingsHandler,
	instances *InstancesHandler,
	acl *ACLHandler,
	inspect *InspectHandler,
	status *StatusHandler,
	accessLog *AccessLogHandler,
) {
	mux.HandleFunc("GET /api/proxy/host-settings", host.withAuth(host.Get))
	mux.HandleFunc("PATCH /api/proxy/host-settings", host.withAuth(host.Patch))
	mux.HandleFunc("GET /api/proxy/reports-dashboard", host.withAuth(host.GetReportsDashboard))
	mux.HandleFunc("PATCH /api/proxy/reports-dashboard", host.withAuth(host.PatchReportsDashboard))

	mux.HandleFunc("GET /api/proxy/instances", instances.withAuth(instances.List))
	mux.HandleFunc("POST /api/proxy/instances", instances.withAuth(instances.Create))
	mux.HandleFunc("GET /api/proxy/instances/{id}", instances.withAuth(instances.Get))
	mux.HandleFunc("PATCH /api/proxy/instances/{id}", instances.withAuth(instances.Patch))
	mux.HandleFunc("DELETE /api/proxy/instances/{id}", instances.withAuth(instances.Delete))
	mux.HandleFunc("GET /api/proxy/instances/{id}/ca/status", instances.withAuth(instances.CAStatus))
	mux.HandleFunc("GET /api/proxy/instances/{id}/ca/cert", instances.withAuth(instances.DownloadCert))
	mux.HandleFunc("GET /api/proxy/instances/{id}/ca/key", instances.withAuth(instances.DownloadKey))
	mux.HandleFunc("POST /api/proxy/instances/{id}/ca/generate", instances.withAuth(instances.GenerateCA))
	mux.HandleFunc("POST /api/proxy/instances/{id}/ca/upload", instances.withAuth(instances.UploadCA))

	mux.HandleFunc("GET /api/proxy/forbidden/status", pages.withAuth(pages.ForbiddenStatus))
	mux.HandleFunc("GET /api/proxy/forbidden/preview", pages.withAuth(pages.ForbiddenPreview))
	mux.HandleFunc("POST /api/proxy/forbidden/upload", pages.withAuth(pages.UploadForbiddenPage))
	mux.HandleFunc("DELETE /api/proxy/forbidden/custom", pages.withAuth(pages.ClearForbiddenPage))
	mux.HandleFunc("GET /api/proxy/gateway/status", pages.withAuth(pages.GatewayStatus))
	mux.HandleFunc("GET /api/proxy/gateway/preview", pages.withAuth(pages.GatewayPreview))
	mux.HandleFunc("POST /api/proxy/gateway/upload", pages.withAuth(pages.UploadGatewayPage))
	mux.HandleFunc("DELETE /api/proxy/gateway/custom", pages.withAuth(pages.ClearGatewayPage))

	mux.HandleFunc("GET /api/proxy/status", status.withAuth(status.Get))
	mux.HandleFunc("GET /api/proxy/instances/{id}/acl/status", acl.withAuth(acl.Status))
	mux.HandleFunc("GET /api/proxy/instances/{id}/acl/policy", acl.withAuth(acl.GetPolicy))
	mux.HandleFunc("PUT /api/proxy/instances/{id}/acl/policy", acl.withAuth(acl.PutPolicy))
	mux.HandleFunc("POST /api/proxy/instances/{id}/acl/policy/validate", acl.withAuth(acl.ValidatePolicy))
	mux.HandleFunc("POST /api/proxy/instances/{id}/acl/evaluate", acl.withAuth(acl.Evaluate))
	mux.HandleFunc("GET /api/proxy/acl/lists", acl.withAuth(acl.ListNamedLists))
	mux.HandleFunc("GET /api/proxy/acl/lists/{id}", acl.withAuth(acl.GetNamedList))
	mux.HandleFunc("PUT /api/proxy/acl/lists", acl.withAuth(acl.SyncNamedLists))
	mux.HandleFunc("POST /api/proxy/acl/lists/{id}/poll", acl.withAuth(acl.PollNamedList))
	if inspect != nil {
		mux.HandleFunc("GET /api/proxy/instances/{id}/inspect/status", inspect.withAuth(inspect.Status))
		mux.HandleFunc("GET /api/proxy/instances/{id}/inspect/rules", inspect.withAuth(inspect.ListRules))
		mux.HandleFunc("GET /api/proxy/instances/{id}/inspect/rules/{ruleId}", inspect.withAuth(inspect.GetRule))
		mux.HandleFunc("PUT /api/proxy/instances/{id}/inspect/rules", inspect.withAuth(inspect.SyncRules))
		mux.HandleFunc("POST /api/proxy/instances/{id}/inspect/validate", inspect.withAuth(inspect.ValidateScript))
	}
	if accessLog != nil {
		mux.HandleFunc("GET /api/proxy/access-log", accessLog.withAuth(accessLog.List))
		mux.HandleFunc("POST /api/proxy/access-log/report", accessLog.withAuth(accessLog.RunReport))
		mux.HandleFunc("POST /api/proxy/access-log/report/table", accessLog.withAuth(accessLog.RunReportTable))
		mux.HandleFunc("POST /api/proxy/access-log/delete", accessLog.withAuth(accessLog.DeleteByPeriod))
	}
}
