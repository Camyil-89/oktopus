package setup

import (
	"fmt"
	"time"
)

// InspectRuleInput — элемент PUT /api/proxy/instances/{id}/inspect/rules.
type InspectRuleInput struct {
	ID        *string
	Name      string
	Script    string
	Action    int16
	Enabled   bool
	SortOrder int
}

type inspectCompileStatusDTO struct {
	BuildStatus string `json:"build_status"`
	BuildError  string `json:"build_error"`
	RulesInSync bool   `json:"rules_in_sync"`
}

// SyncInspectRules заменяет все правила инспекции на инстансе и ждёт публикации.
func (c *Client) SyncInspectRules(rules []InspectRuleInput) error {
	if _, err := c.EnsureTestInstance(); err != nil {
		return err
	}
	path, err := c.instanceAPIPath("/inspect/rules")
	if err != nil {
		return err
	}
	payload := make([]map[string]interface{}, len(rules))
	for i, r := range rules {
		script := r.Script
		item := map[string]interface{}{
			"name":       r.Name,
			"script":     &script,
			"action":     r.Action,
			"enabled":    r.Enabled,
			"sort_order": r.SortOrder,
		}
		if r.ID != nil && *r.ID != "" {
			item["id"] = *r.ID
		}
		payload[i] = item
	}
	if err := c.putJSON(path, map[string]interface{}{"rules": payload}, nil); err != nil {
		return fmt.Errorf("sync inspect rules: %w", err)
	}
	return c.WaitInspectReady(30 * time.Second)
}

// WaitInspectReady ждёт compile ready и rules_in_sync.
func (c *Client) WaitInspectReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := c.inspectCompileStatus()
		if err != nil {
			return err
		}
		if st.BuildStatus == "error" && st.BuildError != "" {
			return fmt.Errorf("inspect build error: %s", st.BuildError)
		}
		if st.RulesInSync && st.BuildStatus != "building" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for inspect publish (build=%s in_sync=%v)", st.BuildStatus, st.RulesInSync)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (c *Client) inspectCompileStatus() (inspectCompileStatusDTO, error) {
	path, err := c.instanceAPIPath("/inspect/status")
	if err != nil {
		return inspectCompileStatusDTO{}, err
	}
	var out inspectCompileStatusDTO
	if err := c.getJSON(path, &out); err != nil {
		return inspectCompileStatusDTO{}, err
	}
	return out, nil
}
