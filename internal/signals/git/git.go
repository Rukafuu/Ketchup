package git

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fastforward/ff/internal/exec"
	"github.com/fastforward/ff/internal/signals"
)

// Provider é o provider Git para sinais do Ketchup
type Provider struct {
	runner exec.CommandRunner
}

// NewProvider cria um novo provider Git
func NewProvider(runner exec.CommandRunner) *Provider {
	if runner == nil {
		runner = &exec.DefaultCommandRunner{}
	}
	return &Provider{runner: runner}
}

// Name retorna o nome do provider
func (p *Provider) Name() string {
	return "git"
}

// FetchEvents busca eventos Git desde a última sessão
func (p *Provider) FetchEvents(ctx context.Context, root string, since signals.LastSessionInfo) ([]signals.NormalizedEvent, error) {
	var events []signals.NormalizedEvent

	// Determina o range de commits
	var commitRange string
	if since.HeadCommit != "" {
		commitRange = fmt.Sprintf("%s..HEAD", since.HeadCommit)
	} else {
		// Sem sessão anterior: últimos 30 dias
		commitRange = "--since=30 days ago"
	}

	// Busca commits com informações detalhadas
	commits, err := p.getCommits(ctx, root, commitRange)
	if err != nil {
		return nil, err
	}

	for _, commit := range commits {
		eventType := "commit"
		if commit.IsMerge {
			eventType = "merge"
		}

		events = append(events, signals.NormalizedEvent{
			ID:          commit.Hash,
			Source:      "git",
			Type:        eventType,
			Timestamp:   commit.Date,
			Actor:       commit.Author,
			Title:       commit.Title,
			Description: commit.Body,
			Files:       commit.FilesChanged,
			Metadata: map[string]any{
				"hash":     commit.Hash,
				"branch":   commit.Branch,
				"is_merge": commit.IsMerge,
			},
		})
	}

	return events, nil
}

// CommitInfo contém informações sobre um commit
type CommitInfo struct {
	Hash         string
	Author       string
	Date         time.Time
	Title        string
	Body         string
	FilesChanged []string
	Branch       string
	IsMerge      bool
}

// getCommits obtém commits desde um ponto
func (p *Provider) getCommits(ctx context.Context, root, commitRange string) ([]CommitInfo, error) {
	args := []string{"-C", root, "log", "--format=%H%x00%an%x00%aI%x00%s%x00%b%x00%d", "--name-only", "-z"}

	if strings.HasPrefix(commitRange, "--since") {
		args = append(args, commitRange)
	} else {
		args = append(args, commitRange)
	}

	exitCode, output, err := p.runner.Run(ctx, "git", args...)
	if err != nil || exitCode != 0 {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	return p.parseCommits(output)
}

// parseCommits transforma output do git log em CommitInfo
func (p *Provider) parseCommits(output []byte) ([]CommitInfo, error) {
	var commits []CommitInfo

	if len(output) == 0 {
		return commits, nil
	}

	// Output vem como: hash\0author\0date\0title\0body\0refs\0\0files\0\0...
	parts := strings.Split(string(output), "\x00\x00")

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		lines := strings.Split(part, "\x00")
		if len(lines) < 6 {
			continue
		}

		hash := strings.TrimSpace(lines[0])
		author := strings.TrimSpace(lines[1])
		dateStr := strings.TrimSpace(lines[2])
		title := strings.TrimSpace(lines[3])
		body := strings.TrimSpace(lines[4])
		refs := strings.TrimSpace(lines[5])

		// Parse date
		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			date = time.Now()
		}

		// Detect merge
		isMerge := strings.Contains(refs, "tag: ") || strings.Contains(title, "Merge")

		// Parse files changed (vem após o sexto elemento)
		var files []string
		if len(lines) > 6 {
			filesStr := strings.Join(lines[6:], "\x00")
			scanner := bufio.NewScanner(strings.NewReader(filesStr))
			for scanner.Scan() {
				f := strings.TrimSpace(scanner.Text())
				if f != "" {
					files = append(files, f)
				}
			}
		}

		commits = append(commits, CommitInfo{
			Hash:         hash,
			Author:       author,
			Date:         date,
			Title:        title,
			Body:         body,
			FilesChanged: files,
			Branch:       refs,
			IsMerge:      isMerge,
		})
	}

	return commits, nil
}
