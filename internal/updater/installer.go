package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Installer é responsável por instalar o novo binário de forma segura
type Installer struct {
	execPath string // caminho do executável atual
}

// NewInstaller cria um novo instalador
func NewInstaller() (*Installer, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	
	return &Installer{
		execPath: execPath,
	}, nil
}

// Install instala um novo binário a partir de um arquivo temporário
// Retorna true se for necessário reiniciar o processo
func (i *Installer) Install(newBinaryPath string) (restartRequired bool, err error) {
	switch runtime.GOOS {
	case "windows":
		return i.installWindows(newBinaryPath)
	case "darwin", "linux":
		return i.installUnix(newBinaryPath)
	default:
		return false, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// installUnix lida com instalação em sistemas Unix-like (Linux, macOS)
func (i *Installer) installUnix(newBinaryPath string) (bool, error) {
	execDir := filepath.Dir(i.execPath)
	execName := filepath.Base(i.execPath)
	
	// Caminhos para backup e novo binário
	oldPath := i.execPath + ".old"
	tempPath := filepath.Join(execDir, "."+execName+".new")
	
	// Passo 1: Copia novo binário para local temporário no mesmo diretório
	if err := copyFile(newBinaryPath, tempPath); err != nil {
		return false, fmt.Errorf("failed to copy new binary: %w", err)
	}
	
	// Garante limpeza em caso de erro
	success := false
	defer func() {
		if !success {
			os.Remove(tempPath)
		}
	}()
	
	// Passo 2: Define permissões de execução
	if err := os.Chmod(tempPath, 0755); err != nil {
		return false, fmt.Errorf("failed to set executable permissions: %w", err)
	}
	
	// Passo 3: Cria backup do binário atual (se existir)
	if _, err := os.Stat(i.execPath); err == nil {
		if err := os.Rename(i.execPath, oldPath); err != nil {
			return false, fmt.Errorf("failed to backup old binary: %w", err)
		}
	}
	
	// Passo 4: Move novo binário para localização final
	if err := os.Rename(tempPath, i.execPath); err != nil {
		// Tenta restaurar backup
		if _, statErr := os.Stat(oldPath); statErr == nil {
			os.Rename(oldPath, i.execPath)
		}
		return false, fmt.Errorf("failed to install new binary: %w", err)
	}
	
	success = true
	
	// Remove backup antigo após sucesso
	if _, err := os.Stat(oldPath); err == nil {
		os.Remove(oldPath)
	}
	
	// Em Unix, podemos substituir o processo em execução
	// Mas para segurança, retornamos que restart é necessário
	// A extensão pode decidir quando reiniciar
	return true, nil
}

// installWindows lida com instalação no Windows
// No Windows não podemos sobrescrever um executável em execução
func (i *Installer) installWindows(newBinaryPath string) (bool, error) {
	execDir := filepath.Dir(i.execPath)
	
	// Caminho para o novo binário
	newPath := i.execPath + ".new"
	oldPath := i.execPath + ".old"
	
	// Passo 1: Copia novo binário para .new
	if err := copyFile(newBinaryPath, newPath); err != nil {
		return false, fmt.Errorf("failed to copy new binary: %w", err)
	}
	
	// Passo 2: Cria script batch para substituição após fechamento
	scriptPath := filepath.Join(execDir, "update.bat")
	scriptContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
if exist "%s" (
    move /y "%s" "%s.old" >nul
    if exist "%s.new" (
        move /y "%s.new" "%s" >nul
        start "" "%s"
    )
)
del "%s"
`, i.execPath, i.execPath, i.execPath, i.execPath, i.execPath, i.execPath, i.execPath, scriptPath)
	
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		os.Remove(newPath)
		return false, fmt.Errorf("failed to create update script: %w", err)
	}
	
	// Passo 3: Agenda execução do script após este processo fechar
	// Usamos cmd.exe /c start para executar assincronamente
	cmd := exec.Command("cmd.exe", "/c", "start", "/b", scriptPath)
	
	if err := cmd.Start(); err != nil {
		os.Remove(newPath)
		os.Remove(scriptPath)
		return false, fmt.Errorf("failed to schedule update: %w", err)
	}
	
	// Não esperamos o script terminar
	cmd.Process.Release()
	
	// Suprimir variáveis não usadas
	_ = oldPath
	
	return true, nil
}

// Rollback reverte para a versão anterior (backup)
func (i *Installer) Rollback() error {
	oldPath := i.execPath + ".old"
	
	// Verifica se backup existe
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup available for rollback")
	}
	
	// Remove binário atual (potencialmente corrompido)
	if _, err := os.Stat(i.execPath); err == nil {
		if err := os.Remove(i.execPath); err != nil {
			return fmt.Errorf("failed to remove corrupted binary: %w", err)
		}
	}
	
	// Restaura backup
	if err := os.Rename(oldPath, i.execPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	// Restaura permissões em Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(i.execPath, 0755); err != nil {
			return fmt.Errorf("failed to restore permissions: %w", err)
		}
	}
	
	return nil
}

// GetExecutablePath retorna o caminho do executável atual
func (i *Installer) GetExecutablePath() string {
	return i.execPath
}

// copyFile copia um arquivo de origem para destino
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}
	
	return nil
}
