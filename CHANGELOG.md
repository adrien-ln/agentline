# Changelog

All notable changes to agentline are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The notes of each GitHub release list every commit; this file keeps what matters to users.

## [Unreleased]

### Added

- `agentline font preview` tries the installed Nerd Fonts live in Windows Terminal or VS Code: each one really becomes the terminal font while you choose (a .bak copy is made), Enter keeps it, Esc puts yours back. An interrupted preview is reported by doctor and undone by the next preview.
- A theme editor: the picker's new Create tab (or `e` on a theme, `agentline theme new <name> [--from <theme>]`, `agentline theme edit <theme>`) tunes each color by lightness, chroma or hue or by its code, warns about low contrast, and previews the statusline on either background and at 256 or 16 colors; `s` saves it to your themes folder.
- `agentline font test` draws every glyph agentline uses in the current font, so a missing or too wide glyph shows at a glance.

## [0.6.0] - 2026-10-25

### Changed

- The picker saves in one place: Enter chooses a theme, p, b or d sets where it applies, and s saves the theme and the statusline style together without closing. A bar shows what is not saved yet, u forgets it, and q asks before quitting with unsaved changes. Esc only steps back.
- Keys are shown as small caps next to what they do, with their Nerd Font glyph for Enter, Tab, Esc, Backspace and Shift; the highlighted theme shows the key that acts on it.

## [0.5.0] - 2026-10-25

### Added

- Alerts that say what to do: a context in alert is followed by a `/compact` hint in Claude Code (`ctx.compact`), and uncommitted changes on `main` or `master` turn the branch red and put the git segment in alert (`git.protected`, set it to your own list or leave it empty to turn it off).

### Changed

- The picker mascot is now Pip, a chick perched on the window like a bird on a wire, in one color of the theme. The `m` key switches it between solid and hidden. It replaces Clawd, whose drawing belongs to Anthropic.
- `go install` needs Go 1.26 instead of 1.27.

### Fixed

- A quota already at 100% no longer shows a projection (`↗103%`).

## [0.4.0] - 2026-10-25

### Added

- Antigravity CLI and Copilot CLI: `agentline render --from antigravity|copilot` reads their statusline input (recognized from the JSON without `--from`), and `agentline install` offers to set their statusline when they are installed.

## [0.3.2] - 2026-10-25

### Changed

- The picker draws 2 frames per second instead of 20 while the mascot sleeps, and each frame is about 20% faster to build.

### Fixed

- When `git status` gives nothing, the branch is read correctly in worktrees and submodules, with its full name (`feat/x`), so branch theme rules keep applying.
- The git cache shared by sessions in the same repository is written atomically.

## [0.3.1] - 2026-10-25

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

[Unreleased]: https://github.com/adrien-ln/agentline/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/adrien-ln/agentline/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/adrien-ln/agentline/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/adrien-ln/agentline/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/adrien-ln/agentline/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/adrien-ln/agentline/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/adrien-ln/agentline/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/adrien-ln/agentline/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/adrien-ln/agentline/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/adrien-ln/agentline/releases/tag/v0.1.0
