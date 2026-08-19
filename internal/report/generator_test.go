package report

import (
	"strings"
	"testing"
	"time"

	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/relevance"
	"github.com/ketchup-ai/ketchup/internal/signals"
)

func sampleChange(title string, score int, ignored bool) relevance.RelevantChange {
	decision := "RELEVANT"
	if ignored {
		decision = "IGNORED"
	}
	return relevance.RelevantChange{
		Event: signals.NormalizedEvent{Title: title},
		Signal: relevance.RelevanceSignal{
			Score:    score,
			Severity: "MEDIUM",
			Reasons:  []string{"test reason"},
			Decision: decision,
		},
		Ignored: ignored,
	}
}

func TestGenerateRelevantMode(t *testing.T) {
	gen := NewGenerator()
	changes := []relevance.RelevantChange{
		sampleChange("Important fix", 80, false),
		sampleChange("README typo", 5, true),
	}

	report := gen.Generate(changes, 2*time.Hour, GenerateOptions{
		Show:        model.CatchUpShowRelevant,
		MaxRelevant: 10,
	})

	if len(report.RelevantChanges) != 1 {
		t.Fatalf("expected 1 relevant change, got %d", len(report.RelevantChanges))
	}
	if report.IgnoredCount != 1 {
		t.Fatalf("expected ignored count 1, got %d", report.IgnoredCount)
	}
	if len(report.AllChanges) != 0 {
		t.Fatal("relevant mode should not populate all_changes")
	}

	text := report.RenderText()
	if !strings.Contains(text, "1 changes matter") {
		t.Fatalf("unexpected text: %s", text)
	}
	if !strings.Contains(text, "1 other events were ignored") {
		t.Fatalf("expected ignored summary, got: %s", text)
	}
}

func TestGenerateAllMode(t *testing.T) {
	gen := NewGenerator()
	changes := []relevance.RelevantChange{
		sampleChange("Important fix", 80, false),
		sampleChange("README typo", 5, true),
	}

	report := gen.Generate(changes, 30*time.Minute, GenerateOptions{
		Show:        model.CatchUpShowAll,
		Explain:     true,
		MaxRelevant: 10,
	})

	if len(report.AllChanges) != 2 {
		t.Fatalf("expected 2 all changes, got %d", len(report.AllChanges))
	}

	text := report.RenderTextWithExplain()
	if !strings.Contains(text, "[RELEVANT") || !strings.Contains(text, "[IGNORED") {
		t.Fatalf("expected tagged changes, got: %s", text)
	}
	if !strings.Contains(text, "Showing all 2 events") {
		t.Fatalf("expected all-events header, got: %s", text)
	}
}
