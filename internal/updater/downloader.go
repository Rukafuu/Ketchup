package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Downloader é responsável por baixar arquivos de update
type Downloader struct {
	httpClient *http.Client
	progressCb ProgressCallback
}

// ProgressCallback é chamado periodicamente durante o download
type ProgressCallback func(downloaded, total int64)

// NewDownloader cria um novo downloader
func NewDownloader() *Downloader {
	return &Downloader{
		httpClient: &http.Client{},
	}
}

// WithProgressCallback define um callback para progresso do download
func (d *Downloader) WithProgressCallback(cb ProgressCallback) *Downloader {
	d.progressCb = cb
	return d
}

// DownloadFile baixa um arquivo da URL especificada para o destino
func (d *Downloader) DownloadFile(url, destPath string) error {
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	// Cria diretório de destino se necessário
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Cria arquivo temporário (.part extension)
	tempPath := destPath + ".part"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	
	// Garante que o arquivo seja fechado e removido em caso de erro
	success := false
	defer func() {
		tempFile.Close()
		if !success {
			os.Remove(tempPath)
		}
	}()
	
	var written int64
	if d.progressCb != nil && resp.ContentLength > 0 {
		// Download com progresso
		buf := make([]byte, 32*1024) // 32KB buffer
		total := resp.ContentLength
		
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := tempFile.Write(buf[:n]); writeErr != nil {
					return fmt.Errorf("failed to write to temp file: %w", writeErr)
				}
				written += int64(n)
				d.progressCb(written, total)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read from response: %w", err)
			}
		}
	} else {
		// Download simples sem progresso
		written, err = io.Copy(tempFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save download: %w", err)
		}
	}
	
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	
	// Renomeia arquivo temporário para nome final
	if err := os.Rename(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	
	success = true
	
	// Log opcional de debug
	_ = written
	
	return nil
}

// DownloadToTemp baixa um arquivo para um local temporário
func (d *Downloader) DownloadToTemp(url string) (string, error) {
	// Cria arquivo temporário no diretório padrão do sistema
	tempFile, err := os.CreateTemp("", "ketchup-update-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	
	success := false
	defer func() {
		tempFile.Close()
		if !success {
			os.Remove(tempPath)
		}
	}()
	
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save download: %w", err)
	}
	
	if err := tempFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync temp file: %w", err)
	}
	
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	
	success = true
	return tempPath, nil
}
