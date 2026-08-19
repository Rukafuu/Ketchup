package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DownloadInfo contém informações sobre download para uma plataforma específica
type DownloadInfo struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size,omitempty"`
}

// ChannelInfo contém informações sobre um canal de release
type ChannelInfo struct {
	Version   string                 `json:"version"`
	Downloads map[string]DownloadInfo `json:"downloads"`
	Published string                 `json:"published,omitempty"`
	Notes     string                 `json:"notes,omitempty"`
}

// Manifest representa o manifest remoto de releases
type Manifest struct {
	Channels map[string]ChannelInfo `json:"channels,omitempty"`
	// Formato legado/compatibilidade: campos no root level
	Version   string                 `json:"version,omitempty"`
	Downloads map[string]DownloadInfo `json:"downloads,omitempty"`
}

// ManifestClient é responsável por buscar e parsear manifests remotos
type ManifestClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewManifestClient cria um novo cliente de manifest
func NewManifestClient(baseURL string) *ManifestClient {
	return &ManifestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// FetchManifest busca o manifest do servidor de releases
func (c *ManifestClient) FetchManifest(channel string) (*Manifest, error) {
	url := fmt.Sprintf("%s/latest.json", c.baseURL)
	
	// Suporte para query parameter de canal
	if channel != "" && channel != "stable" {
		url = fmt.Sprintf("%s?channel=%s", url, channel)
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ketchup-updater/1.0")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request failed with status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}
	
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}
	
	// Normaliza para formato com channels se necessário
	if manifest.Channels == nil {
		// Formato legado: dados no root level
		// Assume como channel stable
		if manifest.Version != "" && manifest.Downloads != nil {
			manifest.Channels = map[string]ChannelInfo{
				"stable": {
					Version:   manifest.Version,
					Downloads: manifest.Downloads,
				},
			}
		}
	}
	
	return &manifest, nil
}

// GetChannelInfo retorna informações de um canal específico
func (m *Manifest) GetChannelInfo(channel string) (*ChannelInfo, bool) {
	if m.Channels == nil {
		return nil, false
	}
	
	info, exists := m.Channels[channel]
	return &info, exists
}

// GetDownloadInfo retorna informações de download para uma plataforma
func (c *ChannelInfo) GetDownloadInfo(platform string) (*DownloadInfo, bool) {
	if c.Downloads == nil {
		return nil, false
	}
	
	info, exists := c.Downloads[platform]
	return &info, exists
}
