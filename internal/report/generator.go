package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/relevance"
	"github.com/ketchup-ai/ketchup/internal/signals"
)

// GenerateOptions controla o conteúdo do relatório de catch-up
type GenerateOptions struct {
	Show        string // relevant | all
	Explain     bool
	MaxRelevant int
}

// CatchUpReport é o relatório final de catch-up
type CatchUpReport struct {
	TimeAway        string                       `json:"time_away"`
	TotalEvents     int                          `json:"total_events"`
	RelevantChanges []relevance.RelevantChange   `json:"relevant_changes"`
	IgnoredCount    int                          `json:"ignored_count"`
	IgnoredChanges  []relevance.RelevantChange   `json:"ignored_changes,omitempty"`
	AllChanges      []relevance.RelevantChange   `json:"all_changes,omitempty"`
	ShowMode        string                       `json:"show_mode"`
	Explain         bool                         `json:"explain"`
	GeneratedAt     time.Time                    `json:"generated_at"`
}

// Generator gera relatórios de catch-up
type Generator struct{}

// NewGenerator cria um novo generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate cria um relatório de catch-up com opções explícitas
func (g *Generator) Generate(changes []relevance.RelevantChange, timeAway time.Duration, opts GenerateOptions) *CatchUpReport {
	if opts.Show == "" {
		opts.Show = model.CatchUpShowRelevant
	}
	if opts.MaxRelevant == 0 {
		opts.MaxRelevant = model.DefaultCatchUpConfig().MaxRelevant
	}

	var relevant []relevance.RelevantChange
	var ignored []relevance.RelevantChange

	for _, change := range changes {
		if change.Ignored {
			ignored = append(ignored, change)
		} else {
			relevant = append(relevant, change)
		}
	}

	if opts.MaxRelevant > 0 && len(relevant) > opts.MaxRelevant {
		relevant = relevant[:opts.MaxRelevant]
	}

	report := &CatchUpReport{
		TimeAway:        formatDuration(timeAway),
		TotalEvents:     len(changes),
		RelevantChanges: relevant,
		IgnoredCount:    len(ignored),
		ShowMode:        opts.Show,
		Explain:         opts.Explain,
		GeneratedAt:     time.Now(),
	}

	if opts.Show == model.CatchUpShowAll {
		report.IgnoredChanges = ignored
		report.AllChanges = append(append([]relevance.RelevantChange{}, relevant...), ignored...)
	}

	return report
}

// RenderText renderiza o relatório em formato texto legível
func (r *CatchUpReport) RenderText() string {
	return r.render(false)
}

// RenderTextWithExplain renderiza com detalhes de score e motivos
func (r *CatchUpReport) RenderTextWithExplain() string {
	return r.render(true)
}

func (r *CatchUpReport) render(forceExplain bool) string {
	explain := r.Explain || forceExplain
	var sb strings.Builder

	sb.WriteString("Ketchup Catch-up\n\n")
	sb.WriteString(fmt.Sprintf("You were away for %s.\n\n", r.TimeAway))

	if r.ShowMode == model.CatchUpShowAll {
		r.renderAllChanges(&sb, explain)
		return sb.String()
	}

	r.renderRelevantOnly(&sb, explain)
	return sb.String()
}

func (r *CatchUpReport) renderRelevantOnly(sb *strings.Builder, explain bool) {
	if len(r.RelevantChanges) == 0 {
		sb.WriteString("No significant changes detected since your last session.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("%d changes matter to your current work:\n\n", len(r.RelevantChanges)))
		r.renderChangesBySeverity(sb, r.RelevantChanges, "RELEVANT", explain)
	}

	if r.IgnoredCount > 0 {
		sb.WriteString(fmt.Sprintf("\n%d other events were ignored as irrelevant.\n", r.IgnoredCount))
		sb.WriteString("Tip: set catchup.show: all in .ketchup.yaml or run ketchup catch-up --show all\n")
	}
}

func (r *CatchUpReport) renderAllChanges(sb *strings.Builder, explain bool) {
	total := len(r.AllChanges)
	if total == 0 {
		sb.WriteString("No changes detected since your last session.\n")
		return
	}

	sb.WriteString(fmt.Sprintf(
		"Showing all %d events (%d relevant, %d ignored):\n\n",
		total,
		len(r.RelevantChanges),
		r.IgnoredCount,
	))

	for i, change := range r.AllChanges {
		tag := "RELEVANT"
		if change.Ignored {
			tag = "IGNORED"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s · %s · %d/100] %s\n",
			i+1,
			tag,
			change.Signal.Severity,
			change.Signal.Score,
			change.Event.Title,
		))
		r.renderFileSummary(sb, change)
		r.renderChangeReasons(sb, change, explain)
		sb.WriteString("\n")
	}
}

func (r *CatchUpReport) renderChangesBySeverity(sb *strings.Builder, changes []relevance.RelevantChange, tag string, explain bool) {
	bySeverity := make(map[string][]relevance.RelevantChange)
	for _, change := range changes {
		sev := change.Signal.Severity
		bySeverity[sev] = append(bySeverity[sev], change)
	}

	for _, severity := range []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"} {
		group := bySeverity[severity]
		if len(group) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s\n", severity))
		for _, change := range group {
			if explain {
				sb.WriteString(fmt.Sprintf("[%s · %d/100] %s\n", tag, change.Signal.Score, change.Event.Title))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n", change.Event.Title))
			}
			r.renderFileSummary(sb, change)
			r.renderChangeReasons(sb, change, explain)
			sb.WriteString("\n")
		}
	}
}

func (r *CatchUpReport) renderFileSummary(sb *strings.Builder, change relevance.RelevantChange) {
	if summary := signals.SummarizeChangedFiles(change.Event.Files); summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n")
		return
	}
	if change.Event.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", change.Event.Description))
	}
}

func (r *CatchUpReport) renderChangeReasons(sb *strings.Builder, change relevance.RelevantChange, explain bool) {
	if len(change.Signal.Reasons) > 0 {
		for _, reason := range change.Signal.Reasons {
			sb.WriteString(fmt.Sprintf("  → %s\n", reason))
		}
	} else if change.Ignored {
		sb.WriteString("  → No significant overlap with your current work context\n")
	}

	if explain && len(change.Contributions) > 0 {
		sb.WriteString("  Scoring breakdown:\n")
		for _, contribution := range change.Contributions {
			sb.WriteString(fmt.Sprintf("    • %s (%+d): %s\n", contribution.Rule, contribution.Delta, contribution.Reason))
		}
	}
}

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
