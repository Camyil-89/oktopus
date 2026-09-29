package setup

import (
	"os"
	"path/filepath"
	"time"
)

// EvaluateInput — тело POST /api/proxy/acl/evaluate.
type EvaluateInput struct {
	SNI      string   `json:"sni"`
	Path     string   `json:"path"`
	SrcIP    string   `json:"src_ip"`
	DstIP    string   `json:"dst_ip"`
	DstPort  int      `json:"dst_port"`
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

// EvaluateResult — ответ evaluate.
type EvaluateResult struct {
	Allowed bool `json:"allowed"`
	Steps   []struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	} `json:"steps"`
}

// AnalyzeResult — ответ POST /api/proxy/acl/policy/validate.
type AnalyzeResult struct {
	OK          bool `json:"ok"`
	Diagnostics []struct {
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Code     string `json:"code"`
		Line     int    `json:"line"`
	} `json:"diagnostics"`
}

// PublishPolicy публикует текст политики (нужен предварительный Login).
func (c *Client) PublishPolicy(aclText string) error {
	var pol struct {
		ConfigText string `json:"config_text"`
	}
	if err := c.putJSON("/api/proxy/acl/policy", map[string]string{
		"config_text": aclText,
	}, &pol); err != nil {
		return err
	}
	return c.WaitACLReady(20 * time.Second)
}

func (c *Client) Evaluate(in EvaluateInput) (EvaluateResult, error) {
	var out EvaluateResult
	if err := c.postJSON("/api/proxy/acl/evaluate", in, &out); err != nil {
		return EvaluateResult{}, err
	}
	return out, nil
}

func (c *Client) ValidatePolicy(configText string) (AnalyzeResult, error) {
	var out AnalyzeResult
	if err := c.postJSON("/api/proxy/acl/policy/validate", map[string]string{
		"config_text": configText,
	}, &out); err != nil {
		return AnalyzeResult{}, err
	}
	return out, nil
}

// LoadPolicyFile читает policy.squid из каталога PoC.
func LoadPolicyFile(pocDir string) (string, error) {
	path := filepath.Join(pocDir, "policy.squid")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return stripACLComments(string(data)), nil
}
