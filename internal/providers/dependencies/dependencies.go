package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ketchup-ai/ketchup/internal/exec"
	"github.com/ketchup-ai/ketchup/internal/fs"
	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/providers"
)

// PackageManager representa um gerenciador de pacotes suportado
type PackageManager string

const (
	NPM  PackageManager = "npm"
	PNPM PackageManager = "pnpm"
	Yarn PackageManager = "yarn"
)

// Provider é o provider de Dependencies do Ketchup
type Provider struct {
	runner exec.CommandRunner
}

// NewProvider cria um novo provider Dependencies
func NewProvider(runner exec.CommandRunner) *Provider {
	if runner == nil {
		runner = &exec.DefaultCommandRunner{}
	}
	return &Provider{runner: runner}
}

// Name retorna o nome do provider
func (p *Provider) Name() string {
	return "dependencies"
}

// LockfileInfo contém informações sobre um lockfile detectado
type LockfileInfo struct {
	Path           string       `json:"path"`
	PackageManager PackageManager `json:"package_manager"`
	Hash           string       `json:"hash"`
}

// Check detecta drift de dependências baseado em lockfiles
func (p *Provider) Check(ctx context.Context, req providers.CheckRequest) (model.Report, error) {
	report := model.Report{
		Provider:   p.Name(),
		Health:     model.HealthClean,
		ObservedAt: time.Now(),
		Facts:      make(model.Facts),
	}

	// Detecta package manager e lockfile
	pm, lockfile, err := p.detectPackageManager(ctx, req.Root)
	if err != nil {
		report.Health = model.HealthUnknown
		report.Summary = fmt.Sprintf("Failed to detect package manager: %v", err)
		return report, nil
	}

	if pm == "" {
		report.Summary = "No supported package manager detected"
		report.Health = model.HealthClean
		return report, nil
	}

	report.Facts["package_manager"] = string(pm)
	report.Facts["lockfile"] = lockfile.Path

	// Calcula hash do lockfile
	hash, err := fs.HashFile(lockfile.Path)
	if err != nil {
		report.Health = model.HealthUnknown
		report.Summary = fmt.Sprintf("Failed to hash lockfile: %v", err)
		return report, nil
	}

	report.Facts["lockfile_hash"] = hash
	report.Revision = hash

	// Verifica se node_modules existe
	nodeModules := filepath.Join(req.Root, "node_modules")
	hasNodeModules := fs.DirExists(nodeModules)
	report.Facts["has_node_modules"] = hasNodeModules

	if !hasNodeModules {
		report.Health = model.HealthDrifted
		report.Summary = "Dependencies not installed (node_modules missing)"
		report.Findings = append(report.Findings, model.Finding{
			Code:     "DEPS_NOT_INSTALLED",
			Severity: model.SeverityWarning,
			Summary:  "node_modules directory is missing",
		})
		return report, nil
	}

	report.Summary = "Dependencies appear to be installed"
	return report, nil
}

// Plan gera operações de instalação baseadas em mudanças prospectivas
func (p *Provider) Plan(ctx context.Context, req providers.PlanRequest) ([]model.Operation, error) {
	var operations []model.Operation

	pm, _ := req.OwnReport.Facts["package_manager"].(string)
	lockfilePath, _ := req.OwnReport.Facts["lockfile"].(string)
	currentHash, _ := req.OwnReport.Facts["lockfile_hash"].(string)
	hasNodeModules, _ := req.OwnReport.Facts["has_node_modules"].(bool)

	_ = currentHash // usado para validação futura

	// Verifica se há mudança no lockfile nas mudanças prospectivas
	var lockfileChanged bool
	var newHash string

	for _, change := range req.ProspectiveChanges {
		if change.Path == lockfilePath || change.Path == "package-lock.json" ||
		   change.Path == "pnpm-lock.yaml" || change.Path == "yarn.lock" {
			lockfileChanged = true
			newHash = change.HashAfter
			break
		}
	}

	// Planeja instalação se:
	// 1. Lockfile mudou nas mudanças prospectivas, OU
	// 2. node_modules não existe
	shouldInstall := (!hasNodeModules && lockfilePath != "") || lockfileChanged

	if shouldInstall && pm != "" {
		input, _ := providers.OperationInput(map[string]interface{}{
			"package_manager": pm,
			"lockfile":        lockfilePath,
			"expected_hash":   newHash,
		})

		operations = append(operations, model.Operation{
			ID:          "deps.install",
			Provider:    p.Name(),
			Kind:        "install",
			Description: fmt.Sprintf("Install dependencies using %s (frozen lockfile)", pm),
			Disposition: model.Safe,
			Preconditions: []model.Precondition{
				{
					ID:          "lockfile_present",
					Description: "Lockfile must exist",
					Check:       "file.exists",
					Expected:    lockfilePath,
				},
				{
					ID:          "lockfile_hash",
					Description: "Lockfile hash must match expected",
					Check:       "file.hash",
					Expected:    newHash,
				},
			},
			Input: input,
		})
	}

	return operations, nil
}

// Validate valida precondições de uma operação de dependências
func (p *Provider) Validate(ctx context.Context, op model.Operation) error {
	switch op.Kind {
	case "install":
		var input map[string]interface{}
		if len(op.Input) > 0 {
			if err := json.Unmarshal(op.Input, &input); err != nil {
				return fmt.Errorf("invalid operation input: %w", err)
			}
		}

		lockfile, _ := input["lockfile"].(string)
		expectedHash, _ := input["expected_hash"].(string)

		if lockfile != "" {
			if !fs.FileExists(lockfile) {
				return fmt.Errorf("lockfile not found: %s", lockfile)
			}

			if expectedHash != "" {
				actualHash, err := fs.HashFile(lockfile)
				if err != nil {
					return fmt.Errorf("failed to hash lockfile: %w", err)
				}
				if actualHash != expectedHash {
					return fmt.Errorf("lockfile hash mismatch: expected %s, got %s", expectedHash, actualHash)
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown operation kind: %s", op.Kind)
	}
}

// Apply executa instalação de dependências
func (p *Provider) Apply(ctx context.Context, op model.Operation) (model.ApplyResult, error) {
	result := model.ApplyResult{
		OperationID: op.ID,
		Status:      "SKIPPED",
	}

	switch op.Kind {
	case "install":
		var input map[string]interface{}
		if len(op.Input) > 0 {
			json.Unmarshal(op.Input, &input)
		}

		pm, _ := input["package_manager"].(string)
		
		var cmd string
		var args []string

		switch PackageManager(pm) {
		case NPM:
			cmd = "npm"
			args = []string{"ci"}
		case PNPM:
			cmd = "pnpm"
			args = []string{"install", "--frozen-lockfile"}
		case Yarn:
			cmd = "yarn"
			args = []string{"install", "--immutable"}
		default:
			result.Status = "FAILED"
			result.Summary = fmt.Sprintf("Unsupported package manager: %s", pm)
			return result, nil
		}

		exitCode, output, err := p.runner.Run(ctx, cmd, args...)
		if err != nil || exitCode != 0 {
			result.Status = "FAILED"
			result.Summary = fmt.Sprintf("Installation failed: %s", string(output))
			return result, nil
		}

		result.Status = "APPLIED"
		result.Summary = fmt.Sprintf("Successfully installed dependencies with %s", pm)

	default:
		result.Summary = fmt.Sprintf("Operation kind '%s' not applicable for apply", op.Kind)
	}

	return result, nil
}

// detectPackageManager detecta o package manager do projeto
func (p *Provider) detectPackageManager(ctx context.Context, root string) (PackageManager, *LockfileInfo, error) {
	packageJSON := filepath.Join(root, "package.json")
	
	// Tenta ler package.json para detectar packageManager field
	if data, err := os.ReadFile(packageJSON); err == nil {
		var pkg struct {
			PackageManager string `json:"packageManager"`
		}
		if json.Unmarshal(data, &pkg) == nil && pkg.PackageManager != "" {
			// Extrai nome do package manager (ex: "npm@9.0.0" -> "npm")
			pmName := strings.Split(pkg.PackageManager, "@")[0]
			switch pmName {
			case "npm":
				return NPM, p.detectLockfile(root, NPM), nil
			case "pnpm":
				return PNPM, p.detectLockfile(root, PNPM), nil
			case "yarn":
				return Yarn, p.detectLockfile(root, Yarn), nil
			}
		}
	}

	// Fallback: detecta pelo lockfile presente
	lockfiles := []struct {
		path string
		pm   PackageManager
	}{
		{"package-lock.json", NPM},
		{"pnpm-lock.yaml", PNPM},
		{"yarn.lock", Yarn},
	}

	var found *LockfileInfo
	var count int

	for _, lf := range lockfiles {
		if info := p.detectLockfile(root, lf.pm); info != nil {
			found = info
			count++
		}
	}

	if count > 1 {
		return "", nil, fmt.Errorf("multiple conflicting lockfiles detected")
	}

	if found != nil {
		switch found.PackageManager {
		case NPM:
			return NPM, found, nil
		case PNPM:
			return PNPM, found, nil
		case Yarn:
			return Yarn, found, nil
		}
	}

	return "", nil, nil
}

// detectLockfile procura por um lockfile específico
func (p *Provider) detectLockfile(root string, pm PackageManager) *LockfileInfo {
	var path string
	switch pm {
	case NPM:
		path = filepath.Join(root, "package-lock.json")
	case PNPM:
		path = filepath.Join(root, "pnpm-lock.yaml")
	case Yarn:
		path = filepath.Join(root, "yarn.lock")
	}

	if !fs.FileExists(path) {
		return nil
	}

	hash, _ := fs.HashFile(path)

	return &LockfileInfo{
		Path:           path,
		PackageManager: pm,
		Hash:           hash,
	}
}
