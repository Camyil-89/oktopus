package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Authority — корневой CA и выпуск leaf-сертификатов для MITM.
type Authority struct {
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey

	mu    sync.Mutex
	cache map[string]*tls.Certificate

	leafKeyOnce sync.Once
	leafKey     *ecdsa.PrivateKey
	leafKeyErr  error
}

// LoadAuthority читает ca.crt и ca.key с диска.
func LoadAuthority(certPath, keyPath string) (*Authority, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read ca cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read ca key: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode ca cert pem")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}

	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("decode ca key pem")
	}
	caKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca key: %w", err)
	}

	if !caCert.IsCA {
		return nil, fmt.Errorf("ca cert is not a CA (IsCA=false)")
	}

	return &Authority{
		caCert: caCert,
		caKey:  caKey,
		cache:  make(map[string]*tls.Certificate),
	}, nil
}

// LogSummary печатает subject и отпечаток CA — сверь с установленным в системе сертификатом.
func (a *Authority) LogSummary(logger *log.Logger) {
	if logger == nil {
		logger = log.Default()
	}
	sum := sha256.Sum256(a.caCert.Raw)
	logger.Printf("CA subject: %s", a.caCert.Subject.String())
	logger.Printf("CA SHA-256: %s", strings.ToUpper(hex.EncodeToString(sum[:])))
}

// LeafCertificate возвращает TLS-сертификат для host (без порта).
func (a *Authority) LeafCertificate(host string) (*tls.Certificate, error) {
	host = stripPort(host)
	if host == "" {
		return nil, fmt.Errorf("empty host")
	}

	a.mu.Lock()
	if c, ok := a.cache[host]; ok {
		a.mu.Unlock()
		return c, nil
	}
	a.mu.Unlock()

	leaf, err := a.issueLeaf(host)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.cache[host] = leaf
	a.mu.Unlock()
	return leaf, nil
}

func (a *Authority) sharedLeafKey() (*ecdsa.PrivateKey, error) {
	a.leafKeyOnce.Do(func() {
		a.leafKey, a.leafKeyErr = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	})
	return a.leafKey, a.leafKeyErr
}

func (a *Authority) issueLeaf(host string) (*tls.Certificate, error) {
	leafKey, err := a.sharedLeafKey()
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: host},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, a.caCert, &leafKey.PublicKey, a.caKey)
	if err != nil {
		return nil, err
	}

	// Клиенту отдаём только leaf; корень должен быть в доверенных хранилищах ОС.
	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  leafKey,
	}, nil
}

func stripPort(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}
