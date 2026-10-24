# Image du README : la statusline en anglais dans quatre thèmes (statusline.png), dans un dépôt de démo « my-project ».
# Depuis la racine du dépôt, avec vhs, ttyd et ffmpeg : vhs docs/demo/statusline.tape
# Les réglages de la démo vont dans un dossier temporaire, vidé au départ (demo.tape y garde un style) : les tiens
# ne sont ni lus ni modifiés.

Set Shell "pwsh"
Set FontFamily "FiraCode Nerd Font Mono"
Set FontSize 16
# Taille ajustée au contenu (4 statuslines de 2 lignes, séparées par une ligne vide) : rien de vide à droite ni en bas.
Set Width 480
Set Height 300
Set Padding 18
Set Theme "TokyoNight"

Hide
Type `$d = Join-Path ([IO.Path]::GetTempPath()) "agentline-demo"; Remove-Item -Recurse -Force "$d\config" -ErrorAction SilentlyContinue; $env:AGENTLINE_CONFIG = "$d\config"; $env:AGENTLINE_LANG = "en"; $env:SSH_CONNECTION = $null; foreach ($v in "THEME", "STYLE", "PALETTE") { Remove-Item "Env:AGENTLINE_$v", "Env:SL_$v" -ErrorAction Ignore }; go build -o "$d\agentline.exe" ./cmd/agentline; Remove-Item -Recurse -Force "$d\my-project" -ErrorAction Ignore; git init -q -b main "$d\my-project"; Clear-Host`
Enter
Wait@60s
# Pas de ligne vide après le dernier thème, et une pause pour que l'invite n'apparaisse pas sur la capture
Type `Push-Location "$d\my-project"; Clear-Host; $n = 0; foreach ($t in "tokyo", "miami", "prairie", "dracula") { if ($n++) { "" }; & "$d\agentline.exe" theme preview $t }; [Console]::CursorVisible = $false; Start-Sleep 10; [Console]::CursorVisible = $true; Pop-Location`
Enter
Sleep 2s
Show
Sleep 200ms
Screenshot docs/demo/statusline.png
Sleep 500ms
