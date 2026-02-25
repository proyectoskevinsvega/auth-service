package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// CertificateManager handles programmetic generation of X.509 certificates for mTLS
type CertificateManager struct {
	keyDir string
}

// NewCertificateManager creates a new CertificateManager
func NewCertificateManager(keyDir string) *CertificateManager {
	return &CertificateManager{keyDir: keyDir}
}

// GenerateMtlsSetup generates CA and Server certificates if they don't exist
func (m *CertificateManager) GenerateMtlsSetup(serverCN string) error {
	caCertPath := fmt.Sprintf("%s/ca.pem", m.keyDir)
	caKeyPath := fmt.Sprintf("%s/ca-key.pem", m.keyDir)
	serverCertPath := fmt.Sprintf("%s/server.pem", m.keyDir)
	serverKeyPath := fmt.Sprintf("%s/server-key.pem", m.keyDir)

	// 1. Check if CA exists
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		fmt.Printf("Generating Root CA in %s...\n", m.keyDir)
		caCert, caKey, err := m.CreateCA("VerterCloud Root CA")
		if err != nil {
			return err
		}
		if err := m.SavePEM(caCertPath, "CERTIFICATE", caCert); err != nil {
			return err
		}
		if err := m.SavePEM(caKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caKey)); err != nil {
			return err
		}
	}

	// 2. Check if Server Cert exists
	if _, err := os.Stat(serverCertPath); os.IsNotExist(err) {
		fmt.Printf("Generating Server Certificate for %s...\n", serverCN)
		// Load CA
		caCert, caKey, err := m.LoadCA(caCertPath, caKeyPath)
		if err != nil {
			return err
		}

		serverCert, serverKey, err := m.CreateCert(serverCN, caCert, caKey, true)
		if err != nil {
			return err
		}

		if err := m.SavePEM(serverCertPath, "CERTIFICATE", serverCert); err != nil {
			return err
		}
		if err := m.SavePEM(serverKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey)); err != nil {
			return err
		}
	}

	return nil
}

// CreateCA creates a new Root Certificate Authority
func (m *CertificateManager) CreateCA(commonName string) ([]byte, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	return derBytes, priv, nil
}

// CreateCert creates a new certificate signed by the CA
func (m *CertificateManager) CreateCert(commonName string, caCert *x509.Certificate, caKey *rsa.PrivateKey, isServer bool) ([]byte, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	if isServer {
		template.DNSNames = []string{"localhost", commonName}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &priv.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	return derBytes, priv, nil
}

// LoadCA loads the CA certificate and private key from files
func (m *CertificateManager) LoadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(certPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return caCert, caKey, nil
}

// SavePEM saves data to a PEM encoded file
func (m *CertificateManager) SavePEM(path, pemType string, data []byte) error {
	pemFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer pemFile.Close()

	return pem.Encode(pemFile, &pem.Block{
		Type:  pemType,
		Bytes: data,
	})
}

// GenerateClientCert generates a new client certificate signed by the existing CA
func (m *CertificateManager) GenerateClientCert(clientName string) error {
	caCertPath := fmt.Sprintf("%s/ca.pem", m.keyDir)
	caKeyPath := fmt.Sprintf("%s/ca-key.pem", m.keyDir)

	caCert, caKey, err := m.LoadCA(caCertPath, caKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load CA: %w (did you run GenerateMtlsSetup first?)", err)
	}

	certBytes, key, err := m.CreateCert(clientName, caCert, caKey, false)
	if err != nil {
		return err
	}

	if err := m.SavePEM(fmt.Sprintf("%s/%s.pem", m.keyDir, clientName), "CERTIFICATE", certBytes); err != nil {
		return err
	}
	if err := m.SavePEM(fmt.Sprintf("%s/%s-key.pem", m.keyDir, clientName), "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)); err != nil {
		return err
	}

	return nil
}
