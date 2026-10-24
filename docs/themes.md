# Make your own theme

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
