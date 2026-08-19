package git

import (
	"strings"
	"testing"
)

func TestParseCommitsWithFiles(t *testing.T) {
	output := "abc1234\x00Alice\x002026-08-19T10:00:00-03:00\x00feat: add cli\x00\x00\x00" +
		"cmd/ketchup/main.go\x00extension/package.json\x00\x00" +
		"def5678\x00Bob\x002026-08-19T11:00:00-03:00\x00docs: update readme\x00\x00\x00" +
		"README.md\x00\x00"

	provider := &Provider{}
	commits, err := provider.parseCommits([]byte(output))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if len(commits[0].FilesChanged) != 2 {
		t.Fatalf("expected 2 files in first commit, got %v", commits[0].FilesChanged)
	}
	if commits[0].FilesChanged[0] != "cmd/ketchup/main.go" {
		t.Fatalf("unexpected files: %v", commits[0].FilesChanged)
	}
	if !strings.Contains(commits[0].Title, "feat: add cli") {
		t.Fatalf("unexpected title: %s", commits[0].Title)
	}
}
