# Ketchup

[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.19-blue)](go.mod)
[![VSCode](https://img.shields.io/badge/vscode-1.85+-blue)](extension/package.json)

**Ketchup** is a CLI tool and VSCode extension that detects and synchronizes differences (drift) between your local development environment and the expected project state.

## Purpose

A security-conservative tool that:
- Detects differences in Git, dependencies, and environment variables
- Never alters the environment during checks (`status`, `diff`, `doctor`)
- Only applies changes via `ketchup sync` with explicit confirmation
- Guarantees **fast-forward only** operations in Git

## Quick Start

### 1. CLI (Command Line)

```bash
# Build
go build -o ketchup ./cmd/ketchup

# Add to PATH
export PATH=$PATH:$(pwd)

# Test
./ketchup help
```

### 2. VSCode Extension

```bash
cd extension
npm install
npm run compile

# In VSCode: F5 to run in development mode
```

## Commands

| Command | Description | Exit Codes |
|---------|-------------|------------|
| `ketchup status` | Summary and aggregated health (read-only) | 0=clean, 1=drift |
| `ketchup diff` | Drift details (read-only) | 0=clean, 1=drift |
| `ketchup sync` | Check + plan + confirmation + apply | 0=success, 1=manual |
| `ketchup doctor` | Validates configuration and tools | 0=ok, 1=failures |

### Example

```bash
# Check status
ketchup status

# View details
ketchup diff

# Synchronize (with confirmation)
ketchup sync

# Validate config
ketchup doctor
```

## VSCode Extension

The extension provides:

- **Activity Bar Panel**: Real-time provider status
- **Quick Commands**: Execute without leaving the editor
- **Notifications**: Drift alerts and sync completion
- **Auto-check**: Automatic verification when opening workspace

![VSCode Extension](assets/extension-screenshot.png)

### Extension Commands

- `Ketchup: Check Status`
- `Ketchup: Show Diff`
- `Ketchup: Sync Workspace`
- `Ketchup: Run Doctor`
- `Ketchup: Refresh Status`

### Settings

```json
{
  "ketchup.cliPath": "ketchup",
  "ketchup.autoCheckOnOpen": true,
  "ketchup.showNotifications": true
}
```

## Architecture

### Providers (MVP 0.1)

1. **Git** — Detects branch drift, allows only fast-forward
2. **Dependencies** — Detects lockfiles and installs only on `sync`
3. **Environment** — Compares variable names without exposing values

### Pipeline

```
config → check → snapshot → plan → confirmation → validate → apply → re-check
```

### Exit Codes

- `0`: Success (clean workspace or sync completed)
- `1`: Drift or pending manual action
- `2`: Invalid configuration
- `3`: Check failed or plan became obsolete

## Structure

```
.
├── cmd/ketchup/             # CLI entry point
├── internal/
│   ├── cli/                 # Commands and UI
│   ├── config/              # YAML configuration
│   ├── engine/              # Orchestration
│   ├── model/               # Domain types
│   └── providers/           # Git, Dependencies, Env
├── extension/               # VSCode Extension
│   ├── src/extension.ts
│   ├── package.json
│   └── README.md
├── testdata/                # Test fixtures
└── docs/                    # Documentation
```

## Configuration

Create `.ketchup.yml` or `.ketchup.yaml` in the root:

```yaml
providers:
  git:
    enabled: true
    allowedBranches: [main, develop]
  dependencies:
    enabled: true
    managers: [npm, go, pip]
  environment:
    enabled: true
    requiredVars: [DATABASE_URL, API_KEY]
```

## Development

### CLI

```bash
go build -o ketchup ./cmd/ketchup
go test ./...
```

### Extension

```bash
cd extension
npm install
npm run compile
npm run watch  # dev mode
vsce package   # package
```

## Roadmap

- [ ] Complete Git Provider
- [ ] Dependencies Provider (npm, go, pip)
- [ ] Environment Provider
- [ ] JSON output in CLI
- [ ] Quick-fix buttons in extension
- [ ] Status bar item
- [ ] Unit tests
- [ ] Publish on VS Marketplace

## Contributing

1. Fork
2. Branch (`git checkout -b feature/amazing`)
3. Commit (`git commit -m 'Add amazing'`)
4. Push (`git push origin feature/amazing`)
5. PR

## License

MIT License

---

**Ketchup** - Keep your workspace always in sync!

---

## Português (Brazilian Portuguese)

**Ketchup** é uma ferramenta CLI e extensão VSCode que detecta e sincroniza diferenças (*drift*) entre o ambiente local de desenvolvimento e o estado esperado do projeto.

### Propósito

Ferramenta conservadora de segurança que:
- Detecta diferenças no Git, dependências e variáveis de ambiente
- Nunca altera o ambiente durante checks (`status`, `diff`, `doctor`)
- Só aplica mudanças via `ketchup sync` com confirmação explícita
- Garante operações **fast-forward only** no Git

### Instalação Rápida

#### CLI (Linha de Comando)

```bash
# Build
go build -o ketchup ./cmd/ketchup

# Adicione ao PATH
export PATH=$PATH:$(pwd)

# Teste
./ketchup help
```

#### Extensão VSCode

```bash
cd extension
npm install
npm run compile

# No VSCode: F5 para rodar em modo de desenvolvimento
```

### Comandos

| Comando | Descrição | Exit Codes |
|---------|-----------|------------|
| `ketchup status` | Resumo e saúde agregada (read-only) | 0=limpo, 1=drift |
| `ketchup diff` | Detalhes do drift (read-only) | 0=limpo, 1=drift |
| `ketchup sync` | Check + plano + confirmação + apply | 0=sucesso, 1=manual |
| `ketchup doctor` | Valida configuração e ferramentas | 0=ok, 1=falhas |

#### Exemplo

```bash
# Verificar status
ketchup status

# Ver detalhes
ketchup diff

# Sincronizar (com confirmação)
ketchup sync

# Validar config
ketchup doctor
```

### Extensão VSCode

A extensão oferece:

- **Painel na Activity Bar**: Status dos providers em tempo real
- **Comandos Rápidos**: Execute sem sair do editor
- **Notificações**: Alertas de drift e conclusão de sync
- **Auto-check**: Verificação automática ao abrir workspace

#### Comandos da Extensão

- `Ketchup: Check Status`
- `Ketchup: Show Diff`
- `Ketchup: Sync Workspace`
- `Ketchup: Run Doctor`
- `Ketchup: Refresh Status`

#### Configurações

```json
{
  "ketchup.cliPath": "ketchup",
  "ketchup.autoCheckOnOpen": true,
  "ketchup.showNotifications": true
}
```

### Arquitetura

#### Providers (MVP 0.1)

1. **Git** — Detecta drift da branch, permite apenas fast-forward
2. **Dependencies** — Detecta lockfiles e instala apenas no `sync`
3. **Environment** — Compara nomes de variáveis sem expor valores

#### Pipeline

```
config → check → snapshot → plan → confirmação → validate → apply → re-check
```

#### Exit Codes

- `0`: Sucesso (workspace limpo ou sync concluído)
- `1`: Drift ou ação manual pendente
- `2`: Configuração inválida
- `3`: Check falhou ou plano ficou obsoleto

### Estrutura

```
.
├── cmd/ketchup/             # CLI entry point
├── internal/
│   ├── cli/                 # Comandos e UI
│   ├── config/              # Configuração YAML
│   ├── engine/              # Orquestração
│   ├── model/               # Tipos de domínio
│   └── providers/           # Git, Dependencies, Env
├── extension/               # Extensão VSCode
│   ├── src/extension.ts
│   ├── package.json
│   └── README.md
├── testdata/                # Fixtures de teste
└── docs/                    # Documentação
```

### Configuração

Crie `.ketchup.yml` ou `.ketchup.yaml` na raiz:

```yaml
providers:
  git:
    enabled: true
    allowedBranches: [main, develop]
  dependencies:
    enabled: true
    managers: [npm, go, pip]
  environment:
    enabled: true
    requiredVars: [DATABASE_URL, API_KEY]
```

### Desenvolvimento

#### CLI

```bash
go build -o ketchup ./cmd/ketchup
go test ./...
```

#### Extensão

```bash
cd extension
npm install
npm run compile
npm run watch  # dev mode
vsce package   # empacotar
```

### Roadmap

- [ ] Provider Git completo
- [ ] Provider Dependencies (npm, go, pip)
- [ ] Provider Environment
- [ ] Output JSON na CLI
- [ ] Quick-fix buttons na extensão
- [ ] Status bar item
- [ ] Tests unitários
- [ ] Publicar no VS Marketplace

### Contribuindo

1. Fork
2. Branch (`git checkout -b feature/amazing`)
3. Commit (`git commit -m 'Add amazing'`)
4. Push (`git push origin feature/amazing`)
5. PR

### Licença

MIT License

---

**Ketchup** - Mantenha seu workspace sempre em sync!
