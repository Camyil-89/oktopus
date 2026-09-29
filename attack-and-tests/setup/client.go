package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

// Учётка прокси, которую PoC выставляет через API перед прогоном.
const (
	ProxyAuthUser = "attack-poc"
	ProxyAuthPass = "attack-poc-secret"
)

// Client — сессия UI/API (cookie после login).
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient создаёт клиент с cookie-jar.
func NewClient(apiBase string) (*Client, error) {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		return nil, fmt.Errorf("empty api base")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseURL: apiBase,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
			Jar:     jar,
		},
	}, nil
}

func (c *Client) Login(username, password string) error {
	var out struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := c.postJSON("/api/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, &out); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	return nil
}

// ApplyProxyTestProfile включает прокси, auth и режим CONNECT, публикует ACL.
func (c *Client) ApplyProxyTestProfile(connectMode, aclText string) (listen string, err error) {
	connectMode = strings.TrimSpace(strings.ToLower(connectMode))
	if connectMode != "mitm" && connectMode != "tunnel" {
		return "", fmt.Errorf("connect_mode must be mitm or tunnel, got %q", connectMode)
	}
	users := ProxyAuthUser + ":" + ProxyAuthPass
	if err := c.patchJSON("/api/proxy/settings", map[string]interface{}{
		"proxy_enabled":     true,
		"connect_mode":      connectMode,
		"auth_enabled":      true,
		"auth_static_users": users,
		"auth_backend":      "static",
	}); err != nil {
		return "", fmt.Errorf("proxy settings: %w", err)
	}
	var pol struct {
		ConfigText string `json:"config_text"`
		UpdatedAt  string `json:"updated_at"`
	}
	if err := c.putJSON("/api/proxy/acl/policy", map[string]string{
		"config_text": aclText,
	}, &pol); err != nil {
		return "", fmt.Errorf("acl policy: %w", err)
	}
	if err := c.WaitACLReady(20 * time.Second); err != nil {
		return "", err
	}
	st, err := c.ProxyStatus()
	if err != nil {
		return "", err
	}
	if !st.ProxyActive {
		if st.ProxyStartError != "" {
			return "", fmt.Errorf("proxy not active: %s", st.ProxyStartError)
		}
		return "", fmt.Errorf("proxy not active (check listen / serve)")
	}
	listen = strings.TrimSpace(st.Listen)
	if listen == "" {
		return "", fmt.Errorf("proxy listen empty in status")
	}
	return listen, nil
}

// WaitACLReady ждёт успешной публикации ACL.
func (c *Client) WaitACLReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := c.ProxyStatus()
		if err != nil {
			return err
		}
		acl := st.ACL
		if acl.BuildStatus == "error" && acl.BuildError != "" {
			return fmt.Errorf("acl build error: %s", acl.BuildError)
		}
		if acl.RulesInSync && acl.BuildStatus != "building" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for acl publish (build=%s in_sync=%v)", acl.BuildStatus, acl.RulesInSync)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// DownloadCACert сохраняет CA MITM в path (создаёт каталоги при необходимости).
func (c *Client) DownloadCACert(path string) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/proxy/ca/cert", nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("GET ca/cert: HTTP %d %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

type ProxyStatusDTO struct {
	ProxyActive     bool   `json:"proxy_active"`
	Listen          string `json:"listen"`
	ProxyStartError string `json:"proxy_start_error"`
	ACL             struct {
		BuildStatus string `json:"build_status"`
		BuildError  string `json:"build_error"`
		RulesInSync bool   `json:"rules_in_sync"`
	} `json:"acl"`
}

func (c *Client) ProxyStatus() (ProxyStatusDTO, error) {
	var out ProxyStatusDTO
	if err := c.getJSON("/api/proxy/status", &out); err != nil {
		return ProxyStatusDTO{}, err
	}
	return out, nil
}

func (c *Client) getJSON(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, out)
}

func (c *Client) postJSON(path string, body interface{}, out interface{}) error {
	return c.jsonRequest(http.MethodPost, path, body, out)
}

func (c *Client) putJSON(path string, body interface{}, out interface{}) error {
	return c.jsonRequest(http.MethodPut, path, body, out)
}

func (c *Client) patchJSON(path string, body interface{}) error {
	return c.jsonRequest(http.MethodPatch, path, body, nil)
}

func (c *Client) jsonRequest(method, path string, body interface{}, out interface{}) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.doJSON(req, out)
}

func (c *Client) doJSON(req *http.Request, out interface{}) error {
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return fmt.Errorf("%s %s: HTTP %d %s", req.Method, req.URL.Path, res.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s: %w", req.URL.Path, err)
	}
	return nil
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	if i := strings.LastIndex(path, "\\"); i >= 0 {
		return path[:i]
	}
	return "."
}

// LoadACLExample читает acl.example.squid из каталога PoC.
func LoadACLExample(pocDir string) (string, error) {
	path := strings.TrimRight(pocDir, "/\\") + "/acl.example.squid"
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return stripACLComments(string(data)), nil
}

func stripACLComments(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
