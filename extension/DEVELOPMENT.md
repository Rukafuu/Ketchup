# Ketchup Extension - Development Notes

## Build & Test

```bash
# Install dependencies
cd extension
npm install

# Compile TypeScript
npm run compile

# Watch mode for development
npm run watch

# Lint
npm run lint

# Package for distribution
vsce package
```

## Testing the Extension

1. Open this folder in VSCode
2. Press `F5` to launch Extension Development Host
3. In the new window, open a workspace with `.ff.yml`
4. The Ketchup icon will appear in the Activity Bar

## Key Features Implemented

### Tree View Provider
- Shows status of each provider (Git, Dependencies, Environment)
- Color-coded health indicators (green=clean, yellow=drifted, blue=unknown)
- Expandable items for findings

### Commands
- `ketchup.refresh` - Refresh tree view
- `ketchup.status` - Run status and show output
- `ketchup.diff` - Show detailed diff
- `ketchup.sync` - Execute sync with notifications
- `ketchup.doctor` - Validate configuration

### Configuration
- `ketchup.cliPath` - Custom CLI path (default: "ff")
- `ketchup.autoCheckOnOpen` - Auto-check on workspace open
- `ketchup.showNotifications` - Enable/disable notifications

### Output Channel
All command outputs are logged to "Ketchup" output channel for debugging.

## Next Steps

1. **Add JSON output to CLI**: Modify Go CLI to support `--json` flag for better parsing
2. **Add inline actions**: Quick-fix buttons for common drift issues
3. **Status bar item**: Show health indicator in status bar
4. **Badge counter**: Show number of drifted providers on icon
5. **Tests**: Add unit tests for extension logic

## Publishing

```bash
# Install vsce globally
npm install -g @vscode/vsce

# Login to VS Marketplace
vsce login <publisher-name>

# Package
vsce package

# Publish
vsce publish
```

## Debugging Tips

- Check "Extension Host" logs in Developer Tools
- Use `console.log()` in extension.ts (visible in Extension Host output)
- Test with different exit codes by mocking CLI responses
