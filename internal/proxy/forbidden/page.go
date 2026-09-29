package forbidden

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"oktopus/internal/proxy/bytecount"
)

const (
	Dir              = "config"
	DefaultFileName  = "forbidden.default.html"
	ActualFileName   = "forbidden.actual.html"
	maxCustomBytes   = 512 << 10
	contentTypeHTML  = "text/html; charset=utf-8"
)

var body atomic.Pointer[[]byte]

// DefaultPath и ActualPath — пути к HTML в каталоге config.
func DefaultPath() string { return filepath.Join(Dir, DefaultFileName) }
func ActualPath() string  { return filepath.Join(Dir, ActualFileName) }

// Status — какая страница сейчас отдаётся клиенту.
type Status struct {
	UsingCustom bool
	CustomFile  bool
}

// Reload читает actual.html при наличии, иначе default.html.
func Reload() error {
	data, err := loadFromDisk()
	if err != nil {
		return err
	}
	body.Store(&data)
	return nil
}

func loadFromDisk() ([]byte, error) {
	if data, ok, err := readActualFile(); err != nil {
		return nil, err
	} else if ok {
		return data, nil
	}
	return readDefaultFile()
}

func readDefaultFile() ([]byte, error) {
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

func readActualFile() ([]byte, bool, error) {
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

// HTMLForPreview возвращает default или custom HTML для предпросмотра в UI.
func HTMLForPreview(variant string) ([]byte, error) {
	switch variant {
	case "default":
		return readDefaultFile()
	case "custom":
		data, ok, err := readActualFile()
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

// Snapshot возвращает тело ответа 403 (после Reload).
func Snapshot() []byte {
	if p := body.Load(); p != nil {
		return *p
	}
	data, _ := loadFromDisk()
	return data
}

// Write403 отправляет HTML-страницу с кодом 403.
func Write403(w http.ResponseWriter) {
	b := Snapshot()
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
	w.WriteHeader(http.StatusForbidden)
	n, _ := w.Write(b)
	bytecount.SupplementDeniedDown(w, n)
}

// PageStatus описывает, используется ли загруженная страница.
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

// SaveCustom записывает HTML в forbidden.actual.html и обновляет кеш.
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

// ClearCustom удаляет actual.html и возвращает default.
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
<title>403 Forbidden</title>
</head>
<body>
<h1>403 Forbidden</h1>
<p>Доступ запрещён политикой Oktopus.</p>
</body>
</html>
`
