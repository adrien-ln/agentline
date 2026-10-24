# Changelog

All notable changes to agentline are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The notes of each GitHub release list every commit; this file keeps what matters to users.

## [Unreleased]

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

[Unreleased]: https://github.com/adrien-ln/agentline/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/adrien-ln/agentline/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/adrien-ln/agentline/releases/tag/v0.1.0
