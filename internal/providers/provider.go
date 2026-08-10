package providers

import (
	"context"
	"encoding/json"

	"github.com/fastforward/ff/internal/model"
)

// CheckRequest é a entrada para o Check de um provider
type CheckRequest struct {
	Root   string                 `json:"root"`
	Config model.ProviderConfig   `json:"config"`
}

// PlanRequest é a entrada para o Plan de um provider
type PlanRequest struct {
	Root               string                 `json:"root"`
	Config             model.ProviderConfig   `json:"config"`
	OwnReport          model.Report           `json:"own_report"`
	ProspectiveChanges []model.FileChange     `json:"prospective_changes"`
}

// Provider é a interface que todos os providers devem implementar
type Provider interface {
	// Name retorna o nome identificador do provider
	Name() string

	// Check executa detecção de drift sem mutações
	Check(ctx context.Context, req CheckRequest) (model.Report, error)

	// Plan gera operações baseadas no report e mudanças prospectivas
	Plan(ctx context.Context, req PlanRequest) ([]model.Operation, error)

	// Validate valida precondições de uma operação antes do apply
	Validate(ctx context.Context, op model.Operation) error

	// Apply executa uma operação confirmada
	Apply(ctx context.Context, op model.Operation) (model.ApplyResult, error)
}

// AggregateHealth calcula a saúde agregada de múltiplos reports
// UNKNOWN tem precedência sobre DRIFTED, que tem precedência sobre CLEAN
func AggregateHealth(reports []model.Report) model.Health {
	hasUnknown := false
	hasDrifted := false

	for _, r := range reports {
		switch r.Health {
		case model.HealthUnknown:
			hasUnknown = true
		case model.HealthDrifted:
			hasDrifted = true
		}
	}

	if hasUnknown {
		return model.HealthUnknown
	}
	if hasDrifted {
		return model.HealthDrifted
	}
	return model.HealthClean
}

// OperationInput helper para criar input JSON tipado
func OperationInput(data any) ([]byte, error) {
	return json.Marshal(data)
}
