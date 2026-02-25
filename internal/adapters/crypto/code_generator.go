package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type CodeGenerator struct{}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// GenerateNumericCode genera un código numérico de N dígitos
func (g *CodeGenerator) GenerateNumericCode(digits int) (string, error) {
	if digits < 4 || digits > 10 {
		return "", fmt.Errorf("digits must be between 4 and 10")
	}

	// Generar un número entre 0 y 10^digits - 1
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	// Formatear con ceros a la izquierda si es necesario
	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, n), nil
}
