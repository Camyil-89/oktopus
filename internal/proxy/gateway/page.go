package gateway

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

const (
	Dir              = "config"
	DefaultFileName  = "gateway.default.html"
	ActualFileName   = "gateway.actual.html"
	ErrorIDPlaceholder = "{{ERROR_ID}}"
	// ErrorPlaceholder — устаревший alias; подставляется тот же UUID, что и в ERROR_ID.
	ErrorPlaceholder   = "{{ERROR}}"
	PreviewErrorID     = "018f0000-0000-7000-8000-000000000099"
	maxCustomBytes   = 512 << 10
	contentTypeHTML  = "text/html; charset=utf-8"
)

// ContentTypeHTML — Content-Type страницы 502.
func ContentTypeHTML() string { return contentTypeHTML }

var template atomic.Pointer[[]byte]

func DefaultPath() string { return filepath.Join(Dir, DefaultFileName) }
func ActualPath() string  { return filepath.Join(Dir, ActualFileName) }

type Status struct {
	UsingCustom bool
	CustomFile  bool
}

func Reload() error {
	data, err := loadTemplateFromDisk()
	if err != nil {
		return err
	}
	template.Store(&data)
	return nil
}

func loadTemplateFromDisk() ([]byte, error) {
	if data, ok, err := readActualTemplate(); err != nil {
		return nil, err
	} else if ok {
		return data, nil
	}
	return readDefaultTemplate()
}

func readDefaultTemplate() ([]byte, error) {
	def := DefaultPath()
	data, err := os.ReadFile(def)
	if err != nil {
		return []byte(fallbackHTML), nil
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []byte(fallbackHTML), nil
	}
	return data, nil
}

func readActualTemplate() ([]byte, bool, error) {
	actual := ActualPath()
	data, err := os.ReadFile(actual)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, false, nil
	}
	return data, true, nil
}

func templateForPreview(variant string) ([]byte, error) {
	switch variant {
	case "default":
		return readDefaultTemplate()
	case "custom":
		data, ok, err := readActualTemplate()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("custom page not installed")
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unknown variant %q", variant)
	}
}

// HTMLForPreview рендерит страницу 502 с примером идентификатора инцидента.
func HTMLForPreview(variant string) ([]byte, error) {
	tpl, err := templateForPreview(variant)
	if err != nil {
		return nil, err
	}
	return render502Body(string(tpl), PreviewErrorID), nil
}

func templateBytes() []byte {
	if p := template.Load(); p != nil {
		return *p
	}
	data, _ := loadTemplateFromDisk()
	return data
}

func render502Body(tpl string, errorID string) []byte {
	errorID = strings.TrimSpace(errorID)
	escaped := html.EscapeString(errorID)
	if !strings.Contains(tpl, ErrorIDPlaceholder) && !strings.Contains(tpl, ErrorPlaceholder) {
		tpl += "<pre>" + ErrorIDPlaceholder + "</pre>"
	}
	body := strings.ReplaceAll(tpl, ErrorIDPlaceholder, escaped)
	body = strings.ReplaceAll(body, ErrorPlaceholder, escaped)
	return []byte(body)
}

// Render502 подставляет UUID инцидента в шаблон (без текста upstream-ошибки).
func Render502(err error) []byte {
	return Build502(err).Body
}

func Write502(w io.Writer, err error) error {
	body := Render502(err)
	res := &http.Response{
		StatusCode:    http.StatusBadGateway,
		Status:        "502 Bad Gateway",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{contentTypeHTML}},
		Body:          io.NopCloser(strings.NewReader(string(body))),
		ContentLength: int64(len(body)),
	}
	return res.Write(w)
}

func PageStatus() Status {
	customFile := false
	if st, err := os.Stat(ActualPath()); err == nil && !st.IsDir() {
		customFile = true
	}
	usingCustom := false
	if customFile {
		if data, err := os.ReadFile(ActualPath()); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			usingCustom = true
		}
	}
	return Status{UsingCustom: usingCustom, CustomFile: customFile}
}

func SaveCustom(r io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(r, maxCustomBytes+1))
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}
	if len(data) > maxCustomBytes {
		return fmt.Errorf("html file too large (max %d bytes)", maxCustomBytes)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return fmt.Errorf("html file is empty")
	}
	if !looksLikeHTML(data) {
		return fmt.Errorf("file does not look like HTML")
	}
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	target := ActualPath()
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write html: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install html: %w", err)
	}
	return Reload()
}

func ClearCustom() error {
	_ = os.Remove(ActualPath())
	return Reload()
}

func looksLikeHTML(b []byte) bool {
	s := strings.ToLower(string(b))
	return strings.Contains(s, "<html") || strings.Contains(s, "<!doctype")
}

const fallbackHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>502 Bad Gateway</title>
</head>
<body>
<h1>502 Bad Gateway</h1>
<pre>` + ErrorIDPlaceholder + `</pre>
</body>
</html>
`
