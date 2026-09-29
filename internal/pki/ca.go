package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CAConfig задаёт параметры корневого CA.
type CAConfig struct {
	CommonName   string
	Organization string
	Country      string
	ValidDays    int
	KeyBits      int
	OutDir       string
	Force        bool
}

// GenerateCA создаёт ca.key и ca.crt в OutDir, если файлов ещё нет (или Force).
func GenerateCA(cfg CAConfig) error {
	if cfg.CommonName == "" {
		return errors.New("common name is required")
	}
	if cfg.ValidDays <= 0 {
		return errors.New("valid days must be positive")
	}
	if cfg.OutDir == "" {
		return errors.New("output directory is required")
	}
	keyBits := cfg.KeyBits
	if keyBits == 0 {
		keyBits = 4096
	}
	if keyBits < 2048 {
		return fmt.Errorf("key bits must be >= 2048, got %d", keyBits)
	}

	keyPath := filepath.Join(cfg.OutDir, "ca.key")
	certPath := filepath.Join(cfg.OutDir, "ca.crt")

	if !cfg.Force {
		if _, err := os.Stat(keyPath); err == nil {
			return fmt.Errorf("%s already exists (use -force to overwrite)", keyPath)
		}
		if _, err := os.Stat(certPath); err == nil {
			return fmt.Errorf("%s already exists (use -force to overwrite)", certPath)
		}
	}

	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return fmt.Errorf("generate rsa key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	subject := pkix.Name{
		CommonName: cfg.CommonName,
	}
	if cfg.Organization != "" {
		subject.Organization = []string{cfg.Organization}
	}
	if cfg.Country != "" {
		subject.Country = []string{cfg.Country}
	}

	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               subject,
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(0, 0, cfg.ValidDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	if err := writeKeyPEM(keyPath, priv); err != nil {
		return err
	}
	if err := writeCertPEM(certPath, certDER); err != nil {
		return err
	}

	return nil
}

// SaveCAFiles записывает PEM сертификата и ключа в OutDir (ca.crt, ca.key).
func SaveCAFiles(outDir string, certPEM, keyPEM []byte, force bool) (certPath, keyPath string, err error) {
	if outDir == "" {
		return "", "", errors.New("output directory is required")
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return "", "", errors.New("cert and key PEM are required")
	}
	keyPath = filepath.Join(outDir, "ca.key")
	certPath = filepath.Join(outDir, "ca.crt")
	if !force {
		if _, statErr := os.Stat(keyPath); statErr == nil {
			return "", "", fmt.Errorf("%s already exists (use force to overwrite)", keyPath)
		}
		if _, statErr := os.Stat(certPath); statErr == nil {
			return "", "", fmt.Errorf("%s already exists (use force to overwrite)", certPath)
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", fmt.Errorf("write %s: %w", keyPath, err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", certPath, err)
	}
	return certPath, keyPath, nil
}

func writeKeyPEM(path string, key *rsa.PrivateKey) error {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, block); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func writeCertPEM(path string, certDER []byte) error {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, block); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}
