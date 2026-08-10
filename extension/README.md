# FastForward VSCode Extension

Extensão do Visual Studio Code para o FastForward - detecta e sincroniza diferenças entre o ambiente local e o estado esperado do projeto.

## Funcionalidades

- **Painel na Activity Bar**: Visualize o status dos providers (Git, Dependências, Variáveis de Ambiente) diretamente na sidebar
- **Comandos Rápidos**: Execute `status`, `diff`, `sync` e `doctor` sem sair do editor
- **Notificações Inteligentes**: Receba alertas quando drift for detectado ou sync for concluído
- **Auto-check**: Verificação automática ao abrir o workspace
- **Output Channel**: Logs detalhados em um canal dedicado no VSCode

## Instalação

### Pré-requisitos
- Ter a CLI do FastForward (`ff`) instalada e no PATH
- VSCode 1.85.0 ou superior

### Desenvolvimento Local

1. Instale as dependências:
```bash
cd extension
npm install
```

2. Compile o TypeScript:
```bash
npm run compile
```

3. No VSCode, pressione `F5` para rodar a extensão em modo de desenvolvimento

4. Para empacotar:
```bash
npm install -g vsce
vsce package
```

## Comandos Disponíveis

| Comando | Descrição | Atalho |
|---------|-----------|--------|
| `FastForward: Check Status` | Verifica saúde geral do workspace | Palette / Sidebar |
| `FastForward: Show Diff` | Mostra detalhes das diferenças | Palette / Sidebar |
| `FastForward: Sync Workspace` | Sincroniza workspace com confirmação | Palette / Explorer |
| `FastForward: Run Doctor` | Valida configuração e ferramentas | Palette |
| `FastForward: Refresh Status` | Atualiza painel da sidebar | Botão no painel |

## Configurações

Adicione ao seu `settings.json`:

```json
{
  "fastforward.cliPath": "ff",  // Caminho para CLI se não estiver no PATH
  "fastforward.autoCheckOnOpen": true,  // Auto-check ao abrir workspace
  "fastforward.showNotifications": true  // Mostrar notificações
}
```

## Uso

1. Abra um projeto com configuração `.ff.yml` ou `.ff.yaml`
2. O ícone do FastForward aparecerá na Activity Bar (ícone de git-pull-request)
3. Clique para ver o status dos providers
4. Use os comandos da palette (`Ctrl+Shift+P` → "FastForward") para ações

## Estrutura da Extensão

```
extension/
├── src/
│   └── extension.ts      # Código principal da extensão
├── out/                   # Compilado (gerado automaticamente)
├── package.json          # Manifesto da extensão
├── tsconfig.json         # Configuração TypeScript
└── README.md             # Esta documentação
```

## Integração com a CLI

A extensão executa a CLI `ff` como subprocesso e parseia a saída:
- Tenta primeiro o formato JSON (`--json`)
- Fallback para parsing do output texto padrão
- Respeita os exit codes da CLI (0, 1, 2, 3)

## Contribuindo

1. Fork o repositório
2. Crie uma branch para sua feature
3. Teste com `npm run watch` durante desenvolvimento
4. Envie um PR

## License

MIT
