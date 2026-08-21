# Roadmap de Desenvolvimento - Ketchup

## Completado (v0.3.0)

### Core CLI
- [x] **Catch-up com resumo de arquivos** — Cada commit mostra arquivos alterados agrupados por área (`extension`, `CLI`, `core`, etc.)
- [x] **Parser Git corrigido** — `git log` parseado corretamente com `--name-only -z`
- [x] **Motivos específicos** — `Critical files changed: package.json` em vez de mensagem genérica
- [x] **Alias `ff`** — Mesmo binário com help adaptado (`ketchup` / `ff`)
- [x] **Catch-up configurável** — Modos `relevant` / `all`, flag `--explain`, seção `catchup` no YAML
- [x] **UX do `diff`** — Mensagem quando limpo, `--drifted-only`, exit code 1 em drift, `--help`
- [x] **`doctor` com sugestões** — Indica próximo comando (`status`, `diff`, `sync`, `catch-up`)
- [x] **Environment auto-skip** — Provider `env` ignorado quando source file não existe

### Extensão VS Code
- [x] **Ketchup Fast-Forward v0.3.0** — [VS Marketplace](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward)
- [x] **Metadados corrigidos** — Repositório apontando para [Rukafuu/Ketchup](https://github.com/Rukafuu/Ketchup)
- [x] **Show Diff condicional** — Botão só aparece em providers com drift
- [x] **Catch-up settings** — `catchUpShow` / `catchUpExplain` na extensão
- [x] **Open VSX** — [Ketchup Fast-Forward v0.3.0](https://open-vsx.org/extension/Reskyume/ketchup-fast-forward) (Cursor / VSCodium)

## Completado (v0.2.x)

- [x] Git, Dependencies e Environment providers
- [x] JSON output, updater remoto, release multiplataforma
- [x] Status bar, quick-fix buttons, auto-update check
- [x] Publicação inicial no VS Marketplace (publisher **Reskyume**)

---

## Pendente para v0.4.0

### Testes
- [ ] Testes para providers (Git, Dependencies, Environment)
- [ ] Testes de integração CLI
- [ ] Testes E2E da extensão VS Code
- [ ] Cobertura mínima de 80% do código

### Publicação e distribuição
- [ ] Pipeline CI/CD (build, test, release automático)
- [ ] GitHub Actions para validar PRs

### Melhorias futuras
- [ ] **Git Provider expandido** — Múltiplos remotes, cenários de merge complexos
- [ ] **Dependencies expandido** — pip, go modules, cargo
- [ ] **Environment avançado** — Validação de schema e valores default
- [ ] **Novos Providers** — Docker, Kubernetes configs, IDE settings
- [ ] **UI Dashboard** — Webview com visualização gráfica do drift
- [ ] **Modo Silent** — Auto-sync para ambientes CI/CD (opt-in)
- [ ] **Plugins System** — API para providers customizados

---

## Status do Projeto

### Código
- **CLI Go**: ~3000 linhas (providers + commands + catch-up + signals)
- **Extensão TS**: ~550 linhas (tree provider + commands + status bar)
- **Testes**: signals, git parser, report, helpers CLI, updater

### Plataformas suportadas
- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

### Próximos passos imediatos
1. Expandir cobertura de testes para >80%
2. Configurar GitHub Actions (CI)

---

**Última atualização**: Agosto 2026  
**Versão atual**: v0.3.0  
**Extensão**: [Ketchup Fast-Forward](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward)  
**Repositório**: [github.com/Rukafuu/Ketchup](https://github.com/Rukafuu/Ketchup)  
**Próxima milestone**: v0.4.0 (testes + CI/CD)
