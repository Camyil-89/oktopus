package view

import (
	"fmt"
	"net/http"

	"oktopus/internal/api/platform/response"
	"oktopus/internal/proxy/gateway"
)

func (h *Handler) GatewayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st := gateway.PageStatus()
	response.JSON(w, http.StatusOK, gatewayStatusResponse{
		UsingCustom: st.UsingCustom,
		CustomFile:  st.CustomFile,
		DefaultPath: gateway.DefaultPath(),
		ActualPath:  gateway.ActualPath(),
	})
}

func (h *Handler) UploadGatewayPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	const maxBody = 512 << 10
	if err := r.ParseMultipartForm(maxBody); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	htmlFile, _, err := r.FormFile("html")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "html file required")
		return
	}
	defer htmlFile.Close()

	if err := gateway.SaveCustom(htmlFile); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	st := gateway.PageStatus()
	response.JSON(w, http.StatusOK, gatewayStatusResponse{
		UsingCustom: st.UsingCustom,
		CustomFile:  st.CustomFile,
		DefaultPath: gateway.DefaultPath(),
		ActualPath:  gateway.ActualPath(),
	})
}

func (h *Handler) ClearGatewayPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := gateway.ClearCustom(); err != nil {
		response.Error(w, http.StatusInternalServerError, "clear gateway page failed")
		return
	}
	st := gateway.PageStatus()
	response.JSON(w, http.StatusOK, gatewayStatusResponse{
		UsingCustom: st.UsingCustom,
		CustomFile:  st.CustomFile,
		DefaultPath: gateway.DefaultPath(),
		ActualPath:  gateway.ActualPath(),
	})
}

func (h *Handler) GatewayPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	variant := r.URL.Query().Get("variant")
	if variant == "" {
		variant = "default"
	}
	body, err := gateway.HTMLForPreview(variant)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

type gatewayStatusResponse struct {
	UsingCustom bool   `json:"using_custom"`
	CustomFile  bool   `json:"custom_file"`
	DefaultPath string `json:"default_path"`
	ActualPath  string `json:"actual_path"`
}
