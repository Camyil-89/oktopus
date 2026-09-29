package forbidden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReload_prefersActual(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, Dir)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	def := filepath.Join(cfgDir, DefaultFileName)
	act := filepath.Join(cfgDir, ActualFileName)
	if err := os.WriteFile(def, []byte("<!DOCTYPE html><html><body>default</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(act, []byte("<!DOCTYPE html><html><body>custom</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Reload(); err != nil {
		t.Fatal(err)
	}
	body := string(Snapshot())
	if !strings.Contains(body, "custom") {
		t.Fatalf("expected custom page, got %q", body)
	}
}
