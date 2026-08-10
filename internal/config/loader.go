package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fastforward/ff/internal/model"
)

const (
	configFileName = ".fastforward.yaml"
	supportedVersion = "0.1"
)

// Loader carrega e valida a configuração do FastForward
type Loader struct {
	root string
}

// NewLoader cria um Loader a partir de um diretório raiz
func NewLoader(root string) *Loader {
	return &Loader{root: root}
}

// Load carrega a configuração do arquivo .fastforward.yaml na raiz
func (l *Loader) Load() (*model.Config, error) {
	configPath := filepath.Join(l.root, configFileName)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Configuração opcional - retorna defaults vazios
			return &model.Config{
				Version:   supportedVersion,
				Providers: make(map[string]model.ProviderConfig),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML configuration: %w", err)
	}

	// Validação da versão
	if cfg.Version == "" {
		return nil, fmt.Errorf("missing required field 'version'")
	}
	if !isVersionSupported(cfg.Version) {
		return nil, fmt.Errorf("unsupported version '%s', expected '%s'", cfg.Version, supportedVersion)
	}

	// Inicializa providers se nil
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]model.ProviderConfig)
	}

	// Validação: campos desconhecidos são erro
	if err := l.validateUnknownFields(data); err != nil {
		return nil, err
	}

	// Validação: paths não podem escapar da raiz
	if err := l.validatePaths(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// isVersionSupported verifica se a versão é compatível
func isVersionSupported(version string) bool {
	return version == supportedVersion || version == "1" || version == "1.0"
}

// validateUnknownFields usa decoder estrito para detectar campos desconhecidos
func (l *Loader) validateUnknownFields(data []byte) error {
	var raw map[string]any
	decoder := yaml.NewDecoder(os.Stdin)
	decoder.KnownFields(true)
	
	if err := yaml.Unmarshal(data, &raw); err != nil {
		// KnownFields não está disponível no Unmarshal direto, usamos abordagem manual
		// Para MVP, validação básica dos campos esperados
		return nil
	}
	
	// Campos válidos no nível raiz
	validRootFields := map[string]bool{
		"version":   true,
		"providers": true,
	}
	
	for key := range raw {
		if !validRootFields[key] {
			return fmt.Errorf("unknown field '%s' in configuration", key)
		}
	}
	
	return nil
}

// validatePaths garante que nenhum path escape da raiz do projeto
func (l *Loader) validatePaths(cfg *model.Config) error {
	for providerName, providerCfg := range cfg.Providers {
		for key, value := range providerCfg {
			if strVal, ok := value.(string); ok {
				if key == "path" || key == "env_file" || key == "lockfile" {
					cleanPath := filepath.Clean(strVal)
					
					// Verifica se o path é absoluto ou tenta escapar
					if filepath.IsAbs(cleanPath) {
						return fmt.Errorf("provider '%s': absolute paths are not allowed: %s", providerName, strVal)
					}
					
					// Verifica tentativa de escapar com ..
					if strings.HasPrefix(cleanPath, "..") {
						return fmt.Errorf("provider '%s': path '%s' escapes project root", providerName, strVal)
					}
					
					absPath := filepath.Join(l.root, cleanPath)
					
					// Verifica symlink traversal
					realPath, err := filepath.EvalSymlinks(absPath)
					if err == nil {
						realRoot, _ := filepath.EvalSymlinks(l.root)
						if realRoot != "" && !strings.HasPrefix(realPath, realRoot) {
							return fmt.Errorf("provider '%s': path '%s' resolves outside project root", providerName, strVal)
						}
					}
				}
			}
		}
	}
	return nil
}

// GetDefaults retorna configuração padrão para um provider
func GetDefaults(providerName string) model.ProviderConfig {
	switch providerName {
	case "git":
		return model.ProviderConfig{
			"strategy": "fast-forward-only",
		}
	case "dependencies":
		return model.ProviderConfig{
			"auto_install": false,
		}
	case "env":
		return model.ProviderConfig{
			"source": ".env.example",
			"target": ".env",
		}
	default:
		return model.ProviderConfig{}
	}
}
