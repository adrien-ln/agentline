# Make your own theme

## With the editor

```bash
agentline theme new my-theme --from nord   # a copy of nord, named my-theme
agentline theme edit my-theme              # change it later (a built-in theme opens as a copy)
```

Or press `e` on a theme in the picker, or Tab to its **Create** tab. Each of the 17 colors is tuned by lightness, chroma or hue (`c` switches, ←→ tunes, Shift for bigger steps) or by typing its code (Enter). The contrast of each color against its background is shown, with ⚠ when text would be hard to read. The preview shows the statusline in three situations, on the other background with `f` and at 256 or 16 colors with `v`. `g` matches the gauge and effort colors to the chosen color, `G` builds the whole theme around it, `a` previews it as people with protanopia, deuteranopia or tritanopia see it (with a warning when gauges blend). `n` renames the theme, `s` saves it to `~/.config/agentline/themes/<name>.theme` without closing, with color tags computed from its colors; `u` drops the changes.

## From a terminal color scheme

```bash
agentline theme import Dracula.itermcolors              # iTerm2
agentline theme import ocean.yaml                       # base16
agentline theme import settings.json --scheme "One Half Dark"   # Windows Terminal
agentline theme import --terminal                       # the scheme of the current Windows Terminal profile
agentline theme export my-theme > my-theme.theme        # share a theme
```

The 16 ANSI colors become a theme (blue folder, cyan branch, magenta model, green to red gauges, each kept readable on the scheme's background), opened in the editor to adjust before saving with `s`.

## By hand

Copy a file from [`themes/`](../themes) to `~/.config/agentline/themes/<name>.theme`:

```bash
# Éclair
# tags: cyan, pink
TEXT=f8f8f2 MUTED=8b8fa8 SEP=3d3f58 DIM=5a5d78
MODEL=00ffff FOLDER=b967ff BRANCH=00d4ff MODEL_END=ff00ff
D0=39ff14 D50=ffe600 D75=ff8800 D100=ff1744
INTENSITY=(00d4ff 00ff9f b967ff ff2e97 ffe600)
```

`MODEL` → `MODEL_END`: model name gradient · `BRANCH`: branch and PR · `D0` → `D100`: gauges · `INTENSITY`: effort, from low to max. A theme made for a light background starts with `# background: light`; on the other background, colors are adjusted for contrast, or taken from `X_LIGHT=` (dark theme) or `X_DARK=` (light theme). The built-in themes use the original French names (`TEXTE`, `DISCRET`, `ETEINT`, `MODELE`, `MODELE_FIN`, `DOSSIER`, `LIEU`, `INTENSITE`, `_CLAIR`, `_SOMBRE`, `# fond: clair`, tags `rose`, `bleu`…), which stay valid.

Then pick it with `agentline` or `agentline theme set <name>`.
