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

	// IgnoredChanges são as mudanças ignoradas (apenas se ShowIgnored for true)
	IgnoredChanges []relevance.RelevantChange `json:"ignored_changes,omitempty"`

	// ShowIgnored indica se deve incluir detalhes dos eventos ignorados
	ShowIgnored bool `json:"show_ignored"`

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
	return g.GenerateWithIgnored(changes, timeAway, false)
}

// GenerateWithIgnored cria um relatório de catch-up com opção de incluir eventos ignorados
func (g *Generator) GenerateWithIgnored(changes []relevance.RelevantChange, timeAway time.Duration, showIgnored bool) *CatchUpReport {
	var relevant []relevance.RelevantChange
	var ignored []relevance.RelevantChange

	for _, change := range changes {
		if !change.Ignored {
			relevant = append(relevant, change)
		} else {
			ignored = append(ignored, change)
		}
	}

	// Ordena por severidade e score (já vem ordenado da engine)
	// Limita a 10 mudanças relevantes para não sobrecarregar
	if len(relevant) > 10 {
		relevant = relevant[:10]
	}

	report := &CatchUpReport{
		TimeAway:        formatDuration(timeAway),
		TotalEvents:     len(changes),
		RelevantChanges: relevant,
		IgnoredCount:    len(ignored),
		ShowIgnored:     showIgnored,
		GeneratedAt:     time.Now(),
	}

	// Inclui detalhes dos ignorados apenas se solicitado
	if showIgnored {
		report.IgnoredChanges = ignored
	}

	return report
}

// RenderText renderiza o relatório em formato texto legível
func (r *CatchUpReport) RenderText() string {
	return r.RenderTextWithIgnored(false)
}

// RenderTextWithIgnored renderiza o relatório com opção de mostrar eventos ignorados
func (r *CatchUpReport) RenderTextWithIgnored(showIgnoredDetails bool) string {
	var sb strings.Builder

	sb.WriteString("Ketchup Catch-up\n\n")
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

	// Mostra resumo dos eventos ignorados
	sb.WriteString(fmt.Sprintf("%d other events were ignored as irrelevant.\n", r.IgnoredCount))

	// Se solicitado, mostra detalhes dos eventos ignorados e seus motivos
	if showIgnoredDetails || r.ShowIgnored {
		if len(r.IgnoredChanges) > 0 {
			sb.WriteString("\n--- Ignored Events Details ---\n\n")
			sb.WriteString(fmt.Sprintf("These %d events were filtered out because their relevance score was below the threshold (<20):\n\n", len(r.IgnoredChanges)))
			
			for i, change := range r.IgnoredChanges {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, change.Event.Title))
				sb.WriteString(fmt.Sprintf("   Score: %d/100 | Severity: %s\n", change.Signal.Score, change.Signal.Severity))
				if len(change.Signal.Reasons) > 0 {
					sb.WriteString("   Partial reasons considered:\n")
					for _, reason := range change.Signal.Reasons {
						sb.WriteString(fmt.Sprintf("     - %s\n", reason))
					}
				} else {
					sb.WriteString("   Reason: No significant overlap with your current work context\n")
				}
				sb.WriteString("\n")
			}
		}
	}

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
