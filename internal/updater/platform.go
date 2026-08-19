package updater

import (
	"fmt"
	"runtime"
)

// PlatformInfo contém informações sobre a plataforma atual
type PlatformInfo struct {
	OS       string
	Arch     string
	Platform string // combinação OS-arch, ex: "windows-amd64"
}

// DetectPlatform detecta automaticamente o sistema operacional e arquitetura
// e retorna uma chave de plataforma compatível com o manifest de releases
func DetectPlatform() *PlatformInfo {
	os := normalizeOS(runtime.GOOS)
	arch := normalizeArch(runtime.GOARCH, os)
	
	return &PlatformInfo{
		OS:       os,
		Arch:     arch,
		Platform: fmt.Sprintf("%s-%s", os, arch),
	}
}

// normalizeOS normaliza o GOOS para nossas convenções de release
func normalizeOS(goos string) string {
	switch goos {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	case "freebsd":
		return "freebsd"
	case "openbsd":
		return "openbsd"
	case "netbsd":
		return "netbsd"
	default:
		return goos
	}
}

// normalizeArch normaliza o GOARCH para nossas convenções de release
// Considera combinações válidas de OS/arquitetura
func normalizeArch(goarch, goos string) string {
	switch goarch {
	case "amd64":
		return "amd64"
	case "arm64":
		// arm64 é suportado em todas as plataformas principais
		return "arm64"
	case "386":
		// 386 apenas para Windows e Linux
		if goos == "windows" || goos == "linux" {
			return "386"
		}
	case "arm":
		// ARM v7+ para Linux
		if goos == "linux" {
			return "arm"
		}
	}
	
	// Fallback para amd64 como arquitetura mais comum
	// O caller deve verificar se a plataforma é realmente suportada
	if goos == "darwin" {
		// macOS moderno usa arm64 (Apple Silicon) ou amd64 (Intel)
		// Se não detectamos corretamente, assumimos amd64 como fallback seguro
		return "amd64"
	}
	
	return goarch
}

// GetExecutableName retorna o nome do executável para a plataforma
func (p *PlatformInfo) GetExecutableName() string {
	baseName := "ketchup"
	if p.OS == "windows" {
		baseName = "ketchup.exe"
	}
	return baseName
}

// IsSupported verifica se a plataforma atual é suportada
func (p *PlatformInfo) IsSupported() bool {
	supportedPlatforms := map[string]bool{
		"windows-amd64": true,
		"windows-386":   true,
		"linux-amd64":   true,
		"linux-arm64":   true,
		"linux-386":     true,
		"linux-arm":     true,
		"darwin-amd64":  true,
		"darwin-arm64":  true,
	}
	
	return supportedPlatforms[p.Platform]
}

// String retorna uma representação legível da plataforma
func (p *PlatformInfo) String() string {
	return p.Platform
}
