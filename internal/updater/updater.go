package updater

import (
	"fmt"
	"log"
	"os"
	"strings"
	
	"github.com/hashicorp/go-version"
)

// VersionInfo contém informações sobre a versão atual e disponível
type VersionInfo struct {
	Current      string // versão atual instalada
	Latest       string // última versão disponível no canal
	Channel      string // canal atual (stable, beta, nightly)
	UpdateAvailable bool
	Platform     string // plataforma detectada
}

// Updater é o módulo principal de atualização
type Updater struct {
	baseURL        string
	channel        string
	currentVersion string
	platform       *PlatformInfo
	logger         *log.Logger
	autoUpdate     bool
}

// UpdaterConfig configurações do updater
type UpdaterConfig struct {
	BaseURL        string
	Channel        string
	CurrentVersion string
	AutoUpdate     bool
	Logger         *log.Logger
}

// NewUpdater cria um novo updater com as configurações especificadas
func NewUpdater(config UpdaterConfig) *Updater {
	if config.BaseURL == "" {
		config.BaseURL = "https://releases.ketchup.dev"
	}
	if config.Channel == "" {
		config.Channel = "stable"
	}
	if config.Logger == nil {
		// Logger silencioso por padrão
		config.Logger = log.New(os.Stderr, "", 0)
	}
	
	return &Updater{
		baseURL:        config.BaseURL,
		channel:        config.Channel,
		currentVersion: config.CurrentVersion,
		platform:       DetectPlatform(),
		logger:         config.Logger,
		autoUpdate:     config.AutoUpdate,
	}
}

// CheckForUpdate verifica se há uma nova versão disponível
// Retorna nil sem erro se não houver update ou se o servidor estiver indisponível
func (u *Updater) CheckForUpdate() (*VersionInfo, error) {
	u.log("[update] checking %s channel", u.channel)
	u.log("[update] local version: %s", u.currentVersion)
	u.log("[update] platform: %s", u.platform.Platform)
	
	// Busca manifest remoto
	client := NewManifestClient(u.baseURL)
	manifest, err := client.FetchManifest(u.channel)
	if err != nil {
		// Falha ao buscar manifest não deve impedir operação
		u.log("[update] warning: failed to fetch manifest: %v", err)
		return nil, nil
	}
	
	// Obtém informações do canal
	channelInfo, exists := manifest.GetChannelInfo(u.channel)
	if !exists {
		u.log("[update] warning: channel %s not found in manifest", u.channel)
		return nil, nil
	}
	
	u.log("[update] remote version: %s", channelInfo.Version)
	
	// Compara versões usando SemVer
	updateAvailable, err := u.isNewerVersion(channelInfo.Version)
	if err != nil {
		u.log("[update] warning: failed to compare versions: %v", err)
		return nil, nil
	}
	
	versionInfo := &VersionInfo{
		Current:       u.currentVersion,
		Latest:        channelInfo.Version,
		Channel:       u.channel,
		UpdateAvailable: updateAvailable,
		Platform:      u.platform.Platform,
	}
	
	if updateAvailable {
		u.log("[update] update available: %s -> %s", u.currentVersion, channelInfo.Version)
	} else {
		u.log("[update] already on latest version")
	}
	
	return versionInfo, nil
}

// DownloadUpdate baixa o update para um arquivo temporário
// Retorna o caminho do arquivo temporário e informações de download
func (u *Updater) DownloadUpdate(manifest *Manifest, channel string) (string, error) {
	channelInfo, exists := manifest.GetChannelInfo(channel)
	if !exists {
		return "", fmt.Errorf("channel %s not found", channel)
	}
	
	downloadInfo, exists := channelInfo.GetDownloadInfo(u.platform.Platform)
	if !exists {
		return "", &PlatformNotSupportedError{
			Platform: u.platform.Platform,
			AvailablePlatforms: getAvailablePlatforms(channelInfo.Downloads),
		}
	}
	
	u.log("[update] downloading from: %s", downloadInfo.URL)
	u.log("[update] expected SHA256: %s", downloadInfo.SHA256)
	
	// Baixa para arquivo temporário
	downloader := NewDownloader()
	tempPath, err := downloader.DownloadToTemp(downloadInfo.URL)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	
	u.log("[update] downloaded to: %s", tempPath)
	
	// Valida checksum
	validator := NewChecksumValidator()
	if err := validator.VerifyFile(tempPath, downloadInfo.SHA256); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("checksum validation failed: %w", err)
	}
	
	u.log("[update] checksum verified")
	
	return tempPath, nil
}

// InstallUpdate instala um update a partir de um arquivo temporário
// Retorna true se for necessário reiniciar o processo
func (u *Updater) InstallUpdate(tempBinaryPath string) (bool, error) {
	u.log("[update] installing update...")
	
	installer, err := NewInstaller()
	if err != nil {
		return false, fmt.Errorf("failed to create installer: %w", err)
	}
	
	restartRequired, err := installer.Install(tempBinaryPath)
	if err != nil {
		// Tenta rollback em caso de falha
		u.log("[update] installation failed, attempting rollback...")
		if rollbackErr := installer.Rollback(); rollbackErr != nil {
			u.log("[update] rollback also failed: %v", rollbackErr)
		}
		return false, fmt.Errorf("failed to install update: %w", err)
	}
	
	u.log("[update] update installed successfully")
	
	return restartRequired, nil
}

// DoUpdate executa todo o fluxo de update: check -> download -> install
func (u *Updater) DoUpdate() (updated bool, restartRequired bool, err error) {
	// Step 1: Check for update
	versionInfo, err := u.CheckForUpdate()
	if err != nil {
		return false, false, err
	}
	
	if versionInfo == nil || !versionInfo.UpdateAvailable {
		return false, false, nil
	}
	
	// Step 2: Fetch manifest
	client := NewManifestClient(u.baseURL)
	manifest, err := client.FetchManifest(u.channel)
	if err != nil {
		return false, false, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	
	// Step 3: Download update
	tempPath, err := u.DownloadUpdate(manifest, u.channel)
	if err != nil {
		return false, false, err
	}
	
	// Garante limpeza do arquivo temporário após uso
	defer func() {
		if _, statErr := os.Stat(tempPath); statErr == nil {
			os.Remove(tempPath)
		}
	}()
	
	// Step 4: Install update
	restartRequired, err = u.InstallUpdate(tempPath)
	if err != nil {
		return false, restartRequired, err
	}
	
	return true, restartRequired, nil
}

// GetPlatform retorna a plataforma detectada
func (u *Updater) GetPlatform() *PlatformInfo {
	return u.platform
}

// GetCurrentVersion retorna a versão atual
func (u *Updater) GetCurrentVersion() string {
	return u.currentVersion
}

// GetChannel retorna o canal atual
func (u *Updater) GetChannel() string {
	return u.channel
}

// isNewerVersion compara versões usando SemVer
func (u *Updater) isNewerVersion(remote string) (bool, error) {
	currentVer, err := version.NewVersion(strings.TrimPrefix(u.currentVersion, "v"))
	if err != nil {
		return false, fmt.Errorf("invalid current version %s: %w", u.currentVersion, err)
	}
	
	remoteVer, err := version.NewVersion(strings.TrimPrefix(remote, "v"))
	if err != nil {
		return false, fmt.Errorf("invalid remote version %s: %w", remote, err)
	}
	
	return remoteVer.GreaterThan(currentVer), nil
}

// getAvailablePlatforms extrai lista de plataformas disponíveis do manifest
func getAvailablePlatforms(downloads map[string]DownloadInfo) []string {
	platforms := make([]string, 0, len(downloads))
	for platform := range downloads {
		platforms = append(platforms, platform)
	}
	return platforms
}

// PlatformNotSupportedError é retornado quando a plataforma não tem build disponível
type PlatformNotSupportedError struct {
	Platform           string
	AvailablePlatforms []string
}

func (e *PlatformNotSupportedError) Error() string {
	return fmt.Sprintf(
		"platform %s is not supported. Available platforms: %v",
		e.Platform, e.AvailablePlatforms,
	)
}

// log imprime mensagem de log se logger estiver configurado
func (u *Updater) log(format string, args ...interface{}) {
	if u.logger != nil {
		u.logger.Printf(format, args...)
	}
}
