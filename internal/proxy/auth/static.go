package auth

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"os"
	"strings"
)

// Static проверяет логин/пароль по таблице из файла (строки login:password).
type Static struct {
	users map[string]string
}

// LoadStaticFile читает учётки: одна строка login:password, # — комментарий.
func LoadStaticFile(path string) (*Static, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("auth static file: %w", err)
	}
	defer f.Close()
	users, err := parseStaticLines(bufio.NewScanner(f), path)
	if err != nil {
		return nil, err
	}
	return &Static{users: users}, nil
}

// ParseStaticUsers разбирает учётки из текста (строки login:password).
func ParseStaticUsers(content string) (*Static, error) {
	users, err := parseStaticLines(bufio.NewScanner(strings.NewReader(content)), "settings")
	if err != nil {
		return nil, err
	}
	return &Static{users: users}, nil
}

func parseStaticLines(sc *bufio.Scanner, label string) (map[string]string, error) {
	users := make(map[string]string)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		user, pass, ok := strings.Cut(line, ":")
		if !ok || user == "" {
			return nil, fmt.Errorf("auth static %s:%d: expected login:password", label, lineNo)
		}
		if _, dup := users[user]; dup {
			return nil, fmt.Errorf("auth static %s: duplicate user %q", label, user)
		}
		users[user] = pass
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("auth static: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("auth static %s: no users", label)
	}
	return users, nil
}

func (s *Static) Authenticate(_ context.Context, username, password string) (Identity, error) {
	stored, ok := s.users[username]
	if !ok {
		return Identity{}, ErrInvalidCredentials{}
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(password)) != 1 {
		return Identity{}, ErrInvalidCredentials{}
	}
	return Identity{Username: username}, nil
}
