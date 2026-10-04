package setup

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

// TestInstanceName — фиксированное имя PoC-инстанса в БД.
const TestInstanceName = "attack-and-tests"

// TestInstanceListen — listen при создании (только loopback).
const TestInstanceListen = "127.0.0.1:9000"

// TestInstanceDialAddr — адрес для CONNECT/HTTP-клиентов PoC (совпадает с listen).
const TestInstanceDialAddr = TestInstanceListen

// TestInstance описывает PoC-прокси после EnsureTestInstance.
type TestInstance struct {
	ID     string
	Listen string // для dial (127.0.0.1:9000)
}

// InstanceDTO — элемент GET /api/proxy/instances.
type InstanceDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

// EnsureTestInstance находит или создаёт прокси на порту 9000 для attack-and-tests.
func (c *Client) EnsureTestInstance() (TestInstance, error) {
	if id := strings.TrimSpace(c.testInstanceID); id != "" {
		return TestInstance{ID: id, Listen: TestInstanceDialAddr}, nil
	}
	list, err := c.ListInstances()
	if err != nil {
		return TestInstance{}, err
	}
	wantPort := listenPort(TestInstanceListen)
	for _, inst := range list {
		if inst.Name == TestInstanceName {
			c.testInstanceID = inst.ID
			log.Printf("test instance: already exists name=%q id=%s listen=%s", inst.Name, inst.ID, inst.Listen)
			return TestInstance{ID: inst.ID, Listen: dialAddrForListen(inst.Listen)}, nil
		}
		if wantPort > 0 && listenPort(inst.Listen) == wantPort {
			c.testInstanceID = inst.ID
			log.Printf("test instance: already exists on port %d id=%s name=%q listen=%s", wantPort, inst.ID, inst.Name, inst.Listen)
			return TestInstance{ID: inst.ID, Listen: dialAddrForListen(inst.Listen)}, nil
		}
	}
	created, err := c.CreateInstance(TestInstanceName, TestInstanceListen)
	if err != nil {
		if strings.Contains(err.Error(), "listen_address_in_use") {
			list, err2 := c.ListInstances()
			if err2 != nil {
				return TestInstance{}, err
			}
			for _, inst := range list {
				if listenPort(inst.Listen) == wantPort {
					c.testInstanceID = inst.ID
					log.Printf("test instance: reusing existing on port %d id=%s listen=%s", wantPort, inst.ID, inst.Listen)
					return TestInstance{ID: inst.ID, Listen: dialAddrForListen(inst.Listen)}, nil
				}
			}
		}
		return TestInstance{}, err
	}
	c.testInstanceID = created.ID
	log.Printf("test instance: created id=%s listen=%s", created.ID, created.Listen)
	return TestInstance{ID: created.ID, Listen: dialAddrForListen(created.Listen)}, nil
}

func (c *Client) ListInstances() ([]InstanceDTO, error) {
	var out []InstanceDTO
	if err := c.getJSON("/api/proxy/instances", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateInstance(name, listen string) (InstanceDTO, error) {
	var out InstanceDTO
	if err := c.postJSON("/api/proxy/instances", map[string]string{
		"name":   name,
		"listen": listen,
	}, &out); err != nil {
		return InstanceDTO{}, err
	}
	return out, nil
}

func (c *Client) requireTestInstanceID() (string, error) {
	if strings.TrimSpace(c.testInstanceID) != "" {
		return c.testInstanceID, nil
	}
	ti, err := c.EnsureTestInstance()
	if err != nil {
		return "", err
	}
	return ti.ID, nil
}

func (c *Client) instanceAPIPath(suffix string) (string, error) {
	id, err := c.requireTestInstanceID()
	if err != nil {
		return "", err
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return "/api/proxy/instances/" + id, nil
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return "/api/proxy/instances/" + id + suffix, nil
}

func listenPort(listen string) int {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return 0
	}
	if strings.HasPrefix(listen, ":") {
		p, err := strconv.Atoi(listen[1:])
		if err != nil {
			return 0
		}
		return p
	}
	_, portStr, err := net.SplitHostPort(listen)
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return p
}

func dialAddrForListen(listen string) string {
	if p := listenPort(listen); p == listenPort(TestInstanceListen) {
		return TestInstanceDialAddr
	}
	listen = strings.TrimSpace(listen)
	if strings.HasPrefix(listen, ":") {
		return "127.0.0.1" + listen
	}
	return listen
}

func (c *Client) instanceRuntime() (InstanceStatusDTO, error) {
	id, err := c.requireTestInstanceID()
	if err != nil {
		return InstanceStatusDTO{}, err
	}
	st, err := c.ProxyStatus()
	if err != nil {
		return InstanceStatusDTO{}, err
	}
	for _, inst := range st.Instances {
		if inst.ID == id {
			return inst, nil
		}
	}
	return InstanceStatusDTO{}, fmt.Errorf("instance %s not found in /api/proxy/status", id)
}
