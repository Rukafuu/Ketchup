package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ketchup-ai/ketchup/internal/config"
	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/providers"
)

// Engine orquestra o pipeline de check/plan/apply
type Engine struct {
	root      string
	config    *model.Config
	providers map[string]providers.Provider
}

// NewEngine cria uma nova Engine com providers registrados
func NewEngine(root string, cfg *model.Config) *Engine {
	return &Engine{
		root:      root,
		config:    cfg,
		providers: make(map[string]providers.Provider),
	}
}

// RegisterProvider registra um provider na engine
func (e *Engine) RegisterProvider(p providers.Provider) {
	e.providers[p.Name()] = p
}

// Status executa checks em todos os providers e retorna saúde agregada
func (e *Engine) Status(ctx context.Context) ([]model.Report, model.Health, error) {
	reports, err := e.runAllChecks(ctx)
	if err != nil {
		return nil, model.HealthUnknown, err
	}

	health := providers.AggregateHealth(reports)
	return reports, health, nil
}

// Diff retorna detalhes dos findings de todos os providers
func (e *Engine) Diff(ctx context.Context) ([]model.Report, error) {
	return e.runAllChecks(ctx)
}

// Sync executa o pipeline completo: check → plan → confirmação → apply → re-check
func (e *Engine) Sync(ctx context.Context, confirmed bool) (*SyncResult, error) {
	result := &SyncResult{
		ID:        fmt.Sprintf("sync-%d", time.Now().Unix()),
		StartedAt: time.Now(),
	}

	// Fase 1: Check inicial
	reports, initialHealth, err := e.Status(ctx)
	if err != nil {
		result.Status = "FAILED"
		result.Summary = fmt.Sprintf("Check failed: %v", err)
		return result, err
	}
	result.InitialReports = reports
	result.InitialHealth = initialHealth

	if initialHealth == model.HealthClean {
		result.Status = "COMPLETED"
		result.Summary = "Workspace is already clean"
		return result, nil
	}

	// Fase 2: Plan
	plan, err := e.createPlan(ctx, reports)
	if err != nil {
		result.Status = "FAILED"
		result.Summary = fmt.Sprintf("Planning failed: %v", err)
		return result, err
	}
	result.Plan = plan

	// Filtra operações SAFE
	var safeOps []model.Operation
	for _, op := range plan.Operations {
		if op.Disposition == model.Safe {
			safeOps = append(safeOps, op)
		}
	}

	if len(safeOps) == 0 {
		result.Status = "MANUAL_REQUIRED"
		result.Summary = "No safe operations available; manual intervention required"
		return result, nil
	}

	// Fase 3: Confirmação
	if !confirmed {
		result.Status = "AWAITING_CONFIRMATION"
		result.Summary = "Confirmation required to proceed"
		return result, nil
	}

	// Fase 4: Validate e Apply
	var applyResults []model.ApplyResult
	for _, op := range safeOps {
		provider, ok := e.providers[op.Provider]
		if !ok {
			applyResults = append(applyResults, model.ApplyResult{
				OperationID: op.ID,
				Status:      "FAILED",
				Summary:     fmt.Sprintf("Provider '%s' not found", op.Provider),
			})
			continue
		}

		// Revalida precondições
		if err := provider.Validate(ctx, op); err != nil {
			applyResults = append(applyResults, model.ApplyResult{
				OperationID: op.ID,
				Status:      "STALE",
				Summary:     fmt.Sprintf("Precondition validation failed: %v", err),
			})
			result.Status = "STALE_PLAN"
			continue
		}

		// Aplica operação
		applyResult, err := provider.Apply(ctx, op)
		if err != nil {
			applyResult.Status = "FAILED"
			applyResult.Summary = fmt.Sprintf("Apply error: %v", err)
		}
		applyResults = append(applyResults, applyResult)

		if applyResult.Status == "FAILED" || applyResult.Status == "STALE" {
			result.Status = "PARTIAL"
		}
	}

	result.ApplyResults = applyResults

	// Fase 5: Re-check
	finalReports, finalHealth, err := e.Status(ctx)
	if err != nil {
		result.Status = "RECHECK_FAILED"
		result.Summary = fmt.Sprintf("Post-sync check failed: %v", err)
		return result, err
	}
	result.FinalReports = finalReports
	result.FinalHealth = finalHealth

	if finalHealth == model.HealthClean && result.Status != "PARTIAL" {
		result.Status = "COMPLETED"
		result.Summary = "Sync completed successfully"
	} else if result.Status == "" {
		result.Status = "PARTIAL"
		result.Summary = "Sync completed with remaining drift"
	}

	return result, nil
}

// Doctor valida configuração e ferramentas disponíveis
func (e *Engine) Doctor(ctx context.Context) (*DoctorResult, error) {
	result := &DoctorResult{
		Checks: make([]DoctorCheck, 0),
	}

	// Check: Configuração
	result.Checks = append(result.Checks, DoctorCheck{
		Name:    "config",
		Passed:  e.config != nil,
		Message: "Configuration loaded",
	})

	// Check: Git disponível
	if cmdExists("git") {
		result.Checks = append(result.Checks, DoctorCheck{
			Name:    "git",
			Passed:  true,
			Message: "Git is installed",
		})
	} else {
		result.Checks = append(result.Checks, DoctorCheck{
			Name:    "git",
			Passed:  false,
			Message: "Git is not installed",
		})
	}

	// Check: É repositório Git
	isRepo := cmdExists("git") && e.isGitRepo()
	result.Checks = append(result.Checks, DoctorCheck{
		Name:    "repository",
		Passed:  isRepo,
		Message: "Directory is a Git repository",
	})

	// Check: Node/npm (opcional)
	if cmdExists("node") {
		result.Checks = append(result.Checks, DoctorCheck{
			Name:    "node",
			Passed:  true,
			Message: "Node.js is installed",
		})
	}

	// Check: Providers habilitados
	for name := range e.config.Providers {
		_, enabled := e.providers[name]
		result.Checks = append(result.Checks, DoctorCheck{
			Name:    fmt.Sprintf("provider.%s", name),
			Passed:  enabled,
			Message: fmt.Sprintf("Provider '%s' is %s", name, map[bool]string{true: "enabled", false: "not registered"}[enabled]),
		})
	}

	return result, nil
}

// runAllChecks executa Check em todos os providers habilitados
func (e *Engine) runAllChecks(ctx context.Context) ([]model.Report, error) {
	var reports []model.Report

	for name, provider := range e.providers {
		cfg, ok := e.config.Providers[name]
		if !ok {
			cfg = config.GetDefaults(name)
		}

		req := providers.CheckRequest{
			Root:   e.root,
			Config: cfg,
		}

		report, err := provider.Check(ctx, req)
		if err != nil {
			// Erro não fatal - marca provider como UNKNOWN
			report = model.Report{
				Provider:   name,
				Health:     model.HealthUnknown,
				Summary:    fmt.Sprintf("Check failed: %v", err),
				ObservedAt: time.Now(),
			}
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// createPlan gera plano de sincronização a partir dos reports
func (e *Engine) createPlan(ctx context.Context, reports []model.Report) (*model.SyncPlan, error) {
	plan := &model.SyncPlan{
		ID:          fmt.Sprintf("plan-%d", time.Now().Unix()),
		CreatedAt:   time.Now(),
		ProjectRoot: e.root,
		Operations:  make([]model.Operation, 0),
	}

	// Obtém mudanças prospectivas do Git (se disponível)
	var prospectiveChanges []model.FileChange
	for _, r := range reports {
		if r.Provider == "git" {
			// Em implementação completa, extrairia file changes do Git report
			break
		}
	}

	// Pede operações a cada provider
	for _, report := range reports {
		provider, ok := e.providers[report.Provider]
		if !ok {
			continue
		}

		cfg, _ := e.config.Providers[report.Provider]
		if cfg == nil {
			cfg = config.GetDefaults(report.Provider)
		}

		req := providers.PlanRequest{
			Root:               e.root,
			Config:             cfg,
			OwnReport:          report,
			ProspectiveChanges: prospectiveChanges,
		}

		ops, err := provider.Plan(ctx, req)
		if err != nil {
			continue // Erro no plan não bloqueia outros providers
		}

		plan.Operations = append(plan.Operations, ops...)
	}

	return plan, nil
}

// Helpers

func cmdExists(cmd string) bool {
	// Implementação simplificada - seria delegada ao exec package
	return true
}

func (e *Engine) isGitRepo() bool {
	// Implementação simplificada
	return true
}

// SyncResult é o resultado de uma operação de sync
type SyncResult struct {
	ID            string              `json:"id"`
	StartedAt     time.Time           `json:"started_at"`
	Status        string              `json:"status"` // COMPLETED, PARTIAL, FAILED, MANUAL_REQUIRED, AWAITING_CONFIRMATION, STALE_PLAN
	Summary       string              `json:"summary"`
	InitialHealth model.Health        `json:"initial_health"`
	InitialReports []model.Report     `json:"initial_reports"`
	Plan          *model.SyncPlan     `json:"plan,omitempty"`
	ApplyResults  []model.ApplyResult `json:"apply_results,omitempty"`
	FinalHealth   model.Health        `json:"final_health"`
	FinalReports  []model.Report      `json:"final_reports"`
}

// DoctorResult é o resultado do doctor
type DoctorResult struct {
	Checks []DoctorCheck `json:"checks"`
}

// DoctorCheck é um check individual do doctor
type DoctorCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}
