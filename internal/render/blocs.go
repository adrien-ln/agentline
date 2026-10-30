package render

import (
	"regexp"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
)

// Segments en blocs de couleur (sep.style : arrow, round, slant) : chaque segment sur le fond de la couleur
// de son premier item (sep.colors, sinon couleursBlocs ; d100 en alerte), le texte en une couleur, sombre
// ou claire selon le meilleur contraste ; un glyphe Powerline passe d'un fond au suivant.

// formesBlocs : le glyphe qui ferme un segment, et celui qui ouvre la ligne (arrondi seulement).
var formesBlocs = map[string][2]string{
	"arrow": {"", ""},
	"round": {"", ""},
	"slant": {"", ""},
}

// couleursBlocs : le rôle du fond de chaque item en tête de segment, par défaut.
var couleursBlocs = map[string]string{
	"model": "model", "effort": "model", "fast": "d50", "folder": "folder", "git": "branch", "worktree": "branch",
	"pr": "i3", "agent": "i4", "style": "i5", "session": "muted", "ctx": "i1", "5h": "i2", "7d": "i3", "cost": "d0",
	"machine": "muted",
}

// sgr : les séquences de couleur et de graisse ; le texte d'un bloc les perd (les liens OSC 8 restent).
var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// blocs : la statusline se dessine-t-elle en blocs ? Il faut la forme, une Nerd Font et des couleurs.
func (r *rendu) blocs() bool {
	_, ok := formesBlocs[r.o.forme]
	return ok && r.o.jeu == "nerd" && r.riche && r.c.Niveau != ansi.SansCouleur
}

// couleurRole : la couleur « rrggbb » d'un rôle du thème (style.Roles) ou d'un code.
func (r *rendu) couleurRole(c string) string {
	p := r.p
	switch c {
	case "text":
		return p.Texte
	case "muted":
		return p.Discret
	case "sep":
		return p.Sep
	case "dim":
		return p.Eteint
	case "model":
		return p.Modele
	case "model_end":
		return p.FinDegrade()
	case "folder":
		return p.Dossier
	case "branch":
		return p.Lieu
	case "d0":
		return p.D0
	case "d50":
		return p.D50
	case "d75":
		return p.D75
	case "d100":
		return p.D100
	case "i1", "i2", "i3", "i4", "i5":
		return p.Intensite[c[1]-'1']
	}
	return c
}

// fondBloc : la couleur du fond d'un segment, d'après son premier item.
func (r *rendu) fondBloc(premier string, alerte bool) string {
	switch {
	case alerte:
		return r.couleurRole("d100")
	case r.o.couleursBlocs[premier] != "":
		return r.couleurRole(r.o.couleursBlocs[premier])
	}
	return r.couleurRole(couleursBlocs[premier])
}

// ecritSur : la couleur du texte sur ce fond, la plus contrastée entre un fond sombre et un fond clair.
func ecritSur(fond string) string {
	if ansi.Contraste(fond, ansi.FondSombre) >= ansi.Contraste(fond, ansi.FondClair) {
		return ansi.FondSombre
	}
	return ansi.FondClair
}

// afficherBlocs : la ligne en blocs, les segments qui tiennent dans la largeur, et la colonne où commence le
// texte de chacun (comme afficher).
func (r *rendu) afficherBlocs(segments []string, infos []infoSegment) (string, []int) {
	largeur := 1000
	if r.env.Columns > 0 {
		largeur = r.env.Columns - 4
	}
	forme := formesBlocs[r.o.forme]
	var textes []string
	vue := len([]rune(forme[1]))
	var debuts []int
	for _, s := range segments {
		t := " " + strings.TrimSpace(sgr.ReplaceAllString(s, "")) + " "
		n := visibles(ansi.Visible(t)) + 1 // et le glyphe qui le ferme
		if len(textes) > 0 && vue+n > largeur {
			break
		}
		debuts = append(debuts, vue+1)
		textes = append(textes, t)
		vue += n
	}
	var b strings.Builder
	fonds := make([]string, len(textes))
	for i := range textes {
		fonds[i] = r.fondBloc(infos[i].premier, infos[i].alerte)
	}
	if forme[1] != "" && len(textes) > 0 {
		b.WriteString(r.c.Fg(fonds[0]) + forme[1] + z)
	}
	for i, t := range textes {
		b.WriteString(r.c.Bg(fonds[i]) + r.c.Fg(ecritSur(fonds[i])) + gras + t + z)
		if i+1 < len(textes) {
			b.WriteString(r.c.Bg(fonds[i+1]))
		}
		b.WriteString(r.c.Fg(fonds[i]) + forme[0] + z)
	}
	return b.String() + "\n", debuts
}
