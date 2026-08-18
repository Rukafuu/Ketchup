package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ketchup-ai/ketchup/internal/config"
	"github.com/ketchup-ai/ketchup/internal/engine"
	"github.com/ketchup-ai/ketchup/internal/model"
	"github.com/ketchup-ai/ketchup/internal/providers/dependencies"
	envprovider "github.com/ketchup-ai/ketchup/internal/providers/env"
	gitprovider "github.com/ketchup-ai/ketchup/internal/providers/git"
	"github.com/ketchup-ai/ketchup/internal/relevance"
	"github.com/ketchup-ai/ketchup/internal/report"
	"github.com/ketchup-ai/ketchup/internal/session"
	"github.com/ketchup-ai/ketchup/internal/signals"
	gitsignals "github.com/ketchup-ai/ketchup/internal/signals/git"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	ctx := context.Background()
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	switch cmd {
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	case "version", "--version", "-v":
		fmt.Printf("ketchup version %s\n", version)
		os.Exit(0)
	case "status":
		os.Exit(runStatus(ctx, root, args))
	case "diff":
		os.Exit(runDiff(ctx, root, args))
	case "sync":
		os.Exit(runSync(ctx, root, args))
	case "doctor":
		os.Exit(runDoctor(ctx, root, args))
	case "catch-up":
		os.Exit(runCatchUp(ctx, root, args))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println(`Ketchup — never run out of sync

Usage:
  ketchup <command> [options]

Commands:
  status      Check workspace health (read-only)
  diff        Show drift details (read-only)
  sync        Plan and apply synchronization
  doctor      Validate configuration and tools
  catch-up    Explain what happened since last session

Options:
  --help, -h     Show this help message
  --version, -v  Show version

Examples:
  ketchup status
  ketchup catch-up --explain
  ketchup catch-up --show-ignored
  ketchup catch-up --json
`)
}

func runStatus(ctx context.Context, root string, args []string) int {
	cfg, eng, err := setupEngine(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	reports, health, err := eng.Status(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Check for JSON output
	jsonOutput := containsArg(args, "--json")
	if jsonOutput {
		output, _ := json.MarshalIndent(reports, "", "  ")
		fmt.Println(string(output))
	} else {
		for _, r := range reports {
			icon := "✓"
			if r.Health == model.HealthDrifted {
				icon = "!"
			} else if r.Health == model.HealthUnknown {
				icon = "?"
			}
			fmt.Printf("[%s] %s: %s\n", icon, r.Provider, r.Summary)
			for _, f := range r.Findings {
				fmt.Printf("  • %s\n", f.Summary)
			}
		}
	}

	if health == model.HealthDrifted {
		return 1
	}
	return 0
}

func runDiff(ctx context.Context, root string, args []string) int {
	cfg, eng, err := setupEngine(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	reports, err := eng.Diff(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	jsonOutput := containsArg(args, "--json")
	if jsonOutput {
		output, _ := json.MarshalIndent(reports, "", "  ")
		fmt.Println(string(output))
	} else {
		for _, r := range reports {
			fmt.Printf("Provider: %s\n", r.Provider)
			fmt.Printf("Health: %s\n", r.Health)
			fmt.Printf("Summary: %s\n", r.Summary)
			for _, f := range r.Findings {
				fmt.Printf("  [%s] %s\n", f.Severity, f.Summary)
				for _, d := range f.Details {
					fmt.Printf("    %s: %s\n", d.Key, d.Value)
				}
			}
			fmt.Println()
		}
	}

	return 0
}

func runSync(ctx context.Context, root string, args []string) int {
	cfg, eng, err := setupEngine(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	confirmed := containsArg(args, "--yes") || containsArg(args, "-y")
	result, err := eng.Sync(ctx, confirmed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 3
	}

	jsonOutput := containsArg(args, "--json")
	if jsonOutput {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Sync %s: %s\n", result.Status, result.Summary)
		if result.Plan != nil {
			fmt.Printf("\nPlanned operations (%d):\n", len(result.Plan.Operations))
			for _, op := range result.Plan.Operations {
				fmt.Printf("  [%s] %s: %s\n", op.Disposition, op.Provider, op.Description)
			}
		}
	}

	switch result.Status {
	case "COMPLETED":
		return 0
	case "MANUAL_REQUIRED":
		return 1
	default:
		return 1
	}
}

func runDoctor(ctx context.Context, root string, args []string) int {
	cfg, eng, err := setupEngine(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	result, err := eng.Doctor(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	jsonOutput := containsArg(args, "--json")
	if jsonOutput {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		allPassed := true
		for _, check := range result.Checks {
			icon := "✓"
			if !check.Passed {
				icon = "✗"
				allPassed = false
			}
			fmt.Printf("[%s] %s: %s\n", icon, check.Name, check.Message)
		}
		if !allPassed {
			return 1
		}
	}

	return 0
}

func runCatchUp(ctx context.Context, root string, args []string) int {
	// Load session
	store := session.NewStore(root)
	lastSession, err := store.Load()

	var since signals.LastSessionInfo
	if err == nil && lastSession != nil {
		since = signals.LastSessionInfo{
			Timestamp:  lastSession.LastActivity,
			HeadCommit: lastSession.HeadCommit,
			Branch:     lastSession.Branch,
		}
	}

	// Fetch events from all signal providers
	var allEvents []signals.NormalizedEvent

	// Git signal provider
	gitSignalProvider := gitsignals.NewProvider(nil)
	events, err := gitSignalProvider.FetchEvents(ctx, root, since)
	if err == nil {
		allEvents = append(allEvents, events...)
	}

	// Get workspace context for relevance
	currentFile := getCurrentFile()
	recentFiles := getRecentFiles(root)

	// Compute relevance
	relEngine := relevance.NewEngine()
	relEngine.CurrentFiles = []string{}
	if currentFile != "" {
		relEngine.CurrentFiles = []string{currentFile}
	}
	relEngine.RecentFiles = recentFiles

	changes := relEngine.ComputeRelevanceWithContributions(allEvents)

	// Generate report
	gen := report.NewGenerator()
	timeAway := time.Since(since.Timestamp)
	if timeAway < 0 {
		timeAway = 0
	}

	showIgnored := containsArg(args, "--show-ignored")
	rep := gen.GenerateWithContributions(changes, timeAway, showIgnored)

	// Output
	jsonOutput := containsArg(args, "--json")
	showExplain := containsArg(args, "--explain")

	if jsonOutput {
		output, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Println(rep.RenderTextWithExplanation(showExplain))
	}

	// Update session
	headCommit := getCurrentHeadCommit(ctx, root)
	branch := getCurrentBranch(ctx, root)
	if headCommit != "" || branch != "" {
		store.UpdateOrCreate(root, headCommit, branch)
	}

	return 0
}

func setupEngine(root string) (*model.Config, *engine.Engine, error) {
	loader := config.NewLoader(root)
	cfg, err := loader.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	eng := engine.NewEngine(root, cfg)

	// Register providers
	if cfg.Providers["git"] != nil || true { // Always register git
		eng.RegisterProvider(gitprovider.NewProvider(nil))
	}
	if cfg.Providers["dependencies"] != nil || true {
		eng.RegisterProvider(dependencies.NewProvider(nil))
	}
	if cfg.Providers["env"] != nil || true {
		eng.RegisterProvider(envprovider.NewProvider())
	}

	return cfg, eng, nil
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func getCurrentFile() string {
	// This would be provided by VS Code extension via env var
	return os.Getenv("KETCHUP_CURRENT_FILE")
}

func getRecentFiles(root string) []string {
	// This would be provided by VS Code extension via env var or file
	// For now, return empty
	return []string{}
}

func getCurrentHeadCommit(ctx context.Context, root string) string {
	// Simplified - would use exec runner in production
	return ""
}

func getCurrentBranch(ctx context.Context, root string) string {
	// Simplified - would use exec runner in production
	return ""
}
