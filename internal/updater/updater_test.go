package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name     string
		wantOS   string
		wantArch string
	}{
		{"linux-amd64", "linux", "amd64"},
		{"darwin-arm64", "darwin", "arm64"},
		{"windows-amd64", "windows", "amd64"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Nota: Este teste usa a plataforma real da máquina de teste
			// Para testes mais precisos, usaríamos mocks
			platform := DetectPlatform()
			
			if platform.OS == "" {
				t.Error("expected non-empty OS")
			}
			if platform.Arch == "" {
				t.Error("expected non-empty Arch")
			}
			if platform.Platform == "" {
				t.Error("expected non-empty Platform")
			}
		})
	}
}

func TestPlatformIsSupported(t *testing.T) {
	supportedPlatforms := []string{
		"windows-amd64",
		"linux-amd64",
		"darwin-arm64",
	}
	
	unsupportedPlatforms := []string{
		"solaris-sparc",
		"plan9-386",
	}
	
	for _, platform := range supportedPlatforms {
		p := &PlatformInfo{Platform: platform}
		if !p.IsSupported() {
			t.Errorf("expected %s to be supported", platform)
		}
	}
	
	for _, platform := range unsupportedPlatforms {
		p := &PlatformInfo{Platform: platform}
		if p.IsSupported() {
			t.Errorf("expected %s to be unsupported", platform)
		}
	}
}

func TestManifestParsing(t *testing.T) {
	manifestJSON := `{
		"channels": {
			"stable": {
				"version": "0.8.1",
				"downloads": {
					"linux-amd64": {
						"url": "https://releases.ketchup.dev/0.8.1/ketchup-linux-amd64",
						"sha256": "abc123"
					},
					"windows-amd64": {
						"url": "https://releases.ketchup.dev/0.8.1/ketchup-windows-amd64.exe",
						"sha256": "def456"
					}
				}
			},
			"beta": {
				"version": "0.9.0-beta.3",
				"downloads": {}
			}
		}
	}`
	
	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	
	// Verifica canal stable
	stableInfo, exists := manifest.GetChannelInfo("stable")
	if !exists {
		t.Fatal("expected stable channel to exist")
	}
	if stableInfo.Version != "0.8.1" {
		t.Errorf("expected stable version 0.8.1, got %s", stableInfo.Version)
	}
	
	// Verifica download linux-amd64
	linuxDownload, exists := stableInfo.GetDownloadInfo("linux-amd64")
	if !exists {
		t.Fatal("expected linux-amd64 download to exist")
	}
	if linuxDownload.SHA256 != "abc123" {
		t.Errorf("expected SHA256 abc123, got %s", linuxDownload.SHA256)
	}
	
	// Verifica canal beta
	betaInfo, exists := manifest.GetChannelInfo("beta")
	if !exists {
		t.Fatal("expected beta channel to exist")
	}
	if betaInfo.Version != "0.9.0-beta.3" {
		t.Errorf("expected beta version 0.9.0-beta.3, got %s", betaInfo.Version)
	}
}

func TestLegacyManifestParsing(t *testing.T) {
	// Formato legado sem channels
	legacyJSON := `{
		"version": "0.4.5",
		"downloads": {
			"linux-amd64": {
				"url": "https://example.com/ketchup",
				"sha256": "hash123"
			}
		}
	}`
	
	var manifest Manifest
	if err := json.Unmarshal([]byte(legacyJSON), &manifest); err != nil {
		t.Fatalf("failed to parse legacy manifest: %v", err)
	}
	
	// Deveria ser normalizado para formato com channels
	if manifest.Channels == nil {
		// A normalização ocorre no FetchManifest, não no Unmarshal
		// Então aqui esperamos os campos no root level
		if manifest.Version != "0.4.5" {
			t.Errorf("expected version 0.4.5, got %s", manifest.Version)
		}
	}
}

func TestManifestClient_FetchManifest(t *testing.T) {
	expectedManifest := `{
		"channels": {
			"stable": {
				"version": "1.0.0",
				"downloads": {
					"linux-amd64": {
						"url": "https://example.com/ketchup",
						"sha256": "abc123"
					}
				}
			}
		}
	}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			t.Errorf("expected /latest.json, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedManifest))
	}))
	defer server.Close()
	
	client := NewManifestClient(server.URL)
	manifest, err := client.FetchManifest("stable")
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if manifest.Channels == nil {
		t.Fatal("expected channels to be populated")
	}
	
	stable, exists := manifest.GetChannelInfo("stable")
	if !exists {
		t.Fatal("expected stable channel")
	}
	
	if stable.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", stable.Version)
	}
}

func TestManifestClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	client := NewManifestClient(server.URL)
	_, err := client.FetchManifest("stable")
	
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestChecksumValidator(t *testing.T) {
	// Cria arquivo temporário com conteúdo conhecido
	content := []byte("test content for checksum validation")
	tmpFile, err := os.CreateTemp("", "ketchup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()
	
	// Calcula checksum
	validator := NewChecksumValidator()
	calculated, err := validator.CalculateSHA256(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to calculate checksum: %v", err)
	}
	
	if calculated == "" {
		t.Error("expected non-empty checksum")
	}
	
	// Verifica checksum correto
	if err := validator.VerifyFile(tmpFile.Name(), calculated); err != nil {
		t.Errorf("expected valid checksum, got error: %v", err)
	}
	
	// Verifica checksum incorreto
	if err := validator.VerifyFile(tmpFile.Name(), "wronghash"); err == nil {
		t.Error("expected checksum mismatch error")
	} else if _, ok := err.(*ChecksumMismatchError); !ok {
		t.Errorf("expected ChecksumMismatchError, got %T", err)
	}
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		current  string
		remote   string
		newer    bool
		hasError bool
	}{
		{"0.8.0", "0.8.1", true, false},
		{"0.8.1", "0.8.0", false, false},
		{"0.8.0", "0.8.0", false, false},
		{"0.8.0", "1.0.0", true, false},
		{"1.0.0", "0.9.9", false, false},
		{"0.8.0", "0.9.0-beta.1", true, false},
		{"v0.8.0", "0.8.1", true, false},
		{"0.8.0", "v0.8.1", true, false},
		{"invalid", "0.8.1", false, true},
		{"0.8.0", "invalid", false, true},
	}
	
	for _, tt := range tests {
		u := &Updater{currentVersion: tt.current}
		newer, err := u.isNewerVersion(tt.remote)
		
		if tt.hasError && err == nil {
			t.Errorf("expected error for current=%s remote=%s", tt.current, tt.remote)
		}
		if !tt.hasError && err != nil {
			t.Errorf("unexpected error for current=%s remote=%s: %v", tt.current, tt.remote, err)
		}
		if newer != tt.newer {
			t.Errorf("current=%s remote=%s: expected newer=%v, got %v", 
				tt.current, tt.remote, tt.newer, newer)
		}
	}
}

func TestPlatformNotSupportedError(t *testing.T) {
	err := &PlatformNotSupportedError{
		Platform: "unknown-arch",
		AvailablePlatforms: []string{"linux-amd64", "windows-amd64"},
	}
	
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestGetExecutableName(t *testing.T) {
	tests := []struct {
		os       string
		expected string
	}{
		{"windows", "ketchup.exe"},
		{"linux", "ketchup"},
		{"darwin", "ketchup"},
	}
	
	for _, tt := range tests {
		p := &PlatformInfo{OS: tt.os, Platform: tt.os + "-amd64"}
		result := p.GetExecutableName()
		if result != tt.expected {
			t.Errorf("os %s: expected %s, got %s", tt.os, tt.expected, result)
		}
	}
}
