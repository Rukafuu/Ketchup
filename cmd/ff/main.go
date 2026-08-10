package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fastforward/ff/internal/config"
	"github.com/fastforward/ff/internal/engine"
	"github.com/fastforward/ff/internal/exec"
	"github.com/fastforward/ff/internal/model"
	"github.com/fastforward/ff/internal/providers/dependencies"
	"github.com/fastforward/ff/internal/providers/env"
	"github.com/fastforward/ff/internal/providers/git"
)

const (
	ExitSuccess        = 0
	ExitDrift          = 1
	ExitConfigError    = 2
	ExitCheckFailed    = 3
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(ExitConfigError)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine working directory: %v\n", err)
		os.Exit(ExitConfigError)
	}

	// Carrega configuração
	cfgLoader := config.NewLoader(root)
	cfg, err := cfgLoader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid configuration: %v\n", err)
		os.Exit(ExitConfigError)
	}

	// Cria engine e registra providers
	runner := &exec.DefaultCommandRunner{Dir: root}
	engine := engine.NewEngine(root, cfg)
	engine.RegisterProvider(git.NewProvider(runner))
	engine.RegisterProvider(env.NewProvider())
	engine.RegisterProvider(dependencies.NewProvider(runner))

	ctx := context.Background()
	command := os.Args[1]

	var exitCode int
	switch command {
	case "status":
		exitCode = cmdStatus(ctx, engine)
	case "diff":
		exitCode = cmdDiff(ctx, engine)
	case "sync":
		exitCode = cmdSync(ctx, engine)
	case "doctor":
		exitCode = cmdDoctor(ctx, engine)
	case "help", "--help", "-h":
		printUsage()
		exitCode = ExitSuccess
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		printUsage()
		exitCode = ExitConfigError
	}

	os.Exit(exitCode)
}

func printUsage() {
	fmt.Println(`FastForward - Detect and sync drift between local workspace and expected state

Usage: ff <command>

Commands:
  status   Show summary and aggregated health (read-only)
  diff     Show detailed drift findings (read-only)
  sync     Check + plan + confirm + apply changes
  doctor   Validate configuration and tools
  help     Show this help message

Exit Codes:
  0  Success (clean workspace or sync completed)
  1  Drift detected or manual action pending
  2  Invalid configuration or usage
  3  Check failed or stale plan`)
}

func cmdStatus(ctx context.Context, eng *engine.Engine) int {
	reports, health, err := eng.Status(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitCheckFailed
	}

	fmt.Printf("Health: %s\n\n", health)

	for _, r := range reports {
		statusIcon := "✓"
		if r.Health == model.HealthDrifted {
			statusIcon = "!"
		} else if r.Health == model.HealthUnknown {
			statusIcon = "?"
		}

		fmt.Printf("[%s] %s: %s\n", statusIcon, r.Provider, r.Summary)
		
		if len(r.Findings) > 0 {
			for _, f := range r.Findings {
				fmt.Printf("    • %s\n", f.Summary)
			}
		}
	}

	if health == model.HealthClean {
		return ExitSuccess
	}
	return ExitDrift
}

func cmdDiff(ctx context.Context, eng *engine.Engine) int {
	reports, err := eng.Diff(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitCheckFailed
	}

	for _, r := range reports {
		fmt.Printf("Provider: %s\n", r.Provider)
		fmt.Printf("Health: %s\n", r.Health)
		fmt.Printf("Summary: %s\n\n", r.Summary)

		if len(r.Findings) > 0 {
			fmt.Println("Findings:")
			for _, f := range r.Findings {
				fmt.Printf("  [%s] %s\n", f.Code, f.Severity)
				fmt.Printf("    %s\n", f.Summary)
				if len(f.Details) > 0 {
					for _, d := range f.Details {
						fmt.Printf("    • %s: %s\n", d.Key, d.Value)
					}
				}
				fmt.Println()
			}
		}
		fmt.Println(strings.Repeat("-", 60))
	}

	health := aggregateHealth(reports)
	if health == model.HealthClean {
		return ExitSuccess
	}
	return ExitDrift
}

func cmdSync(ctx context.Context, eng *engine.Engine) int {
	// Fase inicial de check
	result, err := eng.Sync(ctx, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during sync planning: %v\n", err)
		return ExitCheckFailed
	}

	if result.Status == "COMPLETED" {
		fmt.Println("Workspace is already clean. Nothing to sync.")
		return ExitSuccess
	}

	// Mostra plano
	fmt.Println("Proposed changes:")
	if result.Plan != nil && len(result.Plan.Operations) > 0 {
		for _, op := range result.Plan.Operations {
			icon := "•"
			if op.Disposition == model.Safe {
				icon = "+"
			} else if op.Disposition == model.Blocked {
				icon = "!"
			} else if op.Disposition == model.Manual {
				icon = "?"
			}
			fmt.Printf("  [%s] %s: %s\n", icon, op.Provider, op.Description)
		}
	} else {
		fmt.Println("  No safe operations available.")
	}
	fmt.Println()

	// Solicita confirmação
	if result.Status == "MANUAL_REQUIRED" {
		fmt.Println("Manual intervention required. Please resolve the issues above.")
		return ExitDrift
	}

	fmt.Print("Proceed with sync? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Sync cancelled.")
		return ExitDrift
	}

	// Executa sync confirmado
	result, err = eng.Sync(ctx, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during sync: %v\n", err)
		return ExitCheckFailed
	}

	fmt.Printf("\nSync %s: %s\n", result.Status, result.Summary)

	if len(result.ApplyResults) > 0 {
		fmt.Println("\nOperations:")
		for _, ar := range result.ApplyResults {
			icon := "•"
			if ar.Status == "APPLIED" {
				icon = "✓"
			} else if ar.Status == "FAILED" || ar.Status == "STALE" {
				icon = "!"
			}
			fmt.Printf("  [%s] %s: %s\n", icon, ar.OperationID, ar.Summary)
		}
	}

	if result.FinalHealth == model.HealthClean {
		return ExitSuccess
	}
	return ExitDrift
}

func cmdDoctor(ctx context.Context, eng *engine.Engine) int {
	result, err := eng.Doctor(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitCheckFailed
	}

	fmt.Println("FastForward Doctor\n")

	allPassed := true
	for _, check := range result.Checks {
		icon := "✓"
		if !check.Passed {
			icon = "✗"
			allPassed = false
		}
		fmt.Printf("[%s] %s: %s\n", icon, check.Name, check.Message)
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed!")
		return ExitSuccess
	}
	fmt.Println("Some checks failed. Please review the issues above.")
	return ExitDrift
}

func aggregateHealth(reports []model.Report) model.Health {
	hasUnknown := false
	hasDrifted := false

	for _, r := range reports {
		switch r.Health {
		case model.HealthUnknown:
			hasUnknown = true
		case model.HealthDrifted:
			hasDrifted = true
		}
	}

	if hasUnknown {
		return model.HealthUnknown
	}
	if hasDrifted {
		return model.HealthDrifted
	}
	return model.HealthClean
}
