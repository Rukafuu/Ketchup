package env

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fastforward/ff/internal/model"
	"github.com/fastforward/ff/internal/providers"
)

// Provider é o provider de Environment do FastForward
type Provider struct{}

// NewProvider cria um novo provider Environment
func NewProvider() *Provider {
	return &Provider{}
}

// Name retorna o nome do provider
func (p *Provider) Name() string {
	return "env"
}

// Check compara variáveis de ambiente entre source e target
func (p *Provider) Check(ctx context.Context, req providers.CheckRequest) (model.Report, error) {
	report := model.Report{
		Provider:   p.Name(),
		Health:     model.HealthClean,
		ObservedAt: time.Now(),
		Facts:      make(model.Facts),
	}

	// Configuração padrão
	sourceFile := ".env.example"
	targetFile := ".env"

	if cfg, ok := req.Config["source"].(string); ok {
		sourceFile = cfg
	}
	if cfg, ok := req.Config["target"].(string); ok {
		targetFile = cfg
	}

	sourcePath := filepath.Join(req.Root, sourceFile)
	targetPath := filepath.Join(req.Root, targetFile)

	// Parse source
	sourceVars, sourceWarnings, err := p.parseEnvFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			report.Summary = fmt.Sprintf("Source file '%s' not found", sourceFile)
			report.Health = model.HealthUnknown
			return report, nil
		}
		report.Health = model.HealthUnknown
		report.Summary = fmt.Sprintf("Failed to read source file: %v", err)
		return report, err
	}

	report.Facts["source_file"] = sourceFile
	report.Facts["source_count"] = len(sourceVars)

	// Parse target (pode não existir)
	targetVars := make(map[string]bool)
	var targetWarnings []string
	if _, err := os.Stat(targetPath); err == nil {
		targetVars, targetWarnings, _ = p.parseEnvFile(targetPath)
	}

	report.Facts["target_file"] = targetFile
	report.Facts["target_count"] = len(targetVars)

	// Adiciona warnings aos findings
	for _, w := range sourceWarnings {
		report.Findings = append(report.Findings, model.Finding{
			Code:     "ENV_SOURCE_WARNING",
			Severity: model.SeverityWarning,
			Summary:  w,
		})
	}
	for _, w := range targetWarnings {
		report.Findings = append(report.Findings, model.Finding{
			Code:     "ENV_TARGET_WARNING",
			Severity: model.SeverityWarning,
			Summary:  w,
		})
	}

	// Categoriza variáveis
	var missing []string
	var localOnly []string

	for v := range sourceVars {
		if !targetVars[v] {
			missing = append(missing, v)
		}
	}

	for v := range targetVars {
		if !sourceVars[v] {
			localOnly = append(localOnly, v)
		}
	}

	sort.Strings(missing)
	sort.Strings(localOnly)

	report.Facts["missing"] = missing
	report.Facts["local_only"] = localOnly

	// Determina saúde
	if len(missing) > 0 || len(localOnly) > 0 {
		report.Health = model.HealthDrifted
		
		var findings []model.Finding
		
		if len(missing) > 0 {
			findings = append(findings, model.Finding{
				Code:     "ENV_MISSING_VARS",
				Severity: model.SeverityWarning,
				Summary:  fmt.Sprintf("%d variable(s) missing in target", len(missing)),
				Details:  p.toDetails(missing[:min(len(missing), 5)]), // Mostra apenas 5
			})
		}

		if len(localOnly) > 0 {
			findings = append(findings, model.Finding{
				Code:     "ENV_LOCAL_ONLY_VARS",
				Severity: model.SeverityInfo,
				Summary:  fmt.Sprintf("%d variable(s) only in target", len(localOnly)),
				Details:  p.toDetails(localOnly[:min(len(localOnly), 5)]),
			})
		}

		report.Findings = append(report.Findings, findings...)
		report.Summary = "Environment drift detected"
	} else {
		report.Summary = "Environment files are in sync"
	}

	return report, nil
}

// Plan gera operações para environment (sempre Manual na 0.1)
func (p *Provider) Plan(ctx context.Context, req providers.PlanRequest) ([]model.Operation, error) {
	var operations []model.Operation

	missing, _ := req.OwnReport.Facts["missing"].([]string)
	localOnly, _ := req.OwnReport.Facts["local_only"].([]string)

	if len(missing) > 0 || len(localOnly) > 0 {
		input, _ := providers.OperationInput(map[string]interface{}{
			"missing":    missing,
			"local_only": localOnly,
		})

		operations = append(operations, model.Operation{
			ID:          "env.manual_update",
			Provider:    p.Name(),
			Kind:        "manual_update",
			Description: fmt.Sprintf("Manually update %s with %d missing variable(s)", 
				req.OwnReport.Facts["target_file"], len(missing)),
			Disposition: model.Manual,
			Input:       input,
		})
	}

	return operations, nil
}

// Validate valida precondições (sempre passa para env)
func (p *Provider) Validate(ctx context.Context, op model.Operation) error {
	return nil
}

// Apply não é suportado para env na 0.1
func (p *Provider) Apply(ctx context.Context, op model.Operation) (model.ApplyResult, error) {
	return model.ApplyResult{
		OperationID: op.ID,
		Status:      "SKIPPED",
		Summary:     "Environment sync is manual-only in version 0.1",
	}, nil
}

// parseEnvFile lê um arquivo .env e extrai nomes de variáveis
func (p *Provider) parseEnvFile(path string) (map[string]bool, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	vars := make(map[string]bool)
	var warnings []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Ignora linhas vazias e comentários
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove prefixo 'export ' se presente
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
		}

		// Extrai nome da variável (antes do '=')
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			warnings = append(warnings, fmt.Sprintf("Line %d: invalid format (no '=')", lineNum))
			continue
		}

		name := strings.TrimSpace(line[:eqIdx])
		
		// Valida nome da variável
		if !isValidVarName(name) {
			warnings = append(warnings, fmt.Sprintf("Line %d: invalid variable name '%s'", lineNum, maskSecret(name)))
			continue
		}

		// Detecta duplicatas
		if vars[name] {
			warnings = append(warnings, fmt.Sprintf("Line %d: duplicate variable '%s'", lineNum, maskSecret(name)))
		}

		vars[name] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, warnings, err
	}

	return vars, warnings, nil
}

// isValidVarName verifica se o nome é válido
func isValidVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// maskSecret ofusca nomes que parecem ser secrets
func maskSecret(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "key") || strings.Contains(lower, "secret") || 
	   strings.Contains(lower, "password") || strings.Contains(lower, "token") {
		return name[:2] + "***"
	}
	return name
}

// toDetails converte lista de strings para Details
func (p *Provider) toDetails(items []string) []model.Detail {
	details := make([]model.Detail, len(items))
	for i, item := range items {
		details[i] = model.Detail{
			Key:   "variable",
			Value: maskSecret(item),
		}
	}
	return details
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
