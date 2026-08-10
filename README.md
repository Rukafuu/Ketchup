<p align="center">
  <img src="assets/ketchup-logo.png" alt="FastForward — never run out of sync" width="800">
</p>

# FastForward (`ff`) + VSCode Extension 🚀

[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.19-blue)](go.mod)
[![VSCode](https://img.shields.io/badge/vscode-1.85+-blue)](extension/package.json)

**FastForward** é uma ferramenta CLI e extensão VSCode que detecta e sincroniza diferenças (*drift*) entre o ambiente local de desenvolvimento e o estado esperado do projeto.

## 🎯 Propósito

Ferramenta conservadora de segurança que:
- ✅ Detecta diferenças no Git, dependências e variáveis de ambiente
- ✅ Nunca altera o ambiente durante checks (`status`, `diff`, `doctor`)
- ✅ Só aplica mudanças via `ff sync` com confirmação explícita
- ✅ Garante operações **fast-forward only** no Git

## 📦 Instalação Rápida

### 1. CLI (Linha de Comando)

```bash
# Build
go build -o ff ./cmd/ff

# Adicione ao PATH
export PATH=$PATH:$(pwd)

# Teste
./ff help
```

### 2. Extensão VSCode

```bash
cd extension
npm install
npm run compile

# No VSCode: F5 para rodar em modo de desenvolvimento
```

## 🚀 Comandos

| Comando | Descrição | Exit Codes |
|---------|-----------|------------|
| `ff status` | Resumo e saúde agregada (read-only) | 0=limpo, 1=drift |
| `ff diff` | Detalhes do drift (read-only) | 0=limpo, 1=drift |
| `ff sync` | Check + plano + confirmação + apply | 0=sucesso, 1=manual |
| `ff doctor` | Valida configuração e ferramentas | 0=ok, 1=falhas |

### Exemplo

```bash
# Verificar status
ff status

# Ver detalhes
ff diff

# Sincronizar (com confirmação)
ff sync

# Validar config
ff doctor
```

## 🖥️ Extensão VSCode

A extensão oferece:

- **Painel na Activity Bar**: Status dos providers em tempo real
- **Comandos Rápidos**: Execute sem sair do editor
- **Notificações**: Alertas de drift e conclusão de sync
- **Auto-check**: Verificação automática ao abrir workspace

![Extensão VSCode](assets/extension-screenshot.png)

### Comandos da Extensão

- `FastForward: Check Status`
- `FastForward: Show Diff`
- `FastForward: Sync Workspace`
- `FastForward: Run Doctor`
- `FastForward: Refresh Status`

### Configurações

```json
{
  "fastforward.cliPath": "ff",
  "fastforward.autoCheckOnOpen": true,
  "fastforward.showNotifications": true
}
```

## 🏗️ Arquitetura

### Providers (MVP 0.1)

1. **Git** — Detecta drift da branch, permite apenas fast-forward
2. **Dependencies** — Detecta lockfiles e instala apenas no `sync`
3. **Environment** — Compara nomes de variáveis sem expor valores

### Pipeline

```
config → check → snapshot → plan → confirmação → validate → apply → re-check
```

### Exit Codes

- `0`: Sucesso (workspace limpo ou sync concluído)
- `1`: Drift ou ação manual pendente
- `2`: Configuração inválida
- `3`: Check falhou ou plano ficou obsoleto

## 📁 Estrutura

```
.
├── cmd/ff/                  # CLI entry point
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

## ⚙️ Configuração

Crie `.ff.yml` ou `.ff.yaml` na raiz:

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

## 🔧 Desenvolvimento

### CLI

```bash
go build -o ff ./cmd/ff
go test ./...
```

### Extensão

```bash
cd extension
npm install
npm run compile
npm run watch  # dev mode
vsce package   # empacotar
```

## 📋 Roadmap

- [ ] Provider Git completo
- [ ] Provider Dependencies (npm, go, pip)
- [ ] Provider Environment
- [ ] Output JSON na CLI
- [ ] Quick-fix buttons na extensão
- [ ] Status bar item
- [ ] Tests unitários
- [ ] Publicar no VS Marketplace

## 🤝 Contribuindo

1. Fork
2. Branch (`git checkout -b feature/amazing`)
3. Commit (`git commit -m 'Add amazing'`)
4. Push (`git push origin feature/amazing`)
5. PR

## 📄 Licença

MIT License

---

**FastForward** - Mantenha seu workspace sempre em sync! 🚀
