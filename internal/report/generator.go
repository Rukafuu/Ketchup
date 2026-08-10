package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/fastforward/ff/internal/relevance"
)

// CatchUpReport é o relatório final de catch-up
type CatchUpReport struct {
	// TimeAway é o tempo desde a última sessão
	TimeAway string `json:"time_away"`

	// TotalEvents é o total de eventos capturados
	TotalEvents int `json:"total_events"`

	// RelevantChanges são as mudanças relevantes (não ignoradas)
	RelevantChanges []relevance.RelevantChange `json:"relevant_changes"`

	// IgnoredCount é quantos eventos foram filtrados por baixa relevância
	IgnoredCount int `json:"ignored_count"`

	// GeneratedAt é quando o relatório foi gerado
	GeneratedAt time.Time `json:"generated_at"`
}

// Generator gera relatórios de catch-up
type Generator struct{}

// NewGenerator cria um novo generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate cria um relatório de catch-up
func (g *Generator) Generate(changes []relevance.RelevantChange, timeAway time.Duration) *CatchUpReport {
	var relevant []relevance.RelevantChange
	ignored := 0

	for _, change := range changes {
		if !change.Ignored {
			relevant = append(relevant, change)
		} else {
			ignored++
		}
	}

	// Ordena por severidade e score (já vem ordenado da engine)
	// Limita a 10 mudanças relevantes para não sobrecarregar
	if len(relevant) > 10 {
		relevant = relevant[:10]
	}

	return &CatchUpReport{
		TimeAway:        formatDuration(timeAway),
		TotalEvents:     len(changes),
		RelevantChanges: relevant,
		IgnoredCount:    ignored,
		GeneratedAt:     time.Now(),
	}
}

// RenderText renderiza o relatório em formato texto legível
func (r *CatchUpReport) RenderText() string {
	var sb strings.Builder

	sb.WriteString("🍅 Ketchup Catch-up\n\n")
	sb.WriteString(fmt.Sprintf("You were away for %s.\n\n", r.TimeAway))

	if len(r.RelevantChanges) == 0 {
		sb.WriteString("No significant changes detected since your last session.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("%d changes matter to your current work:\n\n", len(r.RelevantChanges)))

		// Agrupa por severidade
		bySeverity := make(map[string][]relevance.RelevantChange)
		for _, change := range r.RelevantChanges {
			sev := change.Signal.Severity
			bySeverity[sev] = append(bySeverity[sev], change)
		}

		// Renderiza na ordem de severidade
		for _, severity := range []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"} {
			changes := bySeverity[severity]
			if len(changes) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("%s\n", severity))
			for _, change := range changes {
				sb.WriteString(fmt.Sprintf("%s\n", change.Event.Title))
				for _, reason := range change.Signal.Reasons {
					sb.WriteString(fmt.Sprintf("  → %s\n", reason))
				}
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%d other events were ignored as irrelevant.\n", r.IgnoredCount))

	return sb.String()
}

// formatDuration formata duration de forma legível
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
