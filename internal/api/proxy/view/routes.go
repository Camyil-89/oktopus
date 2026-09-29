package view

import "net/http"

func Register(mux *http.ServeMux, h *Handler, acl *ACLHandler, inspect *InspectHandler, status *StatusHandler, accessLog *AccessLogHandler) {
	mux.HandleFunc("GET /api/proxy/settings", h.withAuth(h.Get))
	mux.HandleFunc("PATCH /api/proxy/settings", h.withAuth(h.Patch))
	mux.HandleFunc("GET /api/proxy/ca/status", h.withAuth(h.CAStatus))
	mux.HandleFunc("GET /api/proxy/ca/cert", h.withAuth(h.DownloadCert))
	mux.HandleFunc("GET /api/proxy/ca/key", h.withAuth(h.DownloadKey))
	mux.HandleFunc("POST /api/proxy/ca/generate", h.withAuth(h.GenerateCA))
	mux.HandleFunc("POST /api/proxy/ca/upload", h.withAuth(h.UploadCA))
	mux.HandleFunc("GET /api/proxy/forbidden/status", h.withAuth(h.ForbiddenStatus))
	mux.HandleFunc("GET /api/proxy/forbidden/preview", h.withAuth(h.ForbiddenPreview))
	mux.HandleFunc("POST /api/proxy/forbidden/upload", h.withAuth(h.UploadForbiddenPage))
	mux.HandleFunc("DELETE /api/proxy/forbidden/custom", h.withAuth(h.ClearForbiddenPage))
	mux.HandleFunc("GET /api/proxy/gateway/status", h.withAuth(h.GatewayStatus))
	mux.HandleFunc("GET /api/proxy/gateway/preview", h.withAuth(h.GatewayPreview))
	mux.HandleFunc("POST /api/proxy/gateway/upload", h.withAuth(h.UploadGatewayPage))
	mux.HandleFunc("DELETE /api/proxy/gateway/custom", h.withAuth(h.ClearGatewayPage))
	mux.HandleFunc("GET /api/proxy/status", status.withAuth(status.Get))
	mux.HandleFunc("GET /api/proxy/acl/status", acl.withAuth(acl.Status))
	mux.HandleFunc("GET /api/proxy/acl/policy", acl.withAuth(acl.GetPolicy))
	mux.HandleFunc("PUT /api/proxy/acl/policy", acl.withAuth(acl.PutPolicy))
	mux.HandleFunc("POST /api/proxy/acl/policy/validate", acl.withAuth(acl.ValidatePolicy))
	mux.HandleFunc("GET /api/proxy/acl/lists", acl.withAuth(acl.ListNamedLists))
	mux.HandleFunc("GET /api/proxy/acl/lists/{id}", acl.withAuth(acl.GetNamedList))
	mux.HandleFunc("PUT /api/proxy/acl/lists", acl.withAuth(acl.SyncNamedLists))
	mux.HandleFunc("POST /api/proxy/acl/lists/{id}/poll", acl.withAuth(acl.PollNamedList))
	mux.HandleFunc("POST /api/proxy/acl/evaluate", acl.withAuth(acl.Evaluate))
	if inspect != nil {
		mux.HandleFunc("GET /api/proxy/inspect/status", inspect.withAuth(inspect.Status))
		mux.HandleFunc("GET /api/proxy/inspect/rules", inspect.withAuth(inspect.ListRules))
		mux.HandleFunc("GET /api/proxy/inspect/rules/{id}", inspect.withAuth(inspect.GetRule))
		mux.HandleFunc("PUT /api/proxy/inspect/rules", inspect.withAuth(inspect.SyncRules))
		mux.HandleFunc("POST /api/proxy/inspect/validate", inspect.withAuth(inspect.ValidateScript))
	}
	if accessLog != nil {
		mux.HandleFunc("GET /api/proxy/access-log", accessLog.withAuth(accessLog.List))
		mux.HandleFunc("POST /api/proxy/access-log/report", accessLog.withAuth(accessLog.RunReport))
		mux.HandleFunc("POST /api/proxy/access-log/report/table", accessLog.withAuth(accessLog.RunReportTable))
		mux.HandleFunc("POST /api/proxy/access-log/delete", accessLog.withAuth(accessLog.DeleteByPeriod))
	}
}
