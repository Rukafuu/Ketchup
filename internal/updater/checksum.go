package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ChecksumValidator é responsável por validar checksums SHA-256
type ChecksumValidator struct{}

// NewChecksumValidator cria um novo validador de checksum
func NewChecksumValidator() *ChecksumValidator {
	return &ChecksumValidator{}
}

// CalculateSHA256 calcula o SHA-256 de um arquivo
func (v *ChecksumValidator) CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	
	// Usa buffer para evitar carregar arquivo inteiro na memória
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if _, hashErr := hasher.Write(buf[:n]); hashErr != nil {
				return "", fmt.Errorf("failed to write to hasher: %w", hashErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyFile verifica se o SHA-256 de um arquivo corresponde ao esperado
func (v *ChecksumValidator) VerifyFile(filePath, expectedSHA256 string) error {
	calculated, err := v.CalculateSHA256(filePath)
	if err != nil {
		return err
	}

	if calculated != expectedSHA256 {
		return &ChecksumMismatchError{
			Expected: expectedSHA256,
			Calculated: calculated,
			File:     filePath,
		}
	}

	return nil
}

// ChecksumMismatchError é retornado quando o checksum não corresponde
type ChecksumMismatchError struct {
	Expected   string
	Calculated string
	File       string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf(
		"checksum mismatch for %s: expected %s, got %s",
		e.File, e.Expected, e.Calculated,
	)
}

// ValidateBytes calcula e valida SHA-256 de bytes em memória
func (v *ChecksumValidator) ValidateBytes(data []byte, expectedSHA256 string) error {
	hasher := sha256.New()
	if _, err := hasher.Write(data); err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}
	
	calculated := hex.EncodeToString(hasher.Sum(nil))
	
	if calculated != expectedSHA256 {
		return &ChecksumMismatchError{
			Expected:   expectedSHA256,
			Calculated: calculated,
			File:       "<memory>",
		}
	}
	
	return nil
}
