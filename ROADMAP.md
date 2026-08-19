# Roadmap de Desenvolvimento - Ketchup

## Completado (v0.2.0)

### Core CLI
- [x] **Git Provider** — Detecção de drift, status ahead/behind, upstream tracking
- [x] **Dependencies Provider** — Suporte a npm/pnpm/yarn com detecção automática de lockfiles
- [x] **Environment Provider** — Comparação entre `.env` e `.env.example` (auto-skip se source ausente)
- [x] **JSON Output** — Flag `--json` em todos os comandos para integração
- [x] **Sistema de Atualização Remota** — Módulo updater autônomo com checksum SHA-256
- [x] **Script de Release** — Build multiplataforma automatizado (Windows, Linux, macOS)
- [x] **Alias `ff`** — Mesmo binário com help adaptado (`ketchup` / `ff`)
- [x] **Catch-up configurável** — Modos `relevant` / `all`, flag `--explain`, seção `catchup` no YAML
- [x] **UX do `diff`** — Mensagem quando limpo, `--drifted-only`, exit code 1 em drift, `--help`
- [x] **`doctor` com sugestões** — Indica próximo comando (`status`, `diff`, `sync`, `catch-up`)

### Extensão VS Code
- [x] **Ketchup Fast-Forward** — Publicada no VS Marketplace ([Reskyume.ketchup-fast-forward](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward))
- [x] **Publisher Reskyume** — Conta configurada e extensão verificada
- [x] **Ícone e README** — Assets de marketplace e documentação da extensão
- [x] **Status Bar Item** — Indicador visual do estado do workspace (clean/drifted/error)
- [x] **Quick-fix Buttons** — Ações contextuais nos findings da tree view
- [x] **Comando Catch-up** — Integração do `catch-up` com settings `catchUpShow` / `catchUpExplain`
- [x] **Auto-update Check** — Verificação automática de updates do core no startup
- [x] **Show Diff condicional** — Botão só aparece em providers com drift

### Infraestrutura
- [x] **Módulo Updater Testado** — Suite de testes unitários
- [x] **Versionamento SemVer** — Comparação semântica de versões
- [x] **Build `.vsix`** — Empacotamento via `@vscode/vsce`

---

## Pendente para v0.3.0

### Testes
- [ ] Testes para providers (Git, Dependencies, Environment)
- [ ] Testes de integração CLI
- [ ] Testes E2E da extensão VS Code
- [ ] Cobertura mínima de 80% do código

### Publicação e distribuição
- [ ] Publicar no **Open VSX Registry** (marketplace do Cursor / VSCodium)
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
- **CLI Go**: ~2800 linhas (providers + commands + updater + catch-up)
- **Extensão TS**: ~550 linhas (tree provider + commands + status bar)
- **Testes**: report + helpers CLI + updater
- **Cobertura**: parcial (updater, report, helpers)

### Plataformas suportadas
- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

### Próximos passos imediatos
1. Publicar no Open VSX para instalação nativa no Cursor
2. Expandir cobertura de testes para >80%
3. Configurar GitHub Actions (CI)
4. Sincronizar metadados do Marketplace após updates de `package.json`

---

**Última atualização**: Agosto 2026  
**Versão atual**: v0.2.0  
**Extensão**: [Ketchup Fast-Forward](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward) (VS Marketplace)  
**Repositório**: [github.com/Rukafuu/Ketchup](https://github.com/Rukafuu/Ketchup)  
**Próxima milestone**: v0.3.0 (Open VSX + testes + CI/CD)
