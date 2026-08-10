package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// FileExists verifica se um arquivo existe
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists verifica se um diretório existe
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsEmpty verifica se um diretório está vazio
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// HashFile calcula o hash SHA256 de um arquivo
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ReadFileSafe lê um arquivo com limite de tamanho para evitar DoS
func ReadFileSafe(path string, maxSize int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if maxSize > 0 && info.Size() > maxSize {
		return nil, ErrFileTooLarge
	}

	return io.ReadAll(f)
}

// ErrFileTooLarge é retornado quando um arquivo excede o limite
var ErrFileTooLarge = &fileTooLargeError{}

type fileTooLargeError struct{}

func (e *fileTooLargeError) Error() string {
	return "file exceeds maximum allowed size"
}

// WalkFiles percorre arquivos em um diretório
func WalkFiles(root string, fn func(path string, info os.FileInfo) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return fn(path, info)
	})
}

// EnsureDir cria um diretório se não existir
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// TempFile cria um arquivo temporário no mesmo diretório do original
func TempFile(originalPath string) (*os.File, error) {
	dir := filepath.Dir(originalPath)
	pattern := "." + filepath.Base(originalPath) + ".tmp.*"
	return os.CreateTemp(dir, pattern)
}
