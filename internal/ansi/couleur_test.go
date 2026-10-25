package ansi

import (
	"math"
	"testing"
)

func TestVers256(t *testing.T) {
	for h, n := range map[string]int{
		"000000": 16, "ffffff": 231, "ff0000": 196, "00ff00": 46, "0000ff": 21,
		"5f87af": 67,  // exactement dans le cube (95, 135, 175)
		"808080": 244, // gris 8 + 10×12
		"767676": 243, "e4e4e4": 254, "080808": 232,
		"7aa2f7": 111, // bleu de Tokyo Night : 87afff
	} {
		if v := Vers256(RGB(h)); v != n {
			t.Errorf("Vers256(%s) = %d, attendu %d", h, v, n)
		}
	}
}

func TestVers16(t *testing.T) {
	for h, n := range map[string]int{
		"000000": 0, "ffffff": 15, "808080": 8, "c0c0c0": 7, "3b4261": 8, "c0caf5": 15, "737aa2": 8,
		"ff0000": 9, "800000": 1, "00ff00": 10, "0000ff": 4, "ffff00": 11,
		"7aa2f7": 12, "bb9af7": 13, "ff79c6": 13, "f7768e": 9, "9ece6a": 10, "7dcfff": 14,
		"ff9e64": 3, "e0af68": 3, // l'orange au jaune sombre
	} {
		if v := v16(h, false); v != n {
			t.Errorf("Vers16(%s) = %d, attendu %d", h, v, n)
		}
	}
	// sur fond clair, les couleurs moyennes restent sombres
	for h, n := range map[string]int{"7aa2f7": 4, "9ece6a": 2, "ffffff": 15, "2e7de9": 4, "4a5272": 0, "737aa2": 8, "a4aace": 7} {
		if v := v16(h, true); v != n {
			t.Errorf("Vers16(%s, fond clair) = %d, attendu %d", h, v, n)
		}
	}
}

func v16(h string, clair bool) int { r, g, b := RGB(h); return Vers16(r, g, b, clair) }

func TestContraste(t *testing.T) {
	for _, c := range []struct {
		a, b string
		v    float64
	}{{"000000", "ffffff", 21}, {"ffffff", "000000", 21}, {"777777", "777777", 1}, {"777777", "ffffff", 4.48}} {
		if v := Contraste(c.a, c.b); math.Abs(v-c.v) > 0.01 {
			t.Errorf("Contraste(%s, %s) = %.3f, attendu %.2f", c.a, c.b, v, c.v)
		}
	}
	if !EstClaire(FondClair) || EstClaire(FondSombre) || !EstClaire("e1e2e7") || EstClaire("002b36") {
		t.Error("EstClaire se trompe sur les fonds de référence")
	}
}

func ecartTeinte(a, b float64) float64 {
	d := math.Abs(a - b)
	return math.Min(d, 360-d)
}

// TestAjuster : le contraste demandé est atteint (après l'arrondi à 8 bits), la teinte est conservée, et
// une couleur déjà dans les bornes ne change pas.
func TestAjuster(t *testing.T) {
	couleurs := []string{"c0caf5", "737aa2", "3b4261", "545c7e", "bb9af7", "7aa2f7", "7dcfff", "9ece6a", "e0af68",
		"ff9e64", "f7768e", "ff007c", "f8f8f2", "50fa7b", "f1fa8c", "8be9fd", "ffb86c", "ebdbb2", "fabd2f", "ffffff", "00ff00"}
	for _, fond := range []string{FondClair, FondSombre, "ffffff", "fdf6e3"} {
		for _, b := range [][2]float64{{4.5, 0}, {3, 0}, {1.4, 2}, {1.8, 2.8}} {
			for _, h := range couleurs {
				r := Ajuster(h, fond, b[0], b[1])
				c := Contraste(r, fond)
				if c < b[0]-1e-9 || b[1] > 0 && c > b[1]+1e-9 {
					t.Errorf("Ajuster(%s, %s, %v) = %s : contraste %.2f hors des bornes", h, fond, b, r, c)
				}
				if c0 := Contraste(h, fond); c0 >= b[0] && (b[1] == 0 || c0 <= b[1]) && r != h {
					t.Errorf("Ajuster(%s, %s, %v) = %s : la couleur convenait déjà", h, fond, b, r)
				}
				h0, c0 := Teinte(h)
				h1, c1 := Teinte(r)
				if c0 > 25 && c1 > 10 && ecartTeinte(h0, h1) > 6 {
					t.Errorf("Ajuster(%s, %s, %v) = %s : teinte %.0f° devenue %.0f°", h, fond, b, r, h0, h1)
				}
				if EstClaire(fond) == (luminance(RGB(r)) > luminance(RGB(fond))) {
					t.Errorf("Ajuster(%s, %s, %v) = %s : du mauvais côté du fond", h, fond, b, r)
				}
			}
		}
	}
}

func TestAdapter(t *testing.T) {
	rouge := Fg("ff0000")
	s := "\x1b[1m" + rouge + "X" + Z + " \x1b]8;;https://e.x\aY\x1b]8;;\a \x1b[1;96mZ" + Bg("0000ff") + "\x1b[2K"
	for _, c := range []struct {
		p     Profil
		voulu string
	}{
		{Profil{}, s},
		{Profil{Niveau: Couleurs256}, "\x1b[1m\x1b[38;5;196mX\x1b[0m \x1b]8;;https://e.x\aY\x1b]8;;\a \x1b[1;96mZ\x1b[48;5;21m\x1b[2K"},
		{Profil{Niveau: Couleurs16}, "\x1b[1m\x1b[91mX\x1b[0m \x1b]8;;https://e.x\aY\x1b]8;;\a \x1b[1;96mZ\x1b[44m\x1b[2K"},
		{Profil{Niveau: SansCouleur}, "\x1b[1mX\x1b[0m \x1b]8;;https://e.x\aY\x1b]8;;\a \x1b[1mZ\x1b[2K"},
	} {
		if v := c.p.Adapter(s); v != c.voulu {
			t.Errorf("niveau %v :\nattendu %q\nobtenu  %q", c.p.Niveau, c.voulu, v)
		}
	}
	if v := (Profil{Niveau: Couleurs256}).Adapter("\x1b[1;38;2;255;0;0;4mX"); v != "\x1b[1;4m\x1b[38;5;196mX" {
		t.Errorf("paramètres mêlés : %q", v)
	}
}

func TestProfil(t *testing.T) {
	for _, c := range []struct {
		p      Profil
		fg, bg string
	}{
		{Profil{}, "\x1b[38;2;122;162;247m", "\x1b[48;2;122;162;247m"},
		{Profil{Niveau: Couleurs256}, "\x1b[38;5;111m", "\x1b[48;5;111m"},
		{Profil{Niveau: Couleurs16}, "\x1b[94m", "\x1b[104m"},
		{Profil{Niveau: Couleurs16, Fond: Clair}, "\x1b[34m", "\x1b[44m"},
		{Profil{Niveau: SansCouleur}, "", ""},
	} {
		if fg, bg := c.p.Fg("7aa2f7"), c.p.Bg("7aa2f7"); fg != c.fg || bg != c.bg {
			t.Errorf("%+v : %q %q, attendu %q %q", c.p, fg, bg, c.fg, c.bg)
		}
	}
	if s := (Profil{Niveau: SansCouleur}).SGR("\x1b[1;96m"); s != "\x1b[1m" {
		t.Errorf("SGR sans couleur : %q", s)
	}
	if s := (Profil{Niveau: SansCouleur}).SGR("\x1b[97m"); s != "" {
		t.Errorf("SGR sans couleur : %q", s)
	}
	if s := (Profil{Niveau: Couleurs16}).SGR("\x1b[1;96m"); s != "\x1b[1;96m" {
		t.Errorf("SGR : %q", s)
	}
}

// BenchmarkPaletteClaireSansCache : les 20 couleurs de Tokyo Night corrigées pour le fond clair, sans le
// cache : le coût d'une statusline (un processus par rendu) sur fond clair.
func BenchmarkPaletteClaireSansCache(b *testing.B) {
	p := []string{"c0caf5", "737aa2", "3b4261", "545c7e", "bb9af7", "7aa2f7", "7dcfff", "9ece6a", "e0af68", "ff9e64",
		"f7768e", "7dcfff", "7aa2f7", "bb9af7", "ff007c", "ff9e64"}
	for b.Loop() {
		for _, h := range p {
			ajuster(h, FondClair, 3, 0)
		}
	}
}

func TestLireNiveau(t *testing.T) {
	for s, n := range map[string]Niveau{"truecolor": Truecolor, "24bit": Truecolor, "256": Couleurs256, "16": Couleurs16, "none": SansCouleur, " NONE ": SansCouleur} {
		if v, ok := LireNiveau(s); !ok || v != n {
			t.Errorf("LireNiveau(%q) = %v, %v", s, v, ok)
		}
	}
	for _, s := range []string{"", "8", "true", "couleurs"} {
		if _, ok := LireNiveau(s); ok {
			t.Errorf("LireNiveau(%q) accepté", s)
		}
	}
}

func TestEcart(t *testing.T) {
	if e := Ecart("777777", "777777"); e != 0 {
		t.Errorf("même couleur : %.2f", e)
	}
	if e := Ecart("000000", "ffffff"); math.Abs(e-100) > 0.5 {
		t.Errorf("noir et blanc : %.2f, attendu 100", e)
	}
	// deux teintes de même clarté : contraste faible, écart net
	if Contraste("ff8800", "00b0ff") > 1.5 || Ecart("ff8800", "00b0ff") < 50 {
		t.Error("l'écart doit séparer l'orange du bleu, même à clarté voisine")
	}
}

// TestLCh : une couleur décomposée puis recomposée revient à elle-même (à l'arrondi 8 bits près) ; une
// chroma trop forte est ramenée dans sRGB sans changer la teinte.
func TestLCh(t *testing.T) {
	for _, h := range []string{"bb9af7", "ff9e64", "1a1b26", "ffffff", "000000", "7dcfff", "9ece6a"} {
		l, c, a := LCh(h)
		if r := DepuisLCh(l, c, a); Ecart(r, h) > 1 {
			t.Errorf("%s -> %.1f %.1f %.1f -> %s", h, l, c, a, r)
		}
	}
	l, _, a := LCh("ff9e64")
	r := DepuisLCh(l, 300, a)
	if _, a2 := Teinte(r); math.Abs(a2-a) > 3 {
		t.Errorf("chroma excessive : teinte %0.1f devenue %0.1f (%s)", a, a2, r)
	}
}

// TestSimuler : la vision normale ne change rien ; en deutéranopie, le rouge et le vert se rapprochent
// beaucoup, le gris ne bouge pas.
func TestSimuler(t *testing.T) {
	if Simuler("ff9e64", VisionNormale) != "ff9e64" {
		t.Error("vision normale : inchangé")
	}
	rouge, vert := "e05050", "50a050"
	if a, b := Ecart(rouge, vert), Ecart(Simuler(rouge, Deuteranopie), Simuler(vert, Deuteranopie)); b > a/2 {
		t.Errorf("deutéranopie : écart rouge-vert %.1f -> %.1f, attendu bien plus petit", a, b)
	}
	for _, v := range []Vision{Protanopie, Deuteranopie, Tritanopie} {
		if g := Simuler("808080", v); Ecart(g, "808080") > 2 {
			t.Errorf("vision %d : le gris doit rester gris (%s)", v, g)
		}
	}
}
