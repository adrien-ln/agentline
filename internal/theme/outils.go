package theme

import (
	"math"

	"github.com/adrien-ln/agentline/internal/ansi"
)

// Les outils de l'éditeur de thèmes : les étiquettes de couleur calculées, et des couleurs générées à partir
// d'une seule (les seuils et l'intensité accordés à elle, ou tout un thème en harmonie).

// prototypes : une couleur franche par étiquette ; une couleur prend l'étiquette de la teinte CIELAB la plus
// proche (les gris à part).
var prototypes = []struct{ tag, hex string }{
	{"rouge", "e53935"}, {"orange", "fb8c00"}, {"jaune", "fdd835"}, {"vert", "43a047"},
	{"cyan", "00acc1"}, {"bleu", "1e88e5"}, {"violet", "8e24aa"}, {"rose", "ec407a"},
}

const chromaGris = 12 // en dessous, une couleur passe pour grise

// EtiquetteDe : l'étiquette de couleur d'une couleur « rrggbb » (rouge… rose, gris).
func EtiquetteDe(h string) string {
	angle, chroma := ansi.Teinte(h)
	if chroma < chromaGris {
		return "gris"
	}
	meilleure, ecart := "", 361.0
	for _, p := range prototypes {
		a, _ := ansi.Teinte(p.hex)
		d := math.Abs(a - angle)
		d = math.Min(d, 360-d)
		if d < ecart {
			meilleure, ecart = p.tag, d
		}
	}
	return meilleure
}

// EtiquettesAuto : les étiquettes de couleur d'un thème, d'après ses couleurs principales (nom du modèle,
// dossier, branche) : les plus fréquentes, deux au plus, dans l'ordre de ces rôles à égalité.
func EtiquettesAuto(p Palette) []string {
	var ordre []string
	n := map[string]int{}
	for _, h := range []string{p.Modele, p.Dossier, p.Lieu} {
		if h == "" {
			continue
		}
		e := EtiquetteDe(h)
		if n[e] == 0 {
			ordre = append(ordre, e)
		}
		n[e]++
	}
	var r []string
	for len(r) < 2 && len(ordre) > 0 {
		k := 0
		for i, e := range ordre {
			if n[e] > n[ordre[k]] {
				k = i
			}
		}
		r, ordre = append(r, ordre[k]), append(ordre[:k:k], ordre[k+1:]...)
	}
	return r
}

// Teintes des seuils (vert, jaune, orange, rouge) et de l'intensité (du froid au chaud), en degrés CIELAB.
var (
	teintesSeuils    = [4]float64{136, 95, 62, 32}
	teintesIntensite = [5]float64{220, 285, 315, 350, 62}
)

// accord : la clarté et la chroma d'une couleur, ramenées dans des bornes lisibles sur ce fond.
func accord(h string, clair bool) (l, c float64) {
	l, c, _ = ansi.LCh(h)
	if clair {
		l = math.Max(35, math.Min(60, l))
	} else {
		l = math.Max(55, math.Min(85, l))
	}
	return l, math.Max(c, 40)
}

// Seuils : la palette avec les seuils (D0 à D100) et l'intensité accordés à la couleur accent : même clarté
// et même chroma, seules les teintes changent.
func Seuils(p Palette, accent string) Palette {
	l, c := accord(accent, p.Clair)
	for k, x := range []*string{&p.D0, &p.D50, &p.D75, &p.D100} {
		*x = ansi.DepuisLCh(l, c, teintesSeuils[k])
	}
	for k := range p.Intensite {
		p.Intensite[k] = ansi.DepuisLCh(l, c, teintesIntensite[k])
	}
	return p
}

// Harmoniser : tout un thème à partir de la couleur accent, pour un fond clair ou sombre : le nom du modèle
// en accent, le dégradé, le dossier et la branche en teintes voisines, les gris teintés d'elle, puis les
// seuils et l'intensité accordés (Seuils).
func Harmoniser(accent string, clair bool) Palette {
	l, c := accord(accent, clair)
	_, _, t := ansi.LCh(accent)
	teinte := func(d float64) float64 { return math.Mod(t+d+360, 360) }
	gris := func(lSombre, lClair float64) string {
		if clair {
			return ansi.DepuisLCh(lClair, 8, t)
		}
		return ansi.DepuisLCh(lSombre, 8, t)
	}
	p := Palette{Clair: clair}
	p.Texte, p.Discret, p.Sep, p.Eteint = gris(88, 25), gris(62, 45), gris(30, 82), gris(40, 70)
	p.Modele = ansi.DepuisLCh(l, c, t)
	p.ModeleFin = ansi.DepuisLCh(l, c, teinte(40))
	p.Dossier = ansi.DepuisLCh(l, c, teinte(-40))
	p.Lieu = ansi.DepuisLCh(l, c, teinte(80))
	return Seuils(p, accent)
}
