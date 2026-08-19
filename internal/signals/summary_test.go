package signals

import (
	"strings"
	"testing"
)

func TestSummarizeChangedFiles(t *testing.T) {
	summary := SummarizeChangedFiles([]string{
		"extension/package.json",
		"extension/src/extension.ts",
		"cmd/ketchup/main.go",
		"README.md",
	})

	if !strings.Contains(summary, "4 file(s)") {
		t.Fatalf("expected file count, got: %s", summary)
	}
	if !strings.Contains(summary, "extension (package.json") {
		t.Fatalf("expected extension group, got: %s", summary)
	}
	if !strings.Contains(summary, "CLI (main.go") {
		t.Fatalf("expected CLI group, got: %s", summary)
	}
}

func TestSummarizeChangedFilesEmpty(t *testing.T) {
	if SummarizeChangedFiles(nil) != "" {
		t.Fatal("expected empty summary")
	}
}
