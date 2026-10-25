# Changelog

All notable changes to agentline are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The notes of each GitHub release list every commit; this file keeps what matters to users.

## [Unreleased]

### Changed

- The Nerd Font is downloaded with `curl` instead of a network stack built into the binary: the binary is about 40% smaller and each statusline render starts faster.

### Fixed

- `agentline theme set` and `theme background` write their setting atomically, so a render at the same moment never reads an empty file.
- `agentline install` replaces `settings.json` and the other files it changes atomically, following symbolic links, so a crash never leaves them half written.

## [0.3.0] - 2026-10-25

### Added

- `agentline install` asks which Nerd Font to install: Fira Code, JetBrains Mono, Cascadia Code, Hack or Meslo. `--font <name>` picks it without the question.
- `agentline font` lists the fonts, the installed ones and the font of Windows Terminal and VS Code; `agentline font set <name>` installs a font if needed and selects it there.

### Changed

- `agentline doctor` names the installed font and warns when Windows Terminal or VS Code is set to a font that is not a Nerd Font.

## [0.2.0] - 2026-10-25

### Changed

- The Statusline tab edits the statusline in place: ←→ and ↑↓ pick an item on the live preview, Enter opens its menu (options, join, remove), + adds an item, g opens the general options, s keeps the style.

## [0.1.1] - 2026-10-25

### Changed

- Releases ship archives instead of bare binaries: `.zip` for Windows, installed with Scoop (`scoop bucket add agentline https://github.com/adrien-ln/scoop-agentline`), `.tar.gz` for macOS and Linux, installed with Homebrew (`brew install adrien-ln/agentline/agentline`) or the install script.
- Over SSH, the full statusline is now the default; the short line is opt-in with `agentline style set ssh.short yes`.

### Fixed

- The g of the picker logo has its descender again.
- The option panel of the Statusline tab is aligned with the item list and the preview.

## [0.1.0] - 2026-10-24

First public release.

### Added

- The Claude Code statusline on two lines (model, Git, pull request, context, quotas), one short line over SSH.
- 49 built-in themes, custom themes, one theme by default, per project or per branch.
- The theme picker (`agentline`) with a live preview, and the **Statusline** tab to show, hide, move and tune each item.
- `agentline install`, `doctor` and `uninstall` on Windows, Linux and macOS, with a `.bak` copy of every file changed.
- English and French; 256 colors, 16 colors and no color; light and dark backgrounds.
- Release binaries for Windows, Linux and macOS (amd64 and arm64) with `SHA256SUMS` and build provenance.

[Unreleased]: https://github.com/adrien-ln/agentline/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/adrien-ln/agentline/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/adrien-ln/agentline/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/adrien-ln/agentline/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/adrien-ln/agentline/releases/tag/v0.1.0
