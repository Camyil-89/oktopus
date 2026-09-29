package auth_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"oktopus/internal/proxy/auth"
)

func TestStaticAuthenticate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users")
	if err := os.WriteFile(path, []byte("alice:secret\n# bob\nbob:pass2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := auth.LoadStaticFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	id, err := st.Authenticate(ctx, "alice", "secret")
	if err != nil || id.Username != "alice" {
		t.Fatalf("alice: id=%+v err=%v", id, err)
	}
	if _, err := st.Authenticate(ctx, "alice", "wrong"); !auth.IsInvalidCredentials(err) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if _, err := st.Authenticate(ctx, "nobody", "secret"); !auth.IsInvalidCredentials(err) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}
