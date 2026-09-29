package inspect

import (
	"bytes"
	"io"
	stdhttp "net/http"
	"regexp"
	"strings"
)

const maxInspectBodyBytes = 4 << 20

var (
	filenameParamRe = regexp.MustCompile(`(?i)filename\*=UTF-8''([^;\r\n]+)|(?i)filename="([^"]+)"|(?i)filename=([^;\r\n\s]+)`)
)

// AttachBodyMeta читает тело (с лимитом), восстанавливает req.Body, заполняет имена файлов.
func AttachBodyMeta(c *RequestContext, req *stdhttp.Request) {
	if c == nil || req == nil || req.Body == nil {
		return
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxInspectBodyBytes))
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 {
		return
	}
	c.UploadFilenames = extractUploadFilenames(body, c.ContentType, req.Header)
}

func extractUploadFilenames(body []byte, contentType string, hdr stdhttp.Header) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	cd := strings.TrimSpace(hdr.Get("Content-Disposition"))
	if cd != "" {
		for _, m := range filenameParamRe.FindAllStringSubmatch(cd, -1) {
			for i := 1; i < len(m); i++ {
				if m[i] != "" {
					add(decodeFilenameToken(m[i]))
				}
			}
		}
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(ct, "multipart/") {
		for _, m := range filenameParamRe.FindAllSubmatch(body, -1) {
			for i := 1; i < len(m); i++ {
				if len(m[i]) > 0 {
					add(decodeFilenameToken(string(m[i])))
				}
			}
		}
	}
	return out
}

func decodeFilenameToken(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"`)
	return raw
}
