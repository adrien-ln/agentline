package ansi

import "strings"

// Détection du niveau de couleurs par l'environnement, sans rien demander au terminal : la statusline est
// lancée par Claude Code sans terminal à interroger (son entrée est le JSON de la session), et hérite de
// l'environnement de Claude Code, donc de celui du terminal où il tourne (TERM, COLORTERM, TERM_PROGRAM,
// WT_SESSION…). Par SSH, seul TERM est transmis d'ordinaire (pas COLORTERM).

// terminauxTruecolor : les valeurs de TERM_PROGRAM des terminaux qui affichent les couleurs 24 bits.
var terminauxTruecolor = map[string]bool{
	"iterm.app": true, "wezterm": true, "vscode": true, "ghostty": true, "hyper": true, "tabby": true,
	"rio": true, "warpterminal": true, "zed": true, "cursor": true, "windsurf": true, "mintty": true,
}

// termsTruecolor : les valeurs de TERM (préfixes) des terminaux qui affichent les couleurs 24 bits.
var termsTruecolor = []string{"xterm-kitty", "xterm-ghostty", "alacritty", "foot", "wezterm", "contour", "rio"}

// DetecterNiveau : le niveau de couleurs et la raison du choix (« AGENTLINE_COLORS », « NO_COLOR »,
// « TERM=xterm-256color », « défaut »…), dans cet ordre :
//
//	AGENTLINE_COLORS (truecolor, 256, 16, none)            imposé par l'utilisateur
//	NO_COLOR non vide                                       none (no-color.org)
//	TERM=dumb                                               none
//	COLORTERM=truecolor ou 24bit                            truecolor
//	WT_SESSION (Windows Terminal), TERM_PROGRAM connu       truecolor (iTerm2, WezTerm, VS Code, Ghostty, mintty…)
//	TERM_PROGRAM=Apple_Terminal                             256 (Terminal.app de macOS)
//	TERM : kitty, alacritty, foot… ou « -direct »           truecolor
//	TERM : « 256color »                                     256
//	TERM : linux, vt100, vt220, ansi, cons25                16 (consoles)
//	TERM : autre valeur                                     256 (xterm, screen, tmux : 256 est sûr)
//	rien de tout ça                                         truecolor (le comportement d'avant ; Windows
//	                                                        sans Windows Terminal affiche aussi le 24 bits)
func DetecterNiveau(getenv func(string) string) (Niveau, string) {
	if n, ok := LireNiveau(getenv("AGENTLINE_COLORS")); ok {
		return n, "AGENTLINE_COLORS"
	}
	if getenv("NO_COLOR") != "" {
		return SansCouleur, "NO_COLOR"
	}
	term := strings.ToLower(getenv("TERM"))
	if term == "dumb" {
		return SansCouleur, "TERM=dumb"
	}
	if c := strings.ToLower(getenv("COLORTERM")); c == "truecolor" || c == "24bit" {
		return Truecolor, "COLORTERM=" + c
	}
	if getenv("WT_SESSION") != "" {
		return Truecolor, "WT_SESSION"
	}
	switch tp := getenv("TERM_PROGRAM"); {
	case terminauxTruecolor[strings.ToLower(tp)]:
		return Truecolor, "TERM_PROGRAM=" + tp
	case tp == "Apple_Terminal":
		return Couleurs256, "TERM_PROGRAM=" + tp
	}
	if term == "" {
		return Truecolor, "défaut"
	}
	raison := "TERM=" + term
	for _, p := range termsTruecolor {
		if strings.HasPrefix(term, p) {
			return Truecolor, raison
		}
	}
	switch {
	case strings.HasSuffix(term, "-direct"):
		return Truecolor, raison
	case strings.Contains(term, "256color"):
		return Couleurs256, raison
	case term == "linux" || term == "ansi" || term == "cons25" || strings.HasPrefix(term, "vt"):
		return Couleurs16, raison
	}
	return Couleurs256, raison
}

// NiveauImpose : vrai si le niveau a été choisi par l'utilisateur (AGENTLINE_COLORS ou NO_COLOR) plutôt
// que deviné ; en SSH, seul un niveau imposé remplace les couleurs ANSI fixes de la ligne courte.
func NiveauImpose(raison string) bool { return raison == "AGENTLINE_COLORS" || raison == "NO_COLOR" }
