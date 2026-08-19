# Roadmap de Desenvolvimento - Ketchup

## ✅ Completado (v0.2.0)

### Core CLI
- [x] **Git Provider** - Detecção de drift, status ahead/behind, upstream tracking
- [x] **Dependencies Provider** - Suporte a npm/pnpm/yarn com detecção automática de lockfiles
- [x] **Environment Provider** - Comparação entre .env e .env.example
- [x] **JSON Output** - Flag `--json` em todos os comandos para integração
- [x] **Sistema de Atualização Remota** - Módulo updater autônomo com checksum SHA-256
- [x] **Script de Release** - Build multiplataforma automatizado (Windows, Linux, macOS)

### Extensão VSCode/Cursor
- [x] **Renomeação para Ketchup** - Migração completa do branding FastForward → Ketchup
- [x] **Status Bar Item** - Indicador visual do estado do workspace (clean/drifted/error)
- [x] **Quick-fix Buttons** - Ações rápidas contextuais nos findings da tree view
  - Git drift → Catch-up branch
  - ENV/DEP issues → Sync workspace
- [x] **Comando Catch-up** - Integração do comando `catch-up` na extensão
- [x] **Auto-update Check** - Verificação automática de updates do core no startup
- [x] **Configurações de Update** - Canais stable/beta/nightly configuráveis
- [x] **Tree View Aprimorada** - Ícones de severidade e contagem de issues
- [x] **Compilação TypeScript** - Build completo para JavaScript

### Infraestrutura
- [x] **Módulo Updater Testado** - Suite de testes unitários (10+ testes passando)
- [x] **Versionamento SemVer** - Comparação semântica de versões
- [x] **Build Compilado** - Extensão pronta para publicação (.vsix)

---

## 📋 Pendente para v0.3.0

### Testes Unitários Expandidos
- [ ] Testes para providers (Git, Dependencies, Environment)
- [ ] Testes de integração CLI
- [ ] Testes E2E da extensão VSCode
- [ ] Cobertura mínima de 80% do código

### Publicação
- [ ] Configurar conta publisher no VS Marketplace
- [ ] Criar ícone e assets de marketing
- [ ] Escrever README detalhado para marketplace
- [ ] Configurar pipeline CI/CD para publicação automática
- [ ] Publicar versão 0.2.0 no VS Marketplace
- [ ] Publicar no Open VSX Registry

### Melhorias Futuras
- [ ] **Git Provider Completo** - Suporte a múltiplos remotes, rebase automático
- [ ] **Dependencies Provider Expandido** - Suporte a pip (Python), go modules, cargo (Rust)
- [ ] **Environment Provider Avançado** - Validação de schema, tipos, valores default
- [ ] **Novos Providers** - Docker, Kubernetes configs, IDE settings
- [ ] **UI Dashboard** - Webview com visualização gráfica do drift
- [ ] **Notificações Inteligentes** - Machine learning para priorizar issues críticas
- [ ] **Modo Silent** - Auto-sync sem confirmação para ambientes CI/CD
- [ ] **Plugins System** - API para providers customizados

---

## 📊 Status do Projeto

### Código
- **CLI Go**: ~2500 linhas (providers + commands + updater)
- **Extensão TS**: ~480 linhas (tree provider + commands + status bar)
- **Testes**: 10 testes unitários (módulo updater)
- **Cobertura**: ~40% (apenas updater)

### Plataformas Suportadas
- ✅ Windows (amd64, arm64)
- ✅ Linux (amd64, arm64)
- ✅ macOS (amd64, arm64)

### Próximos Passos Imediatos
1. Expandir cobertura de testes para >80%
2. Configurar GitHub Actions para CI/CD
3. Preparar assets de publicação (ícone, screenshots)
4. Submeter para revisão no VS Marketplace
5. Anunciar lançamento nas comunidades (Dev.to, Hashnode, Reddit)

---

**Última atualização**: Dezembro 2024  
**Versão atual**: v0.2.0  
**Próxima milestone**: v0.3.0 (testes + publicação)
