package ansi

import (
	"fmt"
	"math"
	"sync"
)

// Calculs de couleur, sans dépendance : CIELAB (D65) pour les distances perçues et la teinte, luminance
// relative et contraste au sens de WCAG 2 pour la lisibilité.

// lab : une couleur dans l'espace CIELAB.
type lab struct{ l, a, b float64 }

func lineaire(c int) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func gamma(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

const (
	labE   = 6.0 / 29
	xBlanc = 0.95047
	zBlanc = 1.08883
)

func labF(t float64) float64 {
	if t > labE*labE*labE {
		return math.Cbrt(t)
	}
	return t/(3*labE*labE) + 4.0/29
}

func labFInv(t float64) float64 {
	if t > labE {
		return t * t * t
	}
	return 3 * labE * labE * (t - 4.0/29)
}

// luminance : la luminance relative (Y, de 0 pour le noir à 1 pour le blanc).
func luminance(r, g, b int) float64 {
	return 0.2126729*lineaire(r) + 0.7151522*lineaire(g) + 0.0721750*lineaire(b)
}

func versLab(r, g, b int) lab {
	lr, lg, lb := lineaire(r), lineaire(g), lineaire(b)
	x := (0.4124564*lr + 0.3575761*lg + 0.1804375*lb) / xBlanc
	y := 0.2126729*lr + 0.7151522*lg + 0.0721750*lb
	z := (0.0193339*lr + 0.1191920*lg + 0.9503041*lb) / zBlanc
	fx, fy, fz := labF(x), labF(y), labF(z)
	return lab{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

// depuisLab : la couleur sRGB (composantes 0 à 1, non bornées) d'une couleur CIELAB.
func depuisLab(c lab) (float64, float64, float64) {
	fy := (c.l + 16) / 116
	x, y, z := labFInv(fy+c.a/500)*xBlanc, labFInv(fy), labFInv(fy-c.b/200)*zBlanc
	r := 3.2404542*x - 1.5371385*y - 0.4985314*z
	g := -0.9692660*x + 1.8760108*y + 0.0415560*z
	b := 0.0556434*x - 0.2040259*y + 1.0572252*z
	return gamma(r), gamma(g), gamma(b)
}

func (c lab) chroma() float64 { return math.Hypot(c.a, c.b) }

// teinte : l'angle de teinte dans CIELAB, en degrés (0 à 360).
func (c lab) teinte() float64 {
	h := math.Atan2(c.b, c.a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}

func distance2(p, q lab) float64 {
	dl, da, db := p.l-q.l, p.a-q.a, p.b-q.b
	return dl*dl + da*da + db*db
}

// Teinte : l'angle de teinte CIELAB de « rrggbb », en degrés, et sa chroma (0 : gris).
func Teinte(h string) (angle, chroma float64) {
	c := versLab(RGB(h))
	return c.teinte(), c.chroma()
}

// Contraste : le rapport de contraste WCAG 2 entre deux couleurs « rrggbb » (1 à 21).
func Contraste(h1, h2 string) float64 {
	y1, y2 := luminance(RGB(h1)), luminance(RGB(h2))
	if y1 < y2 {
		y1, y2 = y2, y1
	}
	return (y1 + 0.05) / (y2 + 0.05)
}

// EstClaire : vrai si la couleur « rrggbb » est un fond clair (clarté CIELAB au-dessus de 50).
func EstClaire(h string) bool { return versLab(RGB(h)).l > 50 }

func hex(r, g, b int) string { return fmt.Sprintf("%02x%02x%02x", r, g, b) }

func borner(v float64) int { return int(math.Round(math.Max(0, math.Min(1, v)) * 255)) }

// Ajuster : la couleur de même teinte que h dont le contraste avec le fond est compris entre mini et
// maxi (maxi 0 : pas de plafond) ; h telle quelle si elle y est déjà. Seule la clarté CIELAB change (la
// luminance ne dépend que d'elle) ; la chroma n'est réduite que si la couleur sort du gamut sRGB. Le
// résultat est plus sombre que le fond si le fond est clair, plus clair sinon.
func Ajuster(h, fond string, mini, maxi float64) string {
	cle := cleAjuster{h, fond, mini, maxi}
	if r, ok := cacheAjuster.Load(cle); ok {
		return r.(string)
	}
	r := ajuster(h, fond, mini, maxi)
	cacheAjuster.Store(cle, r)
	return r
}

type cleAjuster struct {
	h, fond    string
	mini, maxi float64
}

var cacheAjuster sync.Map // cleAjuster -> string : le sélecteur redemande sans cesse les mêmes couleurs

func ajuster(h, fond string, mini, maxi float64) string {
	c := Contraste(h, fond)
	if c >= mini && (maxi == 0 || c <= maxi) {
		return h
	}
	cible := mini
	if c > mini {
		cible = maxi
	}
	yf := luminance(RGB(fond))
	clair := EstClaire(fond)
	ycible := func(k float64) float64 {
		if clair {
			return math.Max(0, (yf+0.05)/k-0.05)
		}
		return math.Min(1, k*(yf+0.05)-0.05)
	}
	o := versLab(RGB(h))
	l := 116*labF(ycible(cible)) - 16
	pas := 0.5 // le résultat arrondi à 8 bits doit tenir la borne : on pousse la clarté s'il le faut
	if !clair {
		pas = -0.5
	}
	if cible == maxi {
		pas = -pas
	}
	var r string
	for k := 0; k < 40; k++ {
		r = dansGamut(lab{l, o.a, o.b})
		c := Contraste(r, fond)
		if c >= mini && (maxi == 0 || c <= maxi) {
			return r
		}
		l -= pas
	}
	return r
}

// dansGamut : la couleur à cette clarté et cette teinte, chroma réduite (dichotomie) jusqu'à tenir dans sRGB.
func dansGamut(c lab) string {
	c.l = math.Max(0, math.Min(100, c.l))
	ok := func(k float64) (float64, float64, float64, bool) {
		r, g, b := depuisLab(lab{c.l, c.a * k, c.b * k})
		const e = 1e-9
		return r, g, b, r >= -e && r <= 1+e && g >= -e && g <= 1+e && b >= -e && b <= 1+e
	}
	if r, g, b, dedans := ok(1); dedans {
		return hex(borner(r), borner(g), borner(b))
	}
	bas, haut := 0.0, 1.0
	for i := 0; i < 24; i++ {
		m := (bas + haut) / 2
		if _, _, _, dedans := ok(m); dedans {
			bas = m
		} else {
			haut = m
		}
	}
	r, g, b, _ := ok(bas)
	return hex(borner(r), borner(g), borner(b))
}

// --- Palette xterm-256 ------------------------------------------------------------------------------

// niveaux6 : les six valeurs de chaque composante du cube 6×6×6 de xterm (index 16 à 231).
var niveaux6 = [6]int{0, 95, 135, 175, 215, 255}

// rgb256 : les composantes de la couleur n de la palette xterm (16 à 255 : cube et gris, standard).
func rgb256(n int) (int, int, int) {
	if n >= 232 {
		v := 8 + 10*(n-232)
		return v, v, v
	}
	n -= 16
	return niveaux6[n/36], niveaux6[n/6%6], niveaux6[n%6]
}

var (
	labs256      [240]lab
	labs256Prets sync.Once
	cache256     sync.Map // uint32 rrggbb -> int
)

// Vers256 : l'index (16 à 255) de la couleur de la palette xterm-256 la plus proche, au sens de la
// distance CIELAB (ΔE 1976). Les 16 premières couleurs sont exclues : chaque terminal les règle à sa
// façon, alors que le cube et les gris ont les mêmes valeurs partout.
func Vers256(r, g, b int) int {
	cle := uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	if n, ok := cache256.Load(cle); ok {
		return n.(int)
	}
	labs256Prets.Do(func() {
		for i := range labs256 {
			labs256[i] = versLab(rgb256(16 + i))
		}
	})
	c := versLab(r, g, b)
	meilleur, d := 16, math.Inf(1)
	for i, x := range labs256 {
		if e := distance2(c, x); e < d {
			meilleur, d = 16+i, e
		}
	}
	cache256.Store(cle, meilleur)
	return meilleur
}

// --- Les 16 couleurs ANSI ---------------------------------------------------------------------------

// teinteHSV : la teinte au sens HSV (0 à 360), celle où sont définies les couleurs primaires d'ANSI.
func teinteHSV(r, g, b int) float64 {
	mx, mn := max(r, g, b), min(r, g, b)
	d := float64(mx - mn)
	if d == 0 {
		return 0
	}
	var h float64
	switch mx {
	case r:
		h = 60 * float64(g-b) / d
	case g:
		h = 120 + 60*float64(b-r)/d
	default:
		h = 240 + 60*float64(r-g)/d
	}
	if h < 0 {
		h += 360
	}
	return h
}

// Vers16 : la couleur ANSI de base (0 à 15) qui représente le mieux r, g, b. Les valeurs réelles de ces
// 16 couleurs dépendent du terminal (c'est son thème) : une distance aux valeurs d'un terminal précis
// enverrait beaucoup de couleurs pastel vers le gris. On choisit donc par familles : les couleurs peu
// saturées (chroma CIELAB sous 30 : les blancs bleutés des thèmes sombres restent des blancs) vont aux
// gris selon leur clarté, les autres à la teinte primaire la plus proche (teinte HSV : rouge, jaune,
// vert, cyan, bleu, magenta ; l'orange va au jaune sombre, comme dans la plupart des palettes), en
// version vive si la couleur est claire. Sur fond clair, le seuil de la version vive monte : le jaune, le
// cyan et le vert vifs y sont souvent illisibles.
func Vers16(r, g, b int, fondClair bool) int {
	c := versLab(r, g, b)
	if c.chroma() < 30 {
		s := [3]float64{25, 55, 78} // noir, gris sombre, gris clair, blanc
		if fondClair {
			s = [3]float64{45, 70, 88} // le texte, sombre, y devient du noir
		}
		switch {
		case c.l < s[0]:
			return 0
		case c.l < s[1]:
			return 8
		case c.l < s[2]:
			return 7
		}
		return 15
	}
	var base int
	switch h := teinteHSV(r, g, b); {
	case h < 15 || h >= 330:
		base = 1 // rouge, rose vif
	case h < 40:
		return 3 // orange : jaune sombre, jamais vif (il resterait distinct du jaune)
	case h < 75:
		base = 3
	case h < 165:
		base = 2
	case h < 210:
		base = 6
	case h < 255:
		base = 4
	default:
		base = 5 // violet, magenta
	}
	seuil := 50.0
	if fondClair {
		seuil = 85
	}
	if c.l >= seuil {
		return base + 8
	}
	return base
}
