# Ketchup MVP 0.1 — desenho técnico

## 1. Crítica e recorte da arquitetura

A arquitetura por providers é adequada, mas a interface inicial mistura níveis
de abstração. `Plan` precisa receber o estado observado por `Check`, e `Apply`
deve executar uma operação imutável já planejada. Caso contrário, um provider
pode tomar uma decisão nova e invisível durante a execução.

O MVP terá um único pipeline:

1. carregar e validar a configuração;
2. executar o check de cada provider habilitado, sem mutações;
3. montar um snapshot do workspace e das mudanças previstas;
4. pedir operações planejadas aos providers;
5. exibir o plano completo e solicitar confirmação;
6. revalidar as precondições de cada operação;
7. aplicar somente as operações seguras confirmadas, na ordem de dependência;
8. executar os checks novamente e mostrar a saúde resultante.

`status`, `diff` e a fase de planejamento de `sync` usam os mesmos checks.
Somente a fase de apply do `sync` pode alterar o workspace. `doctor` apenas
inspeciona ferramentas e configuração.

O Engine controla orquestração, ordem, confirmação e isolamento de falhas. Os
providers controlam detecção, planejamento, validação das precondições e
execução de seu domínio. A CLI controla apresentação e interação.

### Fora do escopo da 0.1

- JSON, daemon, telemetria, plugins dinâmicos ou SDK público em Go;
- stash, reset, merge não-FF, rebase, troca de branch ou resolução de conflito;
- escrita automática no `.env`;
- auditoria exata de `node_modules` contra o lockfile;
- providers Docker, banco, n8n, OpenAPI, GraphQL e arquivos gerados.

## 2. Riscos técnicos e decisões

### Plano obsoleto e corrida entre check e apply

Branch, upstream, worktree, ponta remota e lockfile podem mudar depois da
exibição do plano. Cada operação carrega precondições e um fingerprint da
observação. `Apply` revalida ambos e retorna `STALE_PLAN` se algo mudou. Ele não
improvisa; o usuário precisa executar `ketchup sync` novamente.

### Segurança Git e atualidade das refs

Ahead/behind só é tão atual quanto as remote-tracking refs locais. `status` e
`diff` não fazem fetch implícito, pois checks não podem mutar o repositório; o
output informa qual ref/commit foi observado. `sync` pode planejar `git fetch`
como operação segura separada, mas depois do fetch deve refazer check e plano e
pedir nova confirmação.

A atualização usa o equivalente a `git merge --ff-only <upstream>`. Worktree
suja bloqueia o sync Git na 0.1, inclusive quando há untracked files. É uma
regra intencionalmente mais rígida, pois um arquivo remoto pode colidir com um
untracked. Submodules e worktrees serão apenas sinalizados no MVP.

### Drift de dependências não é diretamente comprovável

Não existe prova portátil e confiável de que uma instalação local corresponde
ao lockfile. Na 0.1, `dependency drift` significa que um lockfile reconhecido
está no conjunto de mudanças recebidas pelo fast-forward. Não significa que
`node_modules` foi auditado.

O Engine projeta as mudanças de arquivos a partir do relatório Git. O provider
Dependencies consome essa lista neutra; ele não importa nem chama o provider
Git. Depois do fast-forward, sua precondição confirma que o hash de lockfile
esperado está presente antes da instalação.

A escolha do package manager é determinística:

1. campo `packageManager` do `package.json`, quando suportado;
2. caso contrário, exatamente um lockfile reconhecido;
3. lockfiles conflitantes geram operação bloqueada.

Comandos: `npm ci`, `pnpm install --frozen-lockfile` e `yarn install
--immutable` para Yarn moderno. Yarn Classic deve usar seu equivalente frozen
e ter teste específico.

### Parsing de ambiente e proteção de secrets

O provider Environment interpreta nomes, comentários, linhas vazias e o
prefixo opcional `export`. Valores nunca entram em relatório, erro, log ou
fingerprint. Nomes duplicados ou inválidos geram warning.

Categorias da 0.1:

- `missing`: existe no source e não existe no target;
- `local-only`: existe no target e não existe no source;
- `common`: existe nos dois e não aparece no resumo padrão.

“Removida” é ambígua sem estado histórico. Uma chave `local-only` pode ter sido
removida do template ou ser intencionalmente privada; o Ketchup a informa e
nunca a apaga. Uma categoria histórica poderá ser adicionada depois, com
baseline persistente.

### Falha parcial

Os checks são independentes. Erro em um provider marca apenas ele como
`UNKNOWN` e preserva os resultados dos demais. Durante apply, uma falha bloqueia
operações dependentes. Operações independentes só continuam se suas próprias
precondições permanecerem válidas. O exit code final continua não zero.

## 3. Modelo de domínio

Os nomes concretos podem mudar na implementação, mas estas semânticas devem ser
preservadas:

```go
type Health string

const (
    HealthClean   Health = "CLEAN"
    HealthDrifted Health = "DRIFTED"
    HealthUnknown Health = "UNKNOWN"
)

type Finding struct {
    Code     string   // identificador estável para automação futura
    Severity Severity
    Summary  string   // texto seguro, nunca contém secrets
    Details  []Detail // fatos nomeados, renderizáveis e não secretos
}

type Report struct {
    Provider   string
    Health     Health
    Summary    string
    Findings   []Finding
    ObservedAt time.Time
    Revision   string // fingerprint opaco e não secreto
    Facts      Facts  // resultado tipado do provider
}
```

Na saúde agregada, `UNKNOWN` tem precedência sobre `DRIFTED`, que tem precedência
sobre `CLEAN`. Assim, a falha de um provider nunca produz um falso saudável.

Uma operação é a unidade atômica de um plano:

```go
type Disposition string

const (
    Safe    Disposition = "SAFE"
    Manual  Disposition = "MANUAL"
    Blocked Disposition = "BLOCKED"
)

type Operation struct {
    ID            string
    Provider      string
    Kind          string
    Description   string
    Disposition   Disposition
    DependsOn     []string
    Preconditions []Precondition
    Input         json.RawMessage // tipado/versionado pelo provider, sem secrets
}

type SyncPlan struct {
    ID          string
    CreatedAt   time.Time
    ProjectRoot string
    Operations  []Operation
}

type ApplyResult struct {
    OperationID string
    Status      string // APPLIED, SKIPPED, FAILED, STALE
    Summary     string
}
```

`Input` é produzido apenas pelo provider e tratado como opaco pelo Engine. Ele
contém somente o mínimo necessário para executar a operação. Operações `MANUAL`
são instruções e nunca chegam ao `Apply`. Entradas `BLOCKED` explicam por que
nenhuma operação segura pode ser oferecida.

## 4. Contratos entre Engine e providers

```go
type Provider interface {
    Name() string
    Check(context.Context, CheckRequest) (Report, error)
    Plan(context.Context, PlanRequest) ([]Operation, error)
    Validate(context.Context, Operation) error
    Apply(context.Context, Operation) (ApplyResult, error)
}

type CheckRequest struct {
    Root   string
    Config ProviderConfig
}

type PlanRequest struct {
    Root               string
    Config             ProviderConfig
    OwnReport          Report
    ProspectiveChanges []FileChange
}
```

Regras do contrato:

- `Check`, `Plan` e `Validate` são read-only, determinísticos e repetíveis;
- `Plan` usa a observação recebida e apenas leituras do estado atual;
- `Apply` aceita somente operação criada pelo mesmo provider;
- `Apply` executa exatamente o descrito, sem acrescentar etapas ou relaxar
  regras de segurança;
- `Validate` roda imediatamente antes de `Apply`; falha torna a operação stale;
- providers retornam códigos estáveis; a CLI controla texto e layout;
- providers nunca perguntam ao usuário; a CLI confirma o conjunto completo;
- erros são sanitizados e agregados depois que todos os checks terminam;
- dependências são declaradas por IDs de operação, sem ordem implícita.

Na 0.1, a única projeção entre providers é `ProspectiveChanges`, com paths e
hashes before/after. Git a produz, Dependencies a consome, e nenhum importa o
outro. Ainda não há motivo para event bus ou `map[string]any` genérico.

```text
git.fetch (opcional)
    -> novo check, novo plano e nova confirmação
git.fast_forward
    -> dependencies.install (se lockfile reconhecido mudar)

env.manual_update (somente exibição; nunca aplicado)
```

## 5. Comandos e exit codes

- `ketchup status`: resumo e saúde agregada, sem rede ou mutação;
- `ff diff`: findings e detalhes não secretos, sem rede ou mutação;
- `ketchup sync`: check, plano, confirmação `[y/N]`, validate/apply e novo check;
- `ketchup doctor`: valida config, raiz, Git/repositório, Node e package manager.

Stdin não interativo equivale a “não”, até existir uma flag explícita futura.
Ferramenta ausente só é erro quando o provider habilitado precisa dela.

| Código | Significado |
| ---: | --- |
| 0 | comando concluído; workspace limpo ou sync concluído |
| 1 | drift ou ação manual permanece |
| 2 | configuração ou uso da CLI inválido |
| 3 | check desconhecido, operação falhou ou plano ficou stale |

Texto do output não é API. Finding codes e exit codes são contratos estáveis.

## 6. Configuração

A CLI procura `.ketchup.yaml` somente na raiz do projeto. `version` é
obrigatório e campos desconhecidos são erro, evitando ignorar uma opção de
segurança digitada incorretamente. Paths são resolvidos dentro da raiz; escapar
dela é rejeitado.

Defaults conservadores:

- providers ficam desabilitados quando não configurados;
- a única estratégia Git é `fast-forward-only`;
- `auto_install` permite oferecer uma operação segura, mas nunca dispensa a
  confirmação do `ketchup sync`;
- sincronização de ambiente é sempre manual.

O repositório mantém `.ketchup.example.yaml`, não uma configuração ativa,
para não habilitar providers acidentalmente enquanto a CLI ainda não existe.

## 7. Estrutura final

```text
cmd/ff/main.go                    composition root
internal/cli/                     comandos, prompt, renderização, exits
internal/config/                  YAML, defaults e validação
internal/engine/                  orquestração check/plan/apply
internal/model/                   Report, Finding, Operation, SyncPlan
internal/providers/provider.go    interface e requests
internal/providers/git/           comandos, facts, plan e apply Git
internal/providers/dependencies/  detecção, lockfiles e instalação
internal/providers/env/           parser key-only e comparação
internal/exec/                    command runner injetável
internal/fs/                      helpers mínimos de filesystem
testdata/                         fixtures Git/env/package managers
docs/mvp-0.1-design.md
.ketchup.example.yaml
README.md
go.mod
go.sum
```

Não criar `pkg/` antes de existir um consumidor real da API pública. Manter
facts específicos dentro dos packages dos providers. Começar com standard
library, um pacote pequeno de CLI e um parser YAML; sem framework de DI,
plugin system, banco ou estado persistente.

## 8. Critérios objetivos de pronto

### Core e segurança

- config malformada, versão incompatível, campo desconhecido e path fora da
  raiz falham de modo fechado;
- falha de um provider mantém os outros visíveis e torna a saúde `UNKNOWN`;
- `status`, `diff`, `doctor`, `Check` e `Plan` não alteram arquivos, refs, index,
  worktree ou diretórios de dependências;
- sync começa em “não”, só aplica `SAFE` confirmado, respeita dependências,
  revalida precondições e refaz check;
- output e erros nunca contêm valores lidos de arquivos de ambiente;
- exit codes seguem o contrato.

### Git

- fixtures cobrem branch, upstream, ahead, behind, divergência, tracked changes
  e untracked files;
- fast-forward só é planejado para worktree limpa, com upstream e behind-only;
- dirty, untracked, sem upstream, ahead-only, detached ou diverged nunca geram
  fast-forward aplicável;
- apply usa ff-only e preserva commits e arquivos locais;
- upstream/worktree alterado entre plan e apply retorna `STALE_PLAN`;
- nenhum caminho invoca reset, clean, stash, force push, rebase ou checkout
  destrutivo.

### Dependencies

- npm, pnpm, Yarn moderno e Yarn Classic são detectados deterministicamente;
- evidências conflitantes ou não suportadas bloqueiam a operação;
- mudança remota em lockfile reconhecido planeja install após fast-forward;
- ausência de mudança em lockfile não planeja instalação;
- install só roda no sync confirmado, com comando frozen documentado;
- hash de lockfile inesperado bloqueia apply.

### Environment

- informa nomes missing e local-only nos arquivos configurados;
- suporta comentários, linhas vazias, valores quoted/vazios e `export`;
- chaves duplicadas/inválidas geram warning;
- snapshots de output provam que valores e secrets não aparecem;
- nenhuma operação escreve nos arquivos de ambiente.

### Usabilidade

- binários compilam em Windows, macOS e Linux no CI;
- README ganha instruções de build/install e exemplo de cinco minutos quando
  houver implementação;
- repositório descartável demonstra `status`, `diff`, sync seguro, recusa de
  sync inseguro e `doctor`.

## 9. Roadmap curto

1. **Fundação:** módulo, config, modelos, registry, Engine, renderer, exit codes e
   adapters testáveis de comandos/filesystem.
2. **Valor read-only:** Git check, Environment, `status`, `diff` e `doctor` —
   primeiro marco utilizável localmente.
3. **Sync Git seguro:** planos imutáveis, confirmação, precondições, fronteira
   fetch/replan, apply ff-only e check posterior.
4. **Sync de dependências:** detecção, lockfile prospectivo, install frozen,
   dependência entre operações e testes end-to-end multiplataforma.
5. **Hardening 0.1:** testes de secrets e stale plan, empacotamento, matriz CI e
   walkthrough de cinco minutos.

JSON, estado persistente e novos providers entram apenas depois do uso real da
0.1. Eles devem estender os contratos de report/operação, não reescrever o
pipeline de segurança.
