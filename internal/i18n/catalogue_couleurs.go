package i18n

// couleurs : niveau de couleurs et fond du terminal (doctor, agentline theme background, installation).
var couleurs = map[string]string{
	"Couleurs":     "Colors",
	"Fond":         "Background",
	"24 bits":      "24-bit",
	"256 couleurs": "256 colors",
	"16 couleurs":  "16 colors",
	"sans couleur": "no color",
	"couleurs ANSI fixes de la ligne courte (SSH)":                     "fixed ANSI colors of the short line (SSH)",
	"pour en changer : AGENTLINE_COLORS=truecolor, 256, 16 ou none":    "to change it: AGENTLINE_COLORS=truecolor, 256, 16 or none",
	"AGENTLINE_COLORS=%s inconnue, ignorée : %s":                       "AGENTLINE_COLORS=%s unknown, ignored: %s",
	" : si le terminal affiche le 24 bits, AGENTLINE_COLORS=truecolor": ": if the terminal shows 24-bit color, AGENTLINE_COLORS=truecolor",
	" : le terminal a un fond %s":                                      ": the terminal has a %s background",
	"clair":                                                            "light",
	"sombre":                                                           "dark",
	"réglage %s":                                                       "setting %s",
	"Fond %s détecté : enregistré pour la statusline (agentline theme background pour changer).": "%s background detected: saved for the statusline (agentline theme background to change it).",
	"⚠ fond %s détecté, mais pas enregistré : %s":                                                "⚠ %s background detected, but not saved: %s",
	"⚠ le terminal a un fond %s : agentline theme background %s\n":                               "⚠ the terminal has a %s background: agentline theme background %s\n",
	"fond inconnu : %s (dark, light ou auto)":                                                    "unknown background: %s (dark, light or auto)",
	"✓ fond : détection automatique (%s, %s)\n":                                                  "✓ background: automatic detection (%s, %s)\n",
	"✓ fond → %s\n": "✓ background → %s\n",
	"⚠ AGENTLINE_BACKGROUND=%s l'emporte tant qu'elle est définie": "⚠ AGENTLINE_BACKGROUND=%s wins while it is set",
}
