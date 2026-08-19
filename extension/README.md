# Ketchup Fast-Forward — VS Code Extension

Visual Studio Code extension for [Ketchup](https://github.com/ketchup-ai/ketchup): detect and sync drift between your local workspace and the expected project state.

**Install:** [VS Marketplace — Reskyume.ketchup-fast-forward](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward)

## Requirements

- VS Code 1.85.0 or later
- [Ketchup CLI](https://github.com/ketchup-ai/ketchup) installed and available on `PATH` (or configured via `ketchup.cliPath`)
- A workspace with `.ketchup.yml` or `.ketchup.yaml`

## Install

### From VSIX

```bash
cd extension
npm install
npm run compile
npm run package
```

Then in VS Code: **Extensions → ... → Install from VSIX** and select `ketchup-fast-forward-0.2.0.vsix`.

### Development

1. `npm install && npm run compile`
2. Open the `extension/` folder in VS Code
3. Press `F5` to launch the Extension Development Host
4. Open a project that contains `.ketchup.yml`

## Features

- **Sidebar panel** — live status for Git, Dependencies, and Environment providers
- **Status bar** — clean / drift / error indicator with issue count
- **Quick-fix actions** — contextual actions on findings (catch-up, sync)
- **Auto-check** — runs status when the workspace opens
- **Output channel** — detailed logs under "Ketchup"
- **CLI update check** — optional startup check for new core releases

## Commands

| Command | CLI equivalent |
|---------|----------------|
| Ketchup: Check Status | `ketchup status` |
| Ketchup: Show Diff | `ketchup diff` |
| Ketchup: Sync Workspace | `ketchup sync` |
| Ketchup: Run Doctor | `ketchup doctor` |
| Ketchup: Catch Up | `ketchup catch-up` (uses settings / project config) |
| Ketchup: Catch Up (Show All Changes) | `ketchup catch-up --show all` |
| Ketchup: Check for Updates | `ketchup update --check` |
| Ketchup: Refresh Status | refreshes sidebar |

## Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `ketchup.cliPath` | `ketchup` | Path to the CLI executable |
| `ketchup.autoCheckOnOpen` | `true` | Auto-run status on workspace open |
| `ketchup.showNotifications` | `true` | Show drift/completion notifications |
| `ketchup.showStatusBar` | `true` | Show status bar indicator |
| `ketchup.autoUpdate` | `true` | Check for CLI updates on startup |
| `ketchup.catchUpShow` | `relevant` | Catch-up mode: `relevant` or `all` |
| `ketchup.catchUpExplain` | `false` | Include relevance scores in catch-up output |

## How it works

The extension runs the Ketchup CLI as a subprocess:

1. Tries JSON output first (`status --json`)
2. Falls back to text parsing if JSON is unavailable
3. Respects CLI exit codes (0=clean, 1=drift, 2=config error, 3=check failed)
4. Passes `KETCHUP_CURRENT_FILE` for context-aware catch-up

## License

[MIT](../LICENSE)
