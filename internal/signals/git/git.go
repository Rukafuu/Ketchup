package git

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ketchup-ai/ketchup/internal/exec"
	"github.com/ketchup-ai/ketchup/internal/signals"
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
			Description: signals.SummarizeChangedFiles(commit.FilesChanged),
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

	tokens := strings.Split(string(output), "\x00")
	i := 0

	for i < len(tokens) {
		for i < len(tokens) && strings.TrimSpace(tokens[i]) == "" {
			i++
		}
		if i >= len(tokens) {
			break
		}
		if i+5 >= len(tokens) {
			break
		}

		hash := strings.TrimSpace(tokens[i])
		if !looksLikeCommitHash(hash) {
			i++
			continue
		}

		author := strings.TrimSpace(tokens[i+1])
		dateStr := strings.TrimSpace(tokens[i+2])
		title := strings.TrimSpace(tokens[i+3])
		body := strings.TrimSpace(tokens[i+4])
		refs := strings.TrimSpace(tokens[i+5])
		i += 6

		var files []string
		for i < len(tokens) && strings.TrimSpace(tokens[i]) != "" {
			token := strings.TrimSpace(tokens[i])
			if looksLikeCommitHash(token) {
				break
			}
			files = append(files, token)
			i++
		}

		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			date = time.Now()
		}

		isMerge := strings.Contains(strings.ToLower(title), "merge") || strings.Contains(refs, "tag: ")

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

func looksLikeCommitHash(value string) bool {
	if len(value) < 7 {
		return false
	}
	for _, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'a' && r <= 'f' {
			continue
		}
		if r >= 'A' && r <= 'F' {
			continue
		}
		return false
	}
	return unicode.IsDigit(rune(value[0])) || (value[0] >= 'a' && value[0] <= 'f') || (value[0] >= 'A' && value[0] <= 'F')
}
