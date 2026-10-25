# Roadmap

What is left to check, decide or build. These items will move to GitHub issues. The background of the rewrite is in [docs/HISTORY.md](docs/HISTORY.md).

## To check on real machines

Everything below is covered by automated tests or simulations only. The CI runs the tests on Linux (with the race detector), macOS and Windows, and the release workflow has published every version since v0.1.0, with the Scoop bucket and the Homebrew tap updated automatically.

- [ ] **The commit check on a pull request** (`.github/workflows/commits.yml`): so far it has only been skipped on Dependabot pull requests, never run on a contributor's.
- [ ] **The picker in a real terminal**: keys (including `b`, `u`, `?` and Esc step by step), resizing between the three layouts, smoothness of the animation (Windows Terminal, PowerShell, Git Bash, SSH), emoji width depending on the terminal.
- [ ] **The Statusline tab in a real terminal**: Shift+↑↓ (some terminals do not send it: K and J replace it), Tab, Space, typing a Nerd Font glyph, and how the glyphs of the `text` set (⎇ ⊕ ▸ ▴ ▿) look depending on the font.
- [ ] **Claude Code interface colors in the animated scenes** (`internal/selecteur/scenes.go`: orange of the thinking indicator, diff backgrounds): written from memory, to compare with a real session.
- [ ] **macOS**: only the build and the tests run, in the CI; `brew install`, `agentline install` and the picker are not tried. Terminal.app is detected as 256 colors (`TERM_PROGRAM=Apple_Terminal`): check the rendering, and whether the macOS 26 version shows 24-bit color (would `COLORTERM` say so?).
- [ ] **Color levels in real terminals**: 256 and 16 colors (`AGENTLINE_COLORS`), `NO_COLOR`, and what Claude Code actually passes to the statusline (`TERM`, `COLORTERM`, `TERM_PROGRAM`, `WT_SESSION`: assumed inherited from the terminal, not verified). mintty (Git Bash) is assumed to set `TERM_PROGRAM=mintty` and show 24-bit color.
- [ ] **Light background in a real terminal**: the OSC 11 query in the picker, `agentline install`, `agentline theme background` and `agentline doctor` (Windows Terminal, conhost, iTerm2, Terminal.app, GNOME, over SSH), and the readability of corrected dark themes and of the three light themes (`ETEINT=afb8c1` in GitHub Light is from memory, not taken from Primer).
- [ ] **Installation** (only simulated by the tests): HKCU PATH and `WM_SETTINGCHANGE` (does a new terminal see it without restarting Windows Terminal?), `agentline` in PowerShell, cmd and Git Bash, the PATH block in `~/.zshrc`, `~/.bashrc`, `config.fish`, then `agentline uninstall` undoing everything.
- [ ] **Upgrade from `sl`**: `agentline install` over an `sl` installation (old `sl.exe` renamed while a statusline may be running it, `sl-theme` and `sl-theme.cmd` set aside, PowerShell profile function removed, `~/.config/statusline` copied), then `agentline doctor` with nothing left.
- [ ] **Font**: `scoop install nerd-fonts/FiraCode-NF` (per user, without admin rights?), `brew install --cask font-fira-code-nerd-font`, the `FiraCode.zip` archive (`FiraCodeNerdFont-*.ttf` names, HKCU registration on Windows, `fc-cache -f`), then Windows Terminal and VS Code settings. No winget: no FiraCode Nerd Font identifier confirmed (only `DEVCOM.JetBrainsMonoNerdFont` was seen).
- [ ] **Y/n prompt in Git Bash** (mintty): stdin is not a console for Go there, so `agentline install` does not ask and prints the command instead. To check, and decide whether a flag should force it. Same for `agentline` alone, which prints the help outside a console instead of opening the picker.

## To decide

- [ ] **Clawd in public**: the mascot drawing belongs to Anthropic; check that showing it in a public project is fine.
- [ ] **`*` branch rule and detached HEAD**: a `*` branch rule also applies on a detached HEAD (the renderer then sees the start of the SHA). Acceptable for now.
- [ ] **Old `SL_*` names**: read as a fallback, deprecated. Decide when to remove them.

## Ideas

- [ ] At 100% of the 5-hour quota, the statusline still shows a projection (`↗103%`). This comes from the old bash version and is kept by the references (`tests/golden/`). Fixing it means updating the references (`go test ./internal/render -maj`).
- [ ] More animated scenes, or a scene chosen by project.
- [ ] `agentline theme gallery`: every theme at once, one statusline per theme.
- [ ] Favorites in the picker, theme editor.
- [ ] **Style per project** (`p` in the confirmation panel of the Statusline tab, shown as "soon"): one style file per project, with the same priority as themes.
- [ ] Statusline tab: in the original spec, ← closes the options panel, but ←→ already change the value there; today only Esc closes it.
- [ ] `agentline install --in-place` (Scoop, Homebrew): the binary path follows links to the version folder (`apps/agentline/<version>`), which changes on every update; aim at `current` or the shim instead.
- [ ] Colors: `FORCE_COLOR` and `CLICOLOR_FORCE` are not read; `agentline doctor` and `install` keep their ✓ ⚠ ✗ marks in basic colors even with `NO_COLOR`.
- [ ] Light background: hand-written `_CLAIR` variants for the most used dark themes, if the automatic correction is not good enough; an "on the other background" preview in the picker (key).
- [ ] `agentline doctor`: also check the font selected in Windows Terminal and VS Code (not only that a Nerd Font is installed).
