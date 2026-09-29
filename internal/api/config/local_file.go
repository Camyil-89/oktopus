package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	envConfigFile     = "OKTOPUS_CONFIG_FILE"
	defaultConfigFile = "config/oktopus.local.json"
	jwtSecretBytes    = 32
)

// LocalFile локальные настройки, сохраняемые на диск (рядом с проектом).
type LocalFile struct {
	JWTSecret string `json:"jwt_secret"`
}

func resolveJWTSecret() (string, error) {
	if v := strings.TrimSpace(os.Getenv(envJWTSecret)); v != "" {
		return v, nil
	}
	path := envOr(envConfigFile, defaultConfigFile)
	return ensureJWTSecretFile(path)
}

func ensureJWTSecretFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("config file path is empty")
	}

	existing, err := readLocalFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err == nil && strings.TrimSpace(existing.JWTSecret) != "" {
		return strings.TrimSpace(existing.JWTSecret), nil
	}

	secret, err := generateJWTSecret()
	if err != nil {
		return "", err
	}

	toSave := existing
	if toSave == nil {
		toSave = &LocalFile{}
	}
	toSave.JWTSecret = secret

	if err := writeLocalFile(path, toSave); err != nil {
		return "", err
	}

	log.Printf("api config: создан %s с новым jwt_secret", path)
	return secret, nil
}

func readLocalFile(path string) (*LocalFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &LocalFile{}, nil
	}
	var f LocalFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &f, nil
}

func writeLocalFile(path string, f *LocalFile) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir config dir: %w", err)
		}
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, jwtSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate jwt secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
