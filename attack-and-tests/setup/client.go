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
	BaseURL        string
	HTTP           *http.Client
	testInstanceID string
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
	if _, err := c.EnsureTestInstance(); err != nil {
		return "", err
	}
	connectMode = strings.TrimSpace(strings.ToLower(connectMode))
	if connectMode != "mitm" && connectMode != "tunnel" {
		return "", fmt.Errorf("connect_mode must be mitm or tunnel, got %q", connectMode)
	}
	users := ProxyAuthUser + ":" + ProxyAuthPass
	listen, err = c.EnableProxyStaticAuth(connectMode, users)
	if err != nil {
		return "", err
	}
	if err := c.PublishPolicy(aclText); err != nil {
		return "", fmt.Errorf("acl policy: %w", err)
	}
	return listen, nil
}

// WaitACLReady ждёт успешной публикации ACL тестового инстанса.
func (c *Client) WaitACLReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		inst, err := c.instanceRuntime()
		if err != nil {
			return err
		}
		acl := inst.ACL
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

type CAStatusDTO struct {
	CertInstalled bool   `json:"cert_installed"`
	KeyInstalled  bool   `json:"key_installed"`
	ValidUntil    string `json:"valid_until,omitempty"`
}

// CAStatus — установлен ли CA на тестовом инстансе.
func (c *Client) CAStatus() (CAStatusDTO, error) {
	path, err := c.instanceAPIPath("/ca/status")
	if err != nil {
		return CAStatusDTO{}, err
	}
	var out CAStatusDTO
	if err := c.getJSON(path, &out); err != nil {
		return CAStatusDTO{}, err
	}
	return out, nil
}

// GenerateCA создаёт корневой CA через API (config/<instance-id>/ca.* на сервере).
func (c *Client) GenerateCA() error {
	path, err := c.instanceAPIPath("/ca/generate")
	if err != nil {
		return err
	}
	if err := c.postJSON(path, map[string]interface{}{}, nil); err != nil {
		return fmt.Errorf("POST ca/generate: %w", err)
	}
	return nil
}

// EnsureCA генерирует CA, если cert/key ещё не установлены.
func (c *Client) EnsureCA() error {
	if _, err := c.EnsureTestInstance(); err != nil {
		return err
	}
	st, err := c.CAStatus()
	if err != nil {
		return err
	}
	if st.CertInstalled && st.KeyInstalled {
		return nil
	}
	return c.GenerateCA()
}

// EnsureCAForConnectMode — MITM требует CA до apply настроек прокси.
func (c *Client) EnsureCAForConnectMode(connectMode string) error {
	connectMode = strings.TrimSpace(strings.ToLower(connectMode))
	if connectMode != "mitm" {
		return nil
	}
	return c.EnsureCA()
}

// DownloadCACert сохраняет CA MITM в path (создаёт каталоги при необходимости).
func (c *Client) DownloadCACert(path string) error {
	if err := c.EnsureCA(); err != nil {
		return err
	}
	certPath, err := c.instanceAPIPath("/ca/cert")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+certPath, nil)
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

type InstanceStatusDTO struct {
	ID              string `json:"id"`
	Listen          string `json:"listen"`
	Active          bool   `json:"active"`
	ProxyStartError string `json:"proxy_start_error"`
	ACL             struct {
		BuildStatus string `json:"build_status"`
		BuildError  string `json:"build_error"`
		RulesInSync bool   `json:"rules_in_sync"`
	} `json:"acl"`
}

type ProxyStatusDTO struct {
	ProxyActive bool                `json:"proxy_active"`
	Instances   []InstanceStatusDTO `json:"instances"`
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
