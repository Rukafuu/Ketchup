package main

import (
"context"
"encoding/json"
"fmt"
"log"
"os"
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
"github.com/ketchup-ai/ketchup/internal/updater"
)

// Version é injetado via build flags: -ldflags "-X main.Version=0.8.1"
var Version = "0.1.0"

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
runVersion(args)
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
case "update":
os.Exit(runUpdate(ctx, root, args))
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
  update      Check for and install updates

Catch-up options:
  --show relevant|all   What to display (overrides catchup.show in config)
  --show-ignored        Alias for --show all
  --explain             Include relevance scores and scoring breakdown
  --json                Output as JSON

Options:
  --help, -h     Show this help message
  --version, -v  Show version

Examples:
  ketchup status
  ketchup catch-up --explain
  ketchup catch-up --show all
  ketchup catch-up --show-ignored
  ketchup catch-up --json
  ketchup version --json
  ketchup update --check
  ketchup update --channel beta
`)
}

func runVersion(args []string) {
jsonOutput := containsArg(args, "--json")

if jsonOutput {
platform := updater.DetectPlatform()
output := map[string]interface{}{
"version":  Version,
"channel":  "stable",
"platform": platform.Platform,
"os":       platform.OS,
"arch":     platform.Arch,
}
jsonBytes, _ := json.MarshalIndent(output, "", "  ")
fmt.Println(string(jsonBytes))
} else {
fmt.Printf("Ketchup v%s\n", Version)
}
}

func runUpdate(ctx context.Context, root string, args []string) int {
checkOnly := containsArg(args, "--check")
force := containsArg(args, "--force")

// Determina canal
channel := "stable"
for i, arg := range args {
if arg == "--channel" && i+1 < len(args) {
channel = args[i+1]
}
}

// Configura logger para output visível
logger := log.New(os.Stdout, "", 0)

// Cria updater
up := updater.NewUpdater(updater.UpdaterConfig{
BaseURL:        "https://releases.ketchup.dev",
Channel:        channel,
CurrentVersion: Version,
AutoUpdate:     !checkOnly,
Logger:         logger,
})

// Step 1: Check for update
versionInfo, err := up.CheckForUpdate()
if err != nil {
fmt.Fprintf(os.Stderr, "Warning: failed to check for updates: %v\n", err)
fmt.Println("Continuing with current version...")
return 0 // Não falha se servidor estiver indisponível
}

if versionInfo == nil {
fmt.Println("Unable to check for updates at this time.")
return 0
}

if !versionInfo.UpdateAvailable && !force {
fmt.Printf("Already on latest version (%s) for %s channel.\n", Version, channel)
return 0
}

if checkOnly {
if versionInfo.UpdateAvailable {
fmt.Printf("Update available: %s -> %s (%s channel)\n", 
Version, versionInfo.Latest, channel)
} else {
fmt.Println("No updates available.")
}
return 0
}

// Step 2-4: Download and install
fmt.Printf("Updating from %s to %s...\n", Version, versionInfo.Latest)

updated, restartRequired, err := up.DoUpdate()
if err != nil {
fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
fmt.Println("Your installation is still functional.")
return 1
}

if !updated {
fmt.Println("No update was necessary.")
return 0
}

fmt.Println("Update completed successfully!")

if restartRequired {
fmt.Println("\nPlease restart Ketchup/Cursor to apply the update.")
}

return 0
}

func runStatus(ctx context.Context, root string, args []string) int {
	_, eng, err := setupEngine(root)
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
	_, eng, err := setupEngine(root)
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
	_, eng, err := setupEngine(root)
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
	_, eng, err := setupEngine(root)
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
	cfg, err := config.NewLoader(root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	opts := resolveCatchUpOptions(cfg.CatchUp, args)

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

	var allEvents []signals.NormalizedEvent

	gitSignalProvider := gitsignals.NewProvider(nil)
	events, err := gitSignalProvider.FetchEvents(ctx, root, since)
	if err == nil {
		allEvents = append(allEvents, events...)
	}

	currentFile := getCurrentFile()
	recentFiles := getRecentFiles(root)

	relEngine := relevance.NewEngine()
	relEngine.CurrentFiles = []string{}
	if currentFile != "" {
		relEngine.CurrentFiles = []string{currentFile}
	}
	relEngine.RecentFiles = recentFiles

	changes := relEngine.ComputeRelevanceWithContributions(allEvents)

	gen := report.NewGenerator()
	timeAway := time.Since(since.Timestamp)
	if timeAway < 0 {
		timeAway = 0
	}

	rep := gen.Generate(changes, timeAway, report.GenerateOptions{
		Show:        opts.Show,
		Explain:     opts.Explain,
		MaxRelevant: opts.MaxRelevant,
	})

	if opts.JSON {
		output, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Println(rep.RenderText())
	}

	headCommit := getCurrentHeadCommit(ctx, root)
	branch := getCurrentBranch(ctx, root)
	if headCommit != "" || branch != "" {
		store.UpdateOrCreate(root, headCommit, branch)
	}

	return 0
}

type catchUpOptions struct {
	Show        string
	Explain     bool
	MaxRelevant int
	JSON        bool
}

func resolveCatchUpOptions(cfg model.CatchUpConfig, args []string) catchUpOptions {
	opts := catchUpOptions{
		Show:        cfg.Show,
		Explain:     cfg.Explain,
		MaxRelevant: cfg.MaxRelevant,
		JSON:        containsArg(args, "--json"),
	}

	if containsArg(args, "--show-ignored") || containsArg(args, "--all") {
		opts.Show = model.CatchUpShowAll
	}
	if show := getArgValue(args, "--show"); show != "" {
		opts.Show = show
	}
	if containsArg(args, "--explain") {
		opts.Explain = true
	}

	switch opts.Show {
	case model.CatchUpShowRelevant, model.CatchUpShowAll:
	default:
		opts.Show = model.CatchUpShowRelevant
	}

	return opts
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

func getArgValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
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
