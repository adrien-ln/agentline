<div align="center">

<img src="docs/banniere.svg" alt="agentline: the Claude Code statusline, 49 themes, one binary" width="760">

Model, Git, PR, context and quotas on two lines, in the theme of your choice:<br>
one by default, one per project, one per branch. In English or French.

![Go 1.27](https://img.shields.io/badge/Go-1.27-7dcfff?style=flat-square&logo=go&logoColor=7dcfff&labelColor=1a1b26)
![49 themes](https://img.shields.io/badge/themes-49-bb9af7?style=flat-square&labelColor=1a1b26)
![Windows · Linux · macOS](https://img.shields.io/badge/platforms-Windows%20%C2%B7%20Linux%20%C2%B7%20macOS-9ece6a?style=flat-square&labelColor=1a1b26)
![2 dependencies](https://img.shields.io/badge/dependencies-2-e0af68?style=flat-square&labelColor=1a1b26)
[![MIT license](https://img.shields.io/badge/license-MIT-f7768e?style=flat-square&labelColor=1a1b26)](LICENSE)

</div>

<div align="center">

<img src="docs/demo/selecteur.gif" alt="The theme picker: filterable list, an animated Claude Code mini-session in the highlighted theme, and Clawd" width="760">

</div>

## Install

```bash
go install github.com/adrien-ln/agentline/cmd/agentline@latest
agentline install
```

Or from a clone:

```bash
git clone https://github.com/adrien-ln/agentline.git && cd agentline
go run ./cmd/agentline install
```

Requires Go 1.27+. `agentline install` takes care of the rest: binary on the PATH, statusline declared in `~/.claude/settings.json`, and a [Nerd Font](https://www.nerdfonts.com/) installed and set in your terminal if you don't have one. Every file it changes gets a `.bak` copy; `agentline doctor` checks everything and gives the command that fixes it, `agentline uninstall` removes it all (your settings stay).

The language follows `AGENTLINE_LANG=en` (or `fr`), otherwise `LC_ALL`, `LC_MESSAGES`, `LANG`, then the Windows display language; English by default.

<details>
<summary>Windows</summary>

- `~\bin\agentline.exe`: `agentline` works in PowerShell, cmd and Git Bash, without touching the PowerShell profile.
- `~\bin` is added to the user PATH, no administrator rights needed: open a new terminal.
- The font goes through Scoop if it's there; Windows Terminal and VS Code are configured.
</details>

<details>
<summary>macOS and Linux</summary>

- `~/.local/bin/agentline`, added to the PATH in your shell's file (`~/.zshrc`, `~/.bashrc` or `config.fish`).
- Font: `brew install --cask font-fira-code-nerd-font` on macOS, the official archive in `~/.local/share/fonts` on Linux. For other terminals, `agentline install` tells you where to pick the font.
- Over SSH, the statusline switches to one short line: your local terminal's font is what counts.
</details>

<details>
<summary>From the old name (sl)</summary>

`agentline install` replaces the old installation: the statusline runs `agentline`, the old `sl` binary and the `sl-theme` commands are set aside (`.bak` copies), the `sl-theme` function is removed from the PowerShell profile. Your settings in `~/.config/statusline` are copied once into `~/.config/agentline` (the old folder stays intact). The `SL_THEME`, `SL_CONFIG`… variables are still read, but deprecated: switch to `AGENTLINE_THEME`, `AGENTLINE_CONFIG`… `agentline doctor` reports what's left. The old French option names (`--sur-place`, `ctx.libelle`, `oui`/`non`, `TEXTE=`…) are still accepted.
</details>

## Pick a theme

```bash
agentline
```

<kbd>↑</kbd><kbd>↓</kbd> browse · <kbd>←</kbd><kbd>→</kbd> filter by color · <kbd>/</kbd> search · <kbd>Enter</kbd> keep for this **p**roject, this **b**ranch or everywhere (**d**efault) · <kbd>?</kbd> help

<div align="center">

<img src="docs/demo/statusline.png" alt="The statusline in the Tokyo Night, Miami, Prairie and Dracula themes" width="480">

</div>

## Tune each item

`Tab` in `agentline` (or `agentline style`) opens the **Statusline** tab: <kbd>Space</kbd> shows or hides an item, <kbd>K</kbd><kbd>J</kbd> moves it, <kbd>g</kbd> joins it to the previous one, <kbd>→</kbd> opens its options (style and width of the context bar, alert thresholds, icons, separators, labels…), <kbd>v</kbd> changes the preview. Colors stay the theme's.

```bash
agentline style set ctx.bar blocks     # or: agentline style hide pr, agentline style keys, agentline style reset
```

<details>
<summary>From the command line</summary>

| Command | Effect |
|---|---|
| `agentline theme set <theme>` | default theme |
| `agentline theme link <theme>` | theme of the current project (recognized by its remote) |
| `agentline theme branch "feat/*" <theme>` | theme of matching branches, in every project |
| `agentline theme current --why` | theme in effect, and why |
| `agentline help` | everything else |

Priority: `AGENTLINE_THEME` > branch > project > default. Settings in `~/.config/agentline` (or `$AGENTLINE_CONFIG`).
</details>

<details>
<summary>Make your own theme</summary>

Copy a file from [`themes/`](themes) to `~/.config/agentline/themes/<name>.theme`:

```bash
# Éclair
# tags: cyan, pink
TEXT=f8f8f2 MUTED=8b8fa8 SEP=3d3f58 DIM=5a5d78
MODEL=00ffff FOLDER=b967ff BRANCH=00d4ff MODEL_END=ff00ff
D0=39ff14 D50=ffe600 D75=ff8800 D100=ff1744
INTENSITY=(00d4ff 00ff9f b967ff ff2e97 ffe600)
```

`MODEL` → `MODEL_END`: model name gradient · `BRANCH`: branch and PR · `D0` → `D100`: gauges · `INTENSITY`: effort, from low to max. A theme made for a light background starts with `# background: light`; on the other background, colors are adjusted for contrast, or taken from `X_LIGHT=` (dark theme) or `X_DARK=` (light theme). The built-in themes use the original French names (`TEXTE`, `DISCRET`, `ETEINT`, `MODELE`, `MODELE_FIN`, `DOSSIER`, `LIEU`, `INTENSITE`, `_CLAIR`, `_SOMBRE`, `# fond: clair`, tags `rose`, `bleu`…), which stay valid.
</details>

## Contributing

Bug reports, themes and fixes are welcome: read [CONTRIBUTING.md](CONTRIBUTING.md) first, and report vulnerabilities privately ([SECURITY.md](SECURITY.md)).

```bash
gofmt -l . ; go vet ./... && go test ./...
```

Rendering is checked byte for byte against 212 reference outputs (color levels and light background included), plus 13 in English. History and design notes: [docs/HISTORY.md](docs/HISTORY.md) · Roadmap: [ROADMAP.md](ROADMAP.md) · License: [MIT](LICENSE)
