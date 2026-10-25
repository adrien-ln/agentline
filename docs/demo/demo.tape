# Démo du README : le sélecteur (selecteur.gif), en anglais (AGENTLINE_LANG=en), centrée sur les nouveautés : validation par projet,
# branche ou partout, puis l'onglet Statusline (chaque élément réglable) et `agentline style preview`.
# La statusline seule : statusline.tape. Depuis la racine du dépôt, avec vhs, ttyd et ffmpeg : vhs docs/demo/demo.tape
# Les réglages de la démo vont dans un dossier temporaire : les tiens ne sont ni lus ni modifiés. Le sélecteur
# s'ouvre dans un dépôt de démo « agentline » (temporaire lui aussi), pour un en-tête qui ne dépend pas du clone.
# Les prédictions de PSReadLine sont coupées : sans ça, ton historique s'affiche en gris dès la première lettre tapée.

Output docs/demo/selecteur.gif

Set Shell "pwsh"
Set FontFamily "FiraCode Nerd Font Mono"
Set FontSize 16
Set Width 1400
Set Height 760
Set Padding 18
Set Theme "TokyoNight"
Set Framerate 24
Set TypingSpeed 70ms

Hide
Type `Set-PSReadLineOption -PredictionSource None; $d = Join-Path ([IO.Path]::GetTempPath()) "agentline-demo"; Remove-Item -Recurse -Force "$d\config" -ErrorAction SilentlyContinue; $env:AGENTLINE_CONFIG = "$d\config"; $env:AGENTLINE_LANG = "en"; $env:SSH_CONNECTION = $null; foreach ($v in "THEME", "STYLE", "PALETTE") { Remove-Item "Env:AGENTLINE_$v", "Env:SL_$v" -ErrorAction Ignore }; go build -o "$d\agentline.exe" ./cmd/agentline; $env:PATH = "$d;$env:PATH"; Remove-Item -Recurse -Force "$d\agentline" -ErrorAction Ignore; git init -q -b main "$d\agentline"; git -C "$d\agentline" remote add origin https://github.com/adrien-ln/agentline.git; Set-Location "$d\agentline"; Clear-Host`
Enter
Wait@60s
Show

Type "agentline"
Sleep 500ms
Enter
Sleep 3s

# les thèmes : parcourir, filtrer par couleur
Down@900ms 3
Sleep 1s
Right 4
Sleep 2s
Left 4
Sleep 800ms

# valider : le panneau propose ce projet, la branche, partout. Échap : retour
Enter
Sleep 3s
Escape
Sleep 1s

# l'onglet Statusline : on choisit un élément sur la statusline même
Tab
Sleep 2.5s
# aller jusqu'à la PR, puis la retirer
Right@450ms 6
Sleep 1s
Type "x"
Sleep 1.5s
# la barre du contexte, sur la ligne 2 : son menu, son style, puis sa largeur
Down
Left@400ms 3
Sleep 800ms
Enter
Sleep 1.2s
Right@900ms 2
Sleep 1s
Down
Right@500ms 4
Sleep 1.5s
Escape
# l'aperçu tendu : alertes, conflit, plafond
Type "v"
Sleep 2.5s
# garder pour tous les projets
Type "s"
Sleep 1.5s
Type "d"
Sleep 2.5s

# le résultat, en ligne de commande
Type "agentline style preview"
Sleep 500ms
Enter
Sleep 4s
