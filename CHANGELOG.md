# Changelog

All notable changes to Ketchup are documented in this file.

## [0.3.0] - 2026-08-19

### Added
- **Catch-up file summaries** — each commit now shows modified files grouped by area (`extension`, `CLI`, `core`, `docs`, etc.)
- **`ff` CLI alias** — same binary as `ketchup`; help text adapts to the executable name
- **`catchup` command alias** for `catch-up`
- **Configurable catch-up** via `.ketchup.yaml` (`catchup.show`, `catchup.explain`, `max_relevant`)
- **`diff --drifted-only`** — show only providers with drift or errors
- **`diff --help`** and **`doctor --help`** with usage examples
- **`doctor` next-step hints** — suggests the next command after a successful check

### Changed
- **`diff` exit code** now returns `1` when drift or unknown providers are detected (aligned with `status`)
- **`diff` clean state** prints a clear message instead of listing every clean provider
- **Relevance reasons** now name specific critical files (e.g. `Critical files changed: package.json`)
- **Environment provider** is skipped automatically when the source file (e.g. `.env.example`) does not exist
- **VS Code extension**: Show Diff button only appears on drifted providers; catch-up respects `catchUpShow` / `catchUpExplain` settings
- **Extension metadata** updated to point to [Rukafuu/Ketchup](https://github.com/Rukafuu/Ketchup)

### Fixed
- **Git log parser** for catch-up — file lists no longer mixed with commit metadata when using `--name-only -z`
- **VSIX packaging** excludes local `.ketchup/` session data

### Extension
- [Ketchup Fast-Forward v0.3.0](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward) on the VS Marketplace

---

## [0.2.1] - 2026-08-19

### Changed
- Extension publisher set to **Reskyume**
- Extension renamed to **Ketchup Fast-Forward** (`Reskyume.ketchup-fast-forward`)
- README and marketplace metadata corrections

---

## [0.2.0] - 2026-08-19

### Added
- **Ketchup Fast-Forward** VS Code extension published on the [VS Marketplace](https://marketplace.visualstudio.com/items?itemName=Reskyume.ketchup-fast-forward)
- Activity bar panel with provider status and expandable findings
- Status bar indicator (clean / drift / error) with drift count
- Quick-fix actions on findings (catch-up, sync)
- Auto-check on workspace open and optional CLI update check
- Catch-up integration in the extension

### CLI
- Git, Dependencies, and Environment drift providers
- `status`, `diff`, `sync`, `doctor`, `catch-up`, `update` commands
- JSON output (`--json`) on all commands
- Remote auto-update module with SHA-256 checksum verification
- Multi-platform release script (Windows, Linux, macOS)

---

## [0.1.0] - 2026-08-19

### Added
- Initial Ketchup CLI MVP
- Workspace drift detection pipeline
- Session-based catch-up with relevance filtering
- `.ketchup.yaml` configuration support

[0.3.0]: https://github.com/Rukafuu/Ketchup/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/Rukafuu/Ketchup/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Rukafuu/Ketchup/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rukafuu/Ketchup/releases/tag/v0.1.0
