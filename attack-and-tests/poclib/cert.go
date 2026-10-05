package poclib

import (
	"crypto/tls"
	"os"
	"path/filepath"
)

const labCertDir = "attack-and-tests/mitm-host-mismatch/certs"

// LoadDevKeyPair — dev TLS для lab origin (localhost SAN).
func LoadDevKeyPair() (tls.Certificate, error) {
	certPath := labCertPath()
	keyPath := labKeyPath()
	if err := ensureDevCertFiles(certPath, keyPath); err != nil {
		return tls.Certificate{}, err
	}
	return tls.LoadX509KeyPair(certPath, keyPath)
}

func labCertPath() string {
	if p := os.Getenv("ATTACK_TLS_CERT"); p != "" {
		return p
	}
	return filepath.Join(labCertDir, "dev.crt")
}

func labKeyPath() string {
	if p := os.Getenv("ATTACK_TLS_KEY"); p != "" {
		return p
	}
	return filepath.Join(labCertDir, "dev.key")
}

// ensureDevCertFiles duplicated minimally — call mitm-host-mismatch cert generation via files.
func ensureDevCertFiles(certPath, keyPath string) error {
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}
	// Delegate to existing certs: run from repo root after mitm-host-mismatch lab once,
	// or copy generation — import from main package not possible; use same paths.
	return generateDevCertsIfMissing(certPath, keyPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
