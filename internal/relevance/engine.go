package relevance

import (
	"path/filepath"
	"strings"

	"github.com/fastforward/ff/internal/signals"
)

// RelevanceSignal representa um sinal de relevância computado
type RelevanceSignal struct {
	// Score é a pontuação de relevância (0-100)
	Score int `json:"score"`

	// Reasons explica por que este evento é relevante
	Reasons []string `json:"reasons"`

	// Severity indica a severidade (CRITICAL, HIGH, MEDIUM, LOW)
	Severity string `json:"severity"`
}

// RelevantChange é um evento com sua relevância computada
type RelevantChange struct {
	Event    signals.NormalizedEvent `json:"event"`
	Signal   RelevanceSignal         `json:"signal"`
	Ignored  bool                    `json:"ignored"` // true se foi filtrado por baixa relevância
}

// Engine computa relevância de eventos baseado no contexto
type Engine struct {
	// CurrentBranch é o branch atual do desenvolvedor
	CurrentBranch string

	// CurrentFiles são arquivos atualmente abertos/editados
	CurrentFiles []string

	// RecentFiles são arquivos recentemente modificados localmente
	RecentFiles []string

	// Developer é o nome do desenvolvedor atual
	Developer string
}

// NewEngine cria uma nova engine de relevância
func NewEngine() *Engine {
	return &Engine{}
}

// ComputeRelevance computa a relevância de um conjunto de eventos
func (e *Engine) ComputeRelevance(events []signals.NormalizedEvent) []RelevantChange {
	var changes []RelevantChange

	for _, event := range events {
		signal := e.computeEventRelevance(event)
		change := RelevantChange{
			Event:   event,
			Signal:  signal,
			Ignored: signal.Score < 20, // Threshold configurável
		}
		changes = append(changes, change)
	}

	return changes
}

// computeEventRelevance computa relevância para um único evento
func (e *Engine) computeEventRelevance(event signals.NormalizedEvent) RelevanceSignal {
	signal := RelevanceSignal{
		Score:   0,
		Reasons: []string{},
	}

	// 1. Overlap com arquivos atuais/recentes (alto peso)
	for _, currentFile := range e.CurrentFiles {
		if e.fileInEvent(currentFile, event) {
			signal.Score += 40
			signal.Reasons = append(signal.Reasons,
				"This commit changed a file you currently have open")
		}
	}

	for _, recentFile := range e.RecentFiles {
		if e.fileInEvent(recentFile, event) {
			signal.Score += 30
			signal.Reasons = append(signal.Reasons,
				"This commit changed a file you recently edited")
		}
	}

	// 2. Mesmo diretório/módulo (peso médio)
	for _, currentFile := range e.CurrentFiles {
		dir := filepath.Dir(currentFile)
		if e.dirInEvent(dir, event) {
			signal.Score += 20
			signal.Reasons = append(signal.Reasons,
				"This commit changed files in the same directory as your work")
		}
	}

	// 3. Arquivos de dependência/configuração (alto peso)
	for _, file := range event.Files {
		if isCriticalFile(file) {
			signal.Score += 25
			signal.Reasons = append(signal.Reasons,
				"This commit changed a critical configuration or dependency file")
		}
	}

	// 4. Arquivos de migração (alto peso)
	for _, file := range event.Files {
		if isMigrationFile(file) {
			signal.Score += 30
			signal.Reasons = append(signal.Reasons,
				"This commit changed a database migration file")
		}
	}

	// 5. Merge commits (peso adicional)
	if event.Type == "merge" {
		signal.Score += 15
		signal.Reasons = append(signal.Reasons,
			"This is a merge commit")
	}

	// 6. Autor relevante (se desenvolvedor conhecido)
	if e.Developer != "" && event.Actor == e.Developer {
		// Commits do próprio desenvolvedor têm menor prioridade
		signal.Score -= 10
		signal.Reasons = append(signal.Reasons,
			"This is your own commit")
	}

	// 7. Recência (decai com o tempo)
	// Já considerado no fetch, mas podemos adicionar bônus para muito recente

	// Normaliza score para 0-100
	if signal.Score > 100 {
		signal.Score = 100
	}
	if signal.Score < 0 {
		signal.Score = 0
	}

	// Determina severidade
	signal.Severity = e.computeSeverity(signal.Score)

	return signal
}

// fileInEvent verifica se um arquivo está na lista de arquivos do evento
func (e *Engine) fileInEvent(file string, event signals.NormalizedEvent) bool {
	for _, f := range event.Files {
		if f == file {
			return true
		}
	}
	return false
}

// dirInEvent verifica se algum arquivo do evento está em um diretório
func (e *Engine) dirInEvent(dir string, event signals.NormalizedEvent) bool {
	for _, f := range event.Files {
		if strings.HasPrefix(f, dir+"/") || strings.HasPrefix(f, dir+"\\") {
			return true
		}
	}
	return false
}

// isCriticalFile verifica se é um arquivo crítico
func isCriticalFile(file string) bool {
	criticalPatterns := []string{
		"go.mod", "go.sum",
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		"requirements.txt", "Pipfile.lock", "poetry.lock",
		".env.example", ".env.template",
		"Dockerfile", "docker-compose.yml",
		"Makefile", "Taskfile.yml",
		".github/", ".gitlab-ci.yml",
		"tsconfig.json", "webpack.config.js",
	}

	for _, pattern := range criticalPatterns {
		if file == pattern || strings.Contains(file, "/"+pattern) {
			return true
		}
	}

	return false
}

// isMigrationFile verifica se é um arquivo de migração
func isMigrationFile(file string) bool {
	migrationPatterns := []string{
		"/migrations/", "/migration/",
		"_migration.", "_migrate.",
		".up.sql", ".down.sql",
	}

	for _, pattern := range migrationPatterns {
		if strings.Contains(file, pattern) {
			return true
		}
	}

	return false
}

// computeSeverity determina severidade baseada no score
func (e *Engine) computeSeverity(score int) string {
	switch {
	case score >= 70:
		return "CRITICAL"
	case score >= 50:
		return "HIGH"
	case score >= 30:
		return "MEDIUM"
	case score >= 20:
		return "LOW"
	default:
		return "IGNORED"
	}
}
