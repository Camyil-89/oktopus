package gateway

import (
	"context"
	stdhttp "net/http"

	"github.com/google/uuid"

	"oktopus/internal/id"
	"oktopus/internal/proxy/accesslog"
)

// Page502 — HTML-страница 502 и идентификатор инцидента (UUID v7).
type Page502 struct {
	ErrorID uuid.UUID
	Body    []byte
}

// Build502 рендерит страницу; ErrorID совпадает с id строки access log при RecordGatewayError.
func Build502(err error) Page502 {
	errorID := id.MustNew()
	body := render502Body(string(templateBytes()), errorID.String())
	return Page502{ErrorID: errorID, Body: body}
}

// RecordGatewayError пишет ошибку в access log и возвращает страницу с тем же ErrorID.
func RecordGatewayError(ctx context.Context, rec accesslog.Recorder, req *stdhttp.Request, err error) Page502 {
	page := Build502(err)
	if rec != nil {
		rec.Record(accesslog.GatewayErrorEntry(ctx, req, page.ErrorID, err, ClassifyError(err)))
	}
	return page
}
