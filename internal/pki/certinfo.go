package pki

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// CAStatus состояние файлов CA на диске.
type CAStatus struct {
	CertExists bool
	KeyExists  bool
	NotAfter   time.Time
	HasCert    bool
}

// InspectCAFiles проверяет наличие cert/key и срок действия сертификата.
func InspectCAFiles(certPath, keyPath string) (CAStatus, error) {
	var st CAStatus
	if _, err := os.Stat(certPath); err == nil {
		st.CertExists = true
	}
	if _, err := os.Stat(keyPath); err == nil {
		st.KeyExists = true
	}
	if !st.CertExists {
		return st, nil
	}
	notAfter, err := readCertNotAfter(certPath)
	if err != nil {
		return st, err
	}
	st.HasCert = true
	st.NotAfter = notAfter
	return st, nil
}

func readCertNotAfter(path string) (time.Time, error) {
	der, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("read cert: %w", err)
	}
	block, _ := pem.Decode(der)
	if block == nil {
		return time.Time{}, fmt.Errorf("decode cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cert: %w", err)
	}
	return cert.NotAfter, nil
}
