package model

import "time"

// Health representa o estado de saúde de um provider
type Health string

const (
	HealthClean   Health = "CLEAN"
	HealthDrifted Health = "DRIFTED"
	HealthUnknown Health = "UNKNOWN"
)

// Severity indica a gravidade de um finding
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityError    Severity = "ERROR"
	SeverityCritical Severity = "CRITICAL"
)

// Detail é um fato nomeado e não secreto
type Detail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Finding representa uma anomalia detectada
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Summary  string   `json:"summary"`
	Details  []Detail `json:"details,omitempty"`
}

// Facts é um container para dados tipados do provider
type Facts map[string]any

// Report é o resultado de um check de provider
type Report struct {
	Provider   string    `json:"provider"`
	Health     Health    `json:"health"`
	Summary    string    `json:"summary"`
	Findings   []Finding `json:"findings,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
	Revision   string    `json:"revision,omitempty"`
	Facts      Facts     `json:"facts,omitempty"`
}

// Disposition indica se uma operação pode ser aplicada
type Disposition string

const (
	Safe    Disposition = "SAFE"
	Manual  Disposition = "MANUAL"
	Blocked Disposition = "BLOCKED"
)

// Precondition é uma validação pré-aplicação
type Precondition struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Check       string `json:"check"`
	Expected    string `json:"expected"`
}

// Operation é uma unidade atômica de mudança
type Operation struct {
	ID            string        `json:"id"`
	Provider      string        `json:"provider"`
	Kind          string        `json:"kind"`
	Description   string        `json:"description"`
	Disposition   Disposition   `json:"disposition"`
	DependsOn     []string      `json:"depends_on,omitempty"`
	Preconditions []Precondition `json:"preconditions,omitempty"`
	Input         []byte        `json:"input,omitempty"`
}

// SyncPlan é o plano completo de sincronização
type SyncPlan struct {
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"created_at"`
	ProjectRoot string      `json:"project_root"`
	Operations  []Operation `json:"operations"`
}

// ApplyResult é o resultado da aplicação de uma operação
type ApplyResult struct {
	OperationID string `json:"operation_id"`
	Status      string `json:"status"` // APPLIED, SKIPPED, FAILED, STALE
	Summary     string `json:"summary"`
}

// FileChange representa uma mudança de arquivo detectada pelo Git
type FileChange struct {
	Path       string `json:"path"`
	Status     string `json:"status"` // A, M, D, R, ??
	HashBefore string `json:"hash_before,omitempty"`
	HashAfter  string `json:"hash_after,omitempty"`
}

// ProviderConfig é a configuração específica de um provider
type ProviderConfig map[string]any

// Config é a configuração completa do FastForward
type Config struct {
	Version   string                    `json:"version"`
	Providers map[string]ProviderConfig `json:"providers"`
}
