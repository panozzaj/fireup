// Package certs provides dynamic TLS certificate generation for local development.
// It generates a CA certificate once, then creates per-domain certificates on-the-fly.
package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles CA and dynamic certificate generation
type Manager struct {
	certsDir string
	tld      string

	caCert    *x509.Certificate
	caKey     *ecdsa.PrivateKey
	caTLSCert tls.Certificate

	// Cache of generated certificates per domain
	cache   map[string]*tls.Certificate
	cacheMu sync.RWMutex
}

// NewManager creates a new certificate manager
func NewManager(certsDir, tld string) (*Manager, error) {
	m := &Manager{
		certsDir: certsDir,
		tld:      tld,
		cache:    make(map[string]*tls.Certificate),
	}

	if err := m.loadCA(); err != nil {
		return nil, err
	}

	return m, nil
}

// CAExists checks if the CA certificate exists
func CAExists(certsDir string) bool {
	certPath := filepath.Join(certsDir, "ca.pem")
	keyPath := filepath.Join(certsDir, "ca-key.pem")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

// GenerateCA creates a new CA certificate and key
func GenerateCA(certsDir string) error {
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return fmt.Errorf("creating certs directory: %w", err)
	}

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"fireup"},
			CommonName:   "fireup Local CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Valid for 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	// Write certificate
	certPath := filepath.Join(certsDir, "ca.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("creating cert file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("encoding certificate: %w", err)
	}

	// Write private key
	keyPath := filepath.Join(certsDir, "ca-key.pem")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating key file: %w", err)
	}
	defer keyFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}

	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("encoding private key: %w", err)
	}

	return nil
}

// GetCACertPath returns the path to the CA certificate
func GetCACertPath(certsDir string) string {
	return filepath.Join(certsDir, "ca.pem")
}

// loadCA loads the CA certificate and key from disk
func (m *Manager) loadCA() error {
	certPath := filepath.Join(m.certsDir, "ca.pem")
	keyPath := filepath.Join(m.certsDir, "ca-key.pem")

	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("reading CA certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	m.caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing CA certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("reading CA key: %w", err)
	}

	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	m.caKey, err = x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parsing CA key: %w", err)
	}

	// Create TLS certificate for the CA itself
	m.caTLSCert = tls.Certificate{
		Certificate: [][]byte{m.caCert.Raw},
		PrivateKey:  m.caKey,
		Leaf:        m.caCert,
	}

	return nil
}

// GetCertificate returns a certificate for the given domain, generating one if needed.
// This implements tls.Config.GetCertificate
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := hello.ServerName
	if domain == "" {
		domain = "localhost"
	}

	// Check cache first
	m.cacheMu.RLock()
	if cert, ok := m.cache[domain]; ok {
		m.cacheMu.RUnlock()
		return cert, nil
	}
	m.cacheMu.RUnlock()

	// Generate new certificate
	cert, err := m.generateCert(domain)
	if err != nil {
		return nil, err
	}

	// Cache it
	m.cacheMu.Lock()
	m.cache[domain] = cert
	m.cacheMu.Unlock()

	return cert, nil
}

// generateCert creates a new certificate for the given domain
func (m *Manager) generateCert(domain string) (*tls.Certificate, error) {
	// Generate private key for this cert
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"fireup"},
			CommonName:   domain,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	// Add SANs
	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{domain}
		// Also add wildcard for subdomains
		if domain != "localhost" {
			template.DNSNames = append(template.DNSNames, "*."+domain)
		}
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &privateKey.PublicKey, m.caKey)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER, m.caCert.Raw},
		PrivateKey:  privateKey,
	}

	// Parse the leaf certificate for the Leaf field
	leaf, err := x509.ParseCertificate(certDER)
	if err == nil {
		cert.Leaf = leaf
	}

	return cert, nil
}

// TLSConfig returns a tls.Config that uses dynamic certificate generation
func (m *Manager) TLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: m.GetCertificate,
	}
}
