package usecase

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/vertercloud/auth-service/internal/adapters/crypto"
)

// M2MUseCase handles Machine-to-Machine certificate issuance
type M2MUseCase struct {
	certManager *crypto.CertificateManager
	logger      zerolog.Logger
}

// NewM2MUseCase creates a new M2MUseCase
func NewM2MUseCase(certManager *crypto.CertificateManager, logger zerolog.Logger) *M2MUseCase {
	return &M2MUseCase{
		certManager: certManager,
		logger:      logger,
	}
}

// ClientCertificateResponse represents the issued certificate and key
type ClientCertificateResponse struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	CACert      string `json:"ca_certificate"`
}

// IssueCertificate issues a new mTLS certificate for a client
func (uc *M2MUseCase) IssueCertificate(ctx context.Context, clientName string) (*ClientCertificateResponse, error) {
	uc.logger.Info().Str("client_name", clientName).Msg("issuing new client certificate")

	cert, key, err := uc.certManager.IssueClientCertificate(clientName)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Also provide the CA certificate so the client can verify the server
	// In a real scenario, this might be a public URL, but here we return the file content
	caCert, _, err := uc.certManager.LoadCA("./keys/ca.pem", "./keys/ca-key.pem")
	if err != nil {
		uc.logger.Error().Err(err).Msg("failed to load CA for client response")
		// We can still return the cert/key even if CA fails, though CA is needed for full verification
	}

	// Convert CA to PEM (simplest way is to read the file again or use the LoadCA result)
	// For production, we'd have a public CA endpoint.
	caPEM := ""
	caData, err := uc.certManager.SaveToMemory(caCert) // Let's add this helper
	if err == nil {
		caPEM = string(caData)
	}

	return &ClientCertificateResponse{
		Certificate: cert,
		PrivateKey:  key,
		CACert:      caPEM,
	}, nil
}

// SignClientCSR signs a CSR provided by the client
func (uc *M2MUseCase) SignClientCSR(ctx context.Context, csrPEM string) (*ClientCertificateResponse, error) {
	uc.logger.Info().Msg("signing client-provided CSR")

	cert, err := uc.certManager.SignCSR(csrPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to sign CSR: %w", err)
	}

	// Load CA certificate for the response
	caCert, _, err := uc.certManager.LoadCA("./keys/ca.pem", "./keys/ca-key.pem")
	caPEM := ""
	if err == nil {
		caData, err := uc.certManager.SaveToMemory(caCert)
		if err == nil {
			caPEM = string(caData)
		}
	}

	return &ClientCertificateResponse{
		Certificate: cert,
		PrivateKey:  "", // No private key returned in CSR flow
		CACert:      caPEM,
	}, nil
}
