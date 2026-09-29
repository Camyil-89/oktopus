package auth_test

import (
	"testing"
	"time"

	"oktopus/internal/proxy/auth"
)

func TestAuthCacheClearAndReconfigure(t *testing.T) {
	c := auth.NewAuthCache(time.Minute)
	c.Set("u", "p", auth.Identity{Username: "u"})
	if _, ok := c.Get("u", "p"); !ok {
		t.Fatal("expected cache hit before clear")
	}
	c.Clear()
	if _, ok := c.Get("u", "p"); ok {
		t.Fatal("expected miss after clear")
	}

	c.Set("u", "p", auth.Identity{Username: "u"})
	c.Reconfigure(2 * time.Minute)
	if _, ok := c.Get("u", "p"); ok {
		t.Fatal("expected miss after reconfigure")
	}
	if c.TTL() != 2*time.Minute {
		t.Fatalf("ttl: %s", c.TTL())
	}
}
