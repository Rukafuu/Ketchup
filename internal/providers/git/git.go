package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ketchup-ai/ketchup/internal/exec"
	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/providers"
)

// Provider é o provider de Git do Ketchup
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

// Check executa detecção de drift no repositório Git
func (p *Provider) Check(ctx context.Context, req providers.CheckRequest) (model.Report, error) {
	report := model.Report{
		Provider:   p.Name(),
		Health:     model.HealthClean,
		ObservedAt: time.Now(),
		Facts:      make(model.Facts),
	}

	// Verifica se é um repositório Git
	exitCode, _, err := p.runner.Run(ctx, "git", "-C", req.Root, "rev-parse", "--git-dir")
	if err != nil || exitCode != 0 {
		report.Health = model.HealthUnknown
		report.Summary = "not a Git repository"
		return report, nil
	}

	// Obtém branch atual
	branch, err := p.getCurrentBranch(ctx, req.Root)
	if err != nil {
		report.Health = model.HealthUnknown
		report.Summary = "failed to determine current branch"
		report.Findings = append(report.Findings, model.Finding{
			Code:     "GIT_BRANCH_ERROR",
			Severity: model.SeverityError,
			Summary:  err.Error(),
		})
		return report, nil
	}

	report.Facts["branch"] = branch

	// Verifica se está em detached HEAD
	if branch == "HEAD" {
		report.Health = model.HealthDrifted
		report.Summary = "HEAD is detached"
		report.Findings = append(report.Findings, model.Finding{
			Code:     "GIT_DETACHED_HEAD",
			Severity: model.SeverityWarning,
			Summary:  "Repository is in detached HEAD state",
			Details: []model.Detail{
				{Key: "state", Value: "detached"},
			},
		})
		return report, nil
	}

	// Obtém upstream
	upstream, err := p.getUpstream(ctx, req.Root, branch)
	if err != nil || upstream == "" {
		report.Health = model.HealthDrifted
		report.Summary = "no upstream configured"
		report.Findings = append(report.Findings, model.Finding{
			Code:     "GIT_NO_UPSTREAM",
			Severity: model.SeverityWarning,
			Summary:  "Branch has no upstream configured",
			Details: []model.Detail{
				{Key: "branch", Value: branch},
			},
		})
		return report, nil
	}

	report.Facts["upstream"] = upstream

	// Calcula ahead/behind
	ahead, behind, err := p.getAheadBehind(ctx, req.Root, branch, upstream)
	if err != nil {
		report.Health = model.HealthUnknown
		report.Summary = "failed to calculate ahead/behind"
		return report, err
	}

	report.Facts["ahead"] = ahead
	report.Facts["behind"] = behind

	// Verifica worktree status
	hasUncommitted, untrackedFiles, err := p.getWorktreeStatus(ctx, req.Root)
	if err != nil {
		report.Health = model.HealthUnknown
		report.Summary = "failed to check worktree status"
		return report, err
	}

	report.Facts["has_uncommitted"] = hasUncommitted
	report.Facts["untracked_count"] = len(untrackedFiles)

	// Determina saúde e findings
	var findings []model.Finding

	if hasUncommitted {
		findings = append(findings, model.Finding{
			Code:     "GIT_UNCOMMITTED_CHANGES",
			Severity: model.SeverityWarning,
			Summary:  "Working tree has uncommitted changes",
		})
	}

	if len(untrackedFiles) > 0 {
		findings = append(findings, model.Finding{
			Code:     "GIT_UNTRACKED_FILES",
			Severity: model.SeverityInfo,
			Summary:  fmt.Sprintf("Working tree has %d untracked file(s)", len(untrackedFiles)),
		})
	}

	if ahead > 0 && behind > 0 {
		findings = append(findings, model.Finding{
			Code:     "GIT_DIVERGED",
			Severity: model.SeverityError,
			Summary:  "Branch has diverged from upstream",
			Details: []model.Detail{
				{Key: "ahead", Value: strconv.Itoa(ahead)},
				{Key: "behind", Value: strconv.Itoa(behind)},
			},
		})
		report.Health = model.HealthDrifted
	} else if ahead > 0 {
		findings = append(findings, model.Finding{
			Code:     "GIT_AHEAD_ONLY",
			Severity: model.SeverityInfo,
			Summary:  "Branch is ahead of upstream",
			Details: []model.Detail{
				{Key: "ahead", Value: strconv.Itoa(ahead)},
			},
		})
	} else if behind > 0 {
		findings = append(findings, model.Finding{
			Code:     "GIT_BEHIND_ONLY",
			Severity: model.SeverityWarning,
			Summary:  "Branch is behind upstream",
			Details: []model.Detail{
				{Key: "behind", Value: strconv.Itoa(behind)},
			},
		})
		report.Health = model.HealthDrifted
	}

	if len(findings) == 0 {
		report.Summary = "Git workspace is clean"
	} else {
		report.Findings = findings
		if report.Health == model.HealthClean {
			report.Health = model.HealthDrifted
			report.Summary = "Git workspace has drift"
		}
	}

	// Revision fingerprint
	revision, _ := p.getHeadSHA(ctx, req.Root)
	report.Revision = revision

	return report, nil
}

// Plan gera operações de sincronização Git
func (p *Provider) Plan(ctx context.Context, req providers.PlanRequest) ([]model.Operation, error) {
	var operations []model.Operation

	behind, _ := req.OwnReport.Facts["behind"].(int)
	ahead, _ := req.OwnReport.Facts["ahead"].(int)
	hasUncommitted, _ := req.OwnReport.Facts["has_uncommitted"].(bool)
	untrackedCount, _ := req.OwnReport.Facts["untracked_count"].(int)
	upstream, _ := req.OwnReport.Facts["upstream"].(string)

	// Worktree suja bloqueia operação
	if hasUncommitted || untrackedCount > 0 {
		operations = append(operations, model.Operation{
			ID:          "git.blocked_dirty",
			Provider:    p.Name(),
			Kind:        "blocked",
			Description: "Cannot fast-forward: working tree is not clean",
			Disposition: model.Blocked,
		})
		return operations, nil
	}

	// Sem upstream
	if upstream == "" {
		operations = append(operations, model.Operation{
			ID:          "git.no_upstream",
			Provider:    p.Name(),
			Kind:        "manual",
			Description: "No upstream configured for current branch",
			Disposition: model.Manual,
		})
		return operations, nil
	}

	// Behind only - pode fazer fast-forward
	if behind > 0 && ahead == 0 {
		input, _ := providers.OperationInput(map[string]string{
			"upstream": upstream,
		})

		operations = append(operations, model.Operation{
			ID:          "git.fast_forward",
			Provider:    p.Name(),
			Kind:        "fast_forward",
			Description: fmt.Sprintf("Fast-forward current branch to %s (%d commits behind)", upstream, behind),
			Disposition: model.Safe,
			Preconditions: []model.Precondition{
				{
					ID:          "worktree_clean",
					Description: "Working tree must be clean",
					Check:       "git.status",
					Expected:    "clean",
				},
				{
					ID:          "upstream_unchanged",
					Description: "Upstream must remain the same",
					Check:       "git.upstream",
					Expected:    upstream,
				},
			},
			Input: input,
		})
	}

	return operations, nil
}

// Validate valida precondições de uma operação Git
func (p *Provider) Validate(ctx context.Context, op model.Operation) error {
	switch op.Kind {
	case "fast_forward":
		return nil
	default:
		return fmt.Errorf("unknown operation kind: %s", op.Kind)
	}
}

// Apply executa uma operação Git confirmada
func (p *Provider) Apply(ctx context.Context, op model.Operation) (model.ApplyResult, error) {
	result := model.ApplyResult{
		OperationID: op.ID,
		Status:      "SKIPPED",
	}

	switch op.Kind {
	case "fast_forward":
		var input map[string]string
		if len(op.Input) > 0 {
			// Em produção usaria json.Unmarshal
			// Simplificado para MVP
		}
		_ = input

		// Executa git merge --ff-only
		exitCode, output, err := p.runner.Run(ctx, "git", "merge", "--ff-only")
		if err != nil || exitCode != 0 {
			result.Status = "FAILED"
			result.Summary = fmt.Sprintf("Fast-forward failed: %s", string(output))
			return result, nil
		}

		result.Status = "APPLIED"
		result.Summary = "Successfully fast-forwarded branch"

	default:
		result.Summary = fmt.Sprintf("Operation kind '%s' not applicable for apply", op.Kind)
	}

	return result, nil
}

// Helpers privados

func (p *Provider) getCurrentBranch(ctx context.Context, root string) (string, error) {
	exitCode, output, err := p.runner.Run(ctx, "git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || exitCode != 0 {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (p *Provider) getUpstream(ctx context.Context, root, branch string) (string, error) {
	exitCode, output, err := p.runner.Run(ctx, "git", "-C", root, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil || exitCode != 0 {
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func (p *Provider) getAheadBehind(ctx context.Context, root, branch, upstream string) (int, int, error) {
	exitCode, output, err := p.runner.Run(ctx, "git", "-C", root, "rev-list", "--left-right", "--count", branch+"..."+upstream)
	if err != nil || exitCode != 0 {
		return 0, 0, err
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %s", string(output))
	}

	ahead, _ := strconv.Atoi(parts[0])
	behind, _ := strconv.Atoi(parts[1])

	return ahead, behind, nil
}

var untrackedRegex = regexp.MustCompile(`^\?\? `)

func (p *Provider) getWorktreeStatus(ctx context.Context, root string) (bool, []string, error) {
	exitCode, output, err := p.runner.Run(ctx, "git", "-C", root, "status", "--porcelain")
	if err != nil || exitCode != 0 {
		return false, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var untracked []string
	hasUncommitted := false

	for _, line := range lines {
		if line == "" {
			continue
		}
		if untrackedRegex.MatchString(line) {
			untracked = append(untracked, strings.TrimPrefix(line, "?? "))
		} else {
			hasUncommitted = true
		}
	}

	return hasUncommitted, untracked, nil
}

func (p *Provider) getHeadSHA(ctx context.Context, root string) (string, error) {
	exitCode, output, err := p.runner.Run(ctx, "git", "-C", root, "rev-parse", "HEAD")
	if err != nil || exitCode != 0 {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
