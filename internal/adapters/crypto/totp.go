package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/pquerna/otp/totp"
)

type TOTPService struct {
	issuer string
}

func NewTOTPService(issuer string) *TOTPService {
	return &TOTPService{
		issuer: issuer,
	}
}

func (s *TOTPService) Generate(email string) (secret, qrCode string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: email,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Generate QR code
	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate QR code image: %w", err)
	}

	// Convert image to base64 data URI
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", fmt.Errorf("failed to encode QR code: %w", err)
	}

	qrCodeBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	qrCodeDataURI := fmt.Sprintf("data:image/png;base64,%s", qrCodeBase64)

	return key.Secret(), qrCodeDataURI, nil
}

func (s *TOTPService) Verify(code, secret string) (bool, error) {
	return totp.Validate(code, secret), nil
}
