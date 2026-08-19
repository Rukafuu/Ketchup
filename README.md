# Ketchup

[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.19+-blue)](go.mod)
[![VS Code](https://img.shields.io/badge/vscode-1.85+-blue)](extension/package.json)

**Ketchup** is a CLI tool and VS Code extension that detects and synchronizes drift between your local development workspace and the expected project state.

> Keep your workspace always in sync — safely.

## Features

- **Git drift detection** — branch ahead/behind, fast-forward only sync
- **Dependencies** — detects lockfile drift (npm, pnpm, yarn)
- **Environment** — compares `.env` against `.env.example` without exposing secrets
- **Catch-up** — summarizes what changed since your last session
- **VS Code extension** — sidebar panel, status bar, quick-fix actions, auto-check

## Quick Start

### CLI

```bash
# Build
go build -o ketchup ./cmd/ketchup    # Linux/macOS
go build -o ketchup.exe ./cmd/ketchup  # Windows

# Verify
./ketchup help
./ketchup doctor
./ketchup status
```

Add the binary to your `PATH`, or configure the extension with `"ketchup.cliPath"`.

### VS Code Extension

**From VSIX (local install):**

```bash
cd extension
npm install
npm run compile
npm run package
# Install the generated .vsix via: Extensions → ... → Install from VSIX
```

**Development mode:** open the repo in VS Code, go to `extension/`, press `F5`.

## Commands

| Command | Description | Exit codes |
|---------|-------------|------------|
| `ketchup status` | Workspace health summary (read-only) | 0=clean, 1=drift |
| `ketchup diff` | Detailed drift report (read-only) | 0=clean, 1=drift |
| `ketchup sync` | Plan + confirm + apply changes | 0=success, 1=manual |
| `ketchup doctor` | Validate config and required tools | 0=ok, 1=failures |
| `ketchup catch-up` | Summarize changes since last session | 0=ok |
| `ketchup update` | Check or install CLI updates | 0=ok |

All commands support `--json` for machine-readable output.

### Catch-up display modes

Control what `ketchup catch-up` shows via `.ketchup.yaml` or CLI flags (flags override config):

| Mode | Config | CLI | What you see |
|------|--------|-----|--------------|
| Relevant only | `catchup.show: relevant` | `--show relevant` (default) | Changes Ketchup scored as useful to your current work |
| All changes | `catchup.show: all` | `--show all` or `--show-ignored` | Every event, tagged `[RELEVANT]` or `[IGNORED]` with scores |
| Explain | `catchup.explain: true` | `--explain` | Adds relevance scores and scoring breakdown per change |

```yaml
catchup:
  show: all           # relevant | all
  explain: true       # show why each change was scored
  max_relevant: 10    # limit relevant items (0 = unlimited)
```

```bash
ketchup catch-up                         # uses config defaults
ketchup catch-up --show all              # list every change
ketchup catch-up --show all --explain    # all changes with scoring details
ketchup catch-up --json                  # machine-readable report
```

### Example

```bash
ketchup status
ketchup diff
ketchup sync
ketchup catch-up --explain
ketchup doctor
```

## VS Code Extension

The extension wraps the CLI and adds:

- **Activity Bar panel** — provider status with expandable findings
- **Status bar** — clean / drift / error indicator with count
- **Quick actions** — catch-up for Git drift, sync for env/deps issues
- **Auto-check** — runs `status` when the workspace opens
- **Update check** — optional check for new CLI releases on startup

### Extension commands

| Command | Description |
|---------|-------------|
| `Ketchup: Check Status` | Run `ketchup status` |
| `Ketchup: Show Diff` | Run `ketchup diff` |
| `Ketchup: Sync Workspace` | Run `ketchup sync` |
| `Ketchup: Run Doctor` | Run `ketchup doctor` |
| `Ketchup: Catch Up` | Run `ketchup catch-up` (respects config/settings) |
| `Ketchup: Catch Up (Show All Changes)` | Run catch-up listing every event |
| `Ketchup: Check for Updates` | Check CLI updates |
| `Ketchup: Refresh Status` | Refresh sidebar panel |

### Settings

```json
{
  "ketchup.cliPath": "ketchup",
  "ketchup.autoCheckOnOpen": true,
  "ketchup.showNotifications": true,
  "ketchup.showStatusBar": true,
  "ketchup.autoUpdate": true,
  "ketchup.updateChannel": "stable",
  "ketchup.catchUpShow": "relevant",
  "ketchup.catchUpExplain": false
}
```

## Configuration

Create `.ketchup.yml` or `.ketchup.yaml` in the project root (see [`.ketchup.example.yaml`](.ketchup.example.yaml)):

```yaml
version: 1

catchup:
  show: relevant    # relevant | all
  explain: false
  max_relevant: 10

providers:
  git:
    enabled: true
    strategy: fast-forward-only
  dependencies:
    enabled: true
    auto_install: true
  env:
    enabled: true
    source: .env.example
    target: .env
```

The extension activates automatically when a workspace contains `.ketchup.yml` or `.ketchup.yaml`.

## Architecture

```
config → check → snapshot → plan → confirmation → validate → apply → re-check
```

### Providers

1. **Git** — drift detection, fast-forward only
2. **Dependencies** — lockfile-aware install on sync
3. **Environment** — variable name comparison, no secret values

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Drift or manual action required |
| `2` | Invalid configuration |
| `3` | Check failed or stale plan |

## Project structure

```
.
├── cmd/ketchup/          # CLI entry point
├── internal/             # Core engine, providers, updater
├── extension/            # VS Code extension (TypeScript)
├── scripts/release.sh    # Multi-platform release builds
├── docs/                 # Design docs
└── .ketchup.example.yaml # Example configuration
```

## Development

```bash
# CLI
go build -o ketchup ./cmd/ketchup
go test ./...

# Extension
cd extension
npm install
npm run compile
npm run watch    # dev mode
npm run package  # generate .vsix
```

See [extension/DEVELOPMENT.md](extension/DEVELOPMENT.md) for extension-specific notes.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for completed work and upcoming milestones (tests, CI/CD, VS Marketplace publication).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes
4. Open a pull request

## License

[MIT](LICENSE)

---

## Português

**Ketchup** é uma ferramenta CLI e extensão VS Code que detecta e sincroniza diferenças (*drift*) entre o ambiente local e o estado esperado do projeto.

### Início rápido

```bash
# Compilar CLI
go build -o ketchup.exe ./cmd/ketchup   # Windows
go build -o ketchup ./cmd/ketchup       # Linux/macOS

# Empacotar extensão
cd extension
npm install
npm run compile
npm run package
```

Instale o `.vsix` gerado em **Extensions → ... → Install from VSIX**.

### Comandos principais

| Comando | Descrição |
|---------|-----------|
| `ketchup status` | Resumo de saúde do workspace |
| `ketchup diff` | Detalhes do drift |
| `ketchup sync` | Sincronizar com confirmação |
| `ketchup catch-up` | Resumo do que mudou desde a última sessão |
| `ketchup doctor` | Validar configuração e ferramentas |

### Configuração

Crie `.ketchup.yml` na raiz do projeto. Veja [`.ketchup.example.yaml`](.ketchup.example.yaml).

### Princípios de segurança

- Checks (`status`, `diff`, `doctor`) são somente leitura
- Mudanças só via `ketchup sync` com confirmação explícita
- Git: apenas fast-forward
- Variáveis de ambiente: nomes comparados, valores nunca expostos

**Ketchup** — Mantenha seu workspace sempre em sync!
