package setup

import (
	"path/filepath"
)

type APIFlags struct {
	APIBase   string
	APIUser   string
	APIPass   string
	SkipSetup bool
}

func DefaultAPIFlags() APIFlags {
	return APIFlags{
		APIBase: "http://127.0.0.1:8000",
		APIUser: "admin",
		APIPass: "admin",
	}
}

// PoC используют отдельный инстанс 127.0.0.1:9000 — см. EnsureTestInstance.

// POCDirFromCaller возвращает каталог PoC относительно cwd (attack-and-tests/<name>).
func POCDirFromCaller(subdir string) string {
	return filepath.Join("attack-and-tests", subdir)
}

