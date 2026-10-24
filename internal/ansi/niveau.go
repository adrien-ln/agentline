package ansi

import (
	"strconv"
	"strings"
)

// Niveau : ce que le terminal sait afficher en couleurs. La valeur zéro, Truecolor, est le comportement
// historique (séquences 24 bits).
type Niveau int

const (
	Truecolor   Niveau = iota // 24 bits : 38;2;r;g;b
	Couleurs256               // palette xterm : 38;5;n
	Couleurs16                // couleurs ANSI de base : 30-37, 90-97
	SansCouleur               // aucune couleur ; gras, italique, inverse… restent (no-color.org)
)

var nomsNiveaux = map[Niveau]string{Truecolor: "truecolor", Couleurs256: "256", Couleurs16: "16", SansCouleur: "none"}

func (n Niveau) String() string { return nomsNiveaux[n] }

// LireNiveau : le niveau d'après son nom (truecolor ou 24bit, 256, 16, none) ; faux si inconnu.
func LireNiveau(s string) (Niveau, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "truecolor", "24bit", "24-bit":
		return Truecolor, true
	case "256":
		return Couleurs256, true
	case "16":
		return Couleurs16, true
	case "none", "0", "aucune":
		return SansCouleur, true
	}
	return Truecolor, false
}

// Fond : la couleur de fond du terminal, sombre (zéro : le comportement historique) ou clair.
type Fond int

const (
	Sombre Fond = iota
	Clair
)

func (f Fond) String() string {
	if f == Clair {
		return "light"
	}
	return "dark"
}

// Fonds de référence, pour corriger le contraste des couleurs d'un thème prévu pour l'autre fond, et pour
// peindre la mini-fenêtre du sélecteur. Le clair est un gris très clair, entre le blanc et les fonds
// clairs les plus sombres (Tokyo Night Day, e1e2e7) ; le sombre, un gris presque noir.
const (
	FondClair  = "f0f0f0"
	FondSombre = "1a1a1a"
)

// Profil : comment écrire les couleurs pour un terminal donné. Le seul endroit qui produit les séquences de
// couleur (Fg, Bg, SGR) ; la valeur zéro donne les séquences 24 bits d'avant.
type Profil struct {
	Niveau Niveau
	Fond   Fond // ne sert qu'au choix entre couleurs vives et sombres en 16 couleurs
}

// Fg : la séquence qui donne au texte la couleur « rrggbb », au niveau du profil.
func (p Profil) Fg(h string) string {
	if p.Niveau == Truecolor {
		return Fg(h)
	}
	return p.couleur(h, false)
}

// Bg : la séquence qui donne au fond la couleur « rrggbb », au niveau du profil.
func (p Profil) Bg(h string) string {
	if p.Niveau == Truecolor {
		return Bg(h)
	}
	return p.couleur(h, true)
}

func (p Profil) couleur(h string, fond bool) string {
	r, g, b := RGB(h)
	return p.rgb(r, g, b, fond)
}

func (p Profil) rgb(r, g, b int, fond bool) string {
	switch p.Niveau {
	case Couleurs256:
		x := "38;5;"
		if fond {
			x = "48;5;"
		}
		return "\x1b[" + x + strconv.Itoa(Vers256(r, g, b)) + "m"
	case Couleurs16:
		return "\x1b[" + strconv.Itoa(code16(Vers16(r, g, b, p.Fond == Clair), fond)) + "m"
	case SansCouleur:
		return ""
	}
	if fond {
		return BgRGB(r, g, b)
	}
	return FgRGB(r, g, b)
}

// code16 : le paramètre SGR de la couleur ANSI n (0 à 15), pour le texte ou le fond.
func code16(n int, fond bool) int {
	c := 30 + n
	if n >= 8 {
		c = 90 + n - 8
	}
	if fond {
		c += 10
	}
	return c
}

// SGR : la séquence d'attributs ou de couleurs ANSI de base (« ESC [1;96m ») telle quelle ; sans couleur,
// ses paramètres de couleur retirés (rien s'il ne reste rien).
func (p Profil) SGR(seq string) string {
	if p.Niveau != SansCouleur {
		return seq
	}
	return p.Adapter(seq)
}

// Adapter : le texte, ses séquences de couleur 24 bits ramenées au niveau du profil (et, sans couleur,
// toutes ses couleurs retirées). Pour ce qui a été colorié sans profil (le sélecteur, ses scènes) : un
// seul passage, à la fin. Rien ne change au niveau truecolor.
func (p Profil) Adapter(s string) string {
	if p.Niveau == Truecolor || !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			break
		}
		b.WriteString(s[:i])
		s = s[i+2:]
		j := 0
		for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == ';') {
			j++
		}
		if j >= len(s) || s[j] != 'm' { // pas une séquence de couleur (curseur, effacement…) : telle quelle
			b.WriteString("\x1b[")
			continue
		}
		b.WriteString(p.adapterSGR(s[:j]))
		s = s[j+1:]
	}
	b.WriteString(s)
	return b.String()
}

// adapterSGR : une séquence « ESC [ params m », paramètres réécrits.
func (p Profil) adapterSGR(params string) string {
	if params == "" || params == "0" {
		return "\x1b[" + params + "m"
	}
	x := strings.Split(params, ";")
	var garde []string
	var couleurs strings.Builder
	n := func(k int) int {
		if k >= len(x) {
			return 0
		}
		v, _ := strconv.Atoi(x[k])
		return v
	}
	for k := 0; k < len(x); k++ {
		v := n(k)
		switch {
		case (v == 38 || v == 48) && n(k+1) == 2 && k+4 < len(x):
			couleurs.WriteString(p.rgb(n(k+2), n(k+3), n(k+4), v == 48))
			k += 4
		case (v == 38 || v == 48) && n(k+1) == 5 && k+2 < len(x):
			if p.Niveau != SansCouleur {
				garde = append(garde, x[k:k+3]...)
			}
			k += 2
		case v >= 30 && v <= 39, v >= 40 && v <= 49, v >= 90 && v <= 97, v >= 100 && v <= 107:
			if p.Niveau != SansCouleur {
				garde = append(garde, x[k])
			}
		default:
			garde = append(garde, x[k])
		}
	}
	s := ""
	if len(garde) > 0 {
		s = "\x1b[" + strings.Join(garde, ";") + "m"
	}
	return s + couleurs.String()
}
