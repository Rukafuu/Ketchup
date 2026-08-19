# Ketchup Extension — Development Notes

## Setup

```bash
cd extension
npm install
npm run compile
```

## Scripts

| Script | Description |
|--------|-------------|
| `npm run compile` | Build TypeScript to `out/` |
| `npm run watch` | Watch mode for development |
| `npm run lint` | Run ESLint |
| `npm run package` | Build `.vsix` with `@vscode/vsce` |

## Testing locally

1. Open the repository root in VS Code
2. Run **Run Extension** from `extension/` (F5)
3. In the Extension Development Host, open a workspace with `.ketchup.yml`
4. Confirm the Ketchup icon appears in the Activity Bar

## Package for distribution

```bash
npm run compile
npm run package
# Output: ketchup-0.2.0.vsix
```

Install manually via **Extensions → Install from VSIX**.

## Publishing to VS Marketplace

```bash
npm install -g @vscode/vsce
vsce login ketchup-ai
npm run publish
```

Requires a [publisher account](https://marketplace.visualstudio.com/manage) and verified publisher ID.

## Debugging

- Extension Host logs: **Help → Toggle Developer Tools → Console**
- CLI output: **View → Output → Ketchup**
- Test exit codes by running CLI commands manually in the workspace terminal

## Implemented features (v0.2.0)

- Tree view with provider health and findings
- Status bar indicator with drift count
- Quick-fix commands on findings
- JSON + text fallback parsing
- Shared output channel
- Badge on sidebar when drift is detected
- Auto-check on open and optional CLI update check

## Next steps

- Unit tests for JSON normalization logic
- E2E tests with mocked CLI
- Marketplace assets (screenshots, demo GIF)
