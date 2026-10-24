package theme

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
)

// roles : chaque couleur de la palette, ses bornes de contraste.
func roles(p Palette) map[string]struct {
	c string
	b [2]float64
} {
	r := map[string]struct {
		c string
		b [2]float64
	}{
		"TEXTE": {p.Texte, bornesTexte}, "DISCRET": {p.Discret, bornesDiscret}, "SEP": {p.Sep, bornesSep}, "ETEINT": {p.Eteint, bornesEteint},
		"MODELE": {p.Modele, bornesAccent}, "DOSSIER": {p.Dossier, bornesAccent}, "LIEU": {p.Lieu, bornesAccent},
		"D0": {p.D0, bornesAccent}, "D50": {p.D50, bornesAccent}, "D75": {p.D75, bornesAccent}, "D100": {p.D100, bornesAccent},
	}
	for i, c := range p.Intensite {
		r["INTENSITE"+string(rune('0'+i))] = struct {
			c string
			b [2]float64
		}{c, bornesAccent}
	}
	return r
}

// TestPourFond : chaque thème, posé sur l'autre fond, atteint les contrastes de chaque rôle, garde ses
// teintes, et ne change pas sur son propre fond.
func TestPourFond(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	for _, n := range s.Noms() {
		p, _ := s.Charger(n)
		if q := p.PourFond(p.Clair); q != p {
			t.Errorf("%s : changé sur son propre fond", n)
		}
		autre, ref := !p.Clair, ansi.FondClair
		if p.Clair {
			ref = ansi.FondSombre
		}
		q := p.PourFond(autre)
		if q.Clair != autre {
			t.Errorf("%s : fond de la palette adaptée", n)
		}
		avant := roles(p)
		for nom, x := range roles(q) {
			c := ansi.Contraste(x.c, ref)
			if c < x.b[0]-1e-9 || x.b[1] > 0 && c > x.b[1]+1e-9 {
				t.Errorf("%s, %s : %s, contraste %.2f hors de %v", n, nom, x.c, c, x.b)
			}
			h0, c0 := ansi.Teinte(avant[nom].c)
			h1, c1 := ansi.Teinte(x.c)
			if d := math.Abs(h0 - h1); c0 > 25 && c1 > 10 && math.Min(d, 360-d) > 6 {
				t.Errorf("%s, %s : teinte %.0f° devenue %.0f° (%s -> %s)", n, nom, h0, h1, avant[nom].c, x.c)
			}
		}
		if r := q.PourFond(p.Clair); r.Couleurs != p.Couleurs {
			t.Errorf("%s : l'aller-retour ne rend pas les couleurs d'origine", n)
		}
	}
}

// TestPrioriteDuFond : AGENTLINE_BACKGROUND > réglage > COLORFGBG > sombre ; auto et les valeurs inconnues.
func TestPrioriteDuFond(t *testing.T) {
	s := Sources{Config: t.TempDir()}
	type att struct {
		f      ansi.Fond
		source string
	}
	verifier := func(etape string, env map[string]string, a att) {
		t.Helper()
		if f, src := s.Fond(func(k string) string { return env[k] }); f != a.f || src != a.source {
			t.Errorf("%s, %v : %v (%s), attendu %v (%s)", etape, env, f, src, a.f, a.source)
		}
	}
	verifier("rien", nil, att{ansi.Sombre, FondDefaut})
	verifier("COLORFGBG", map[string]string{"COLORFGBG": "0;15"}, att{ansi.Clair, FondCOLORFGBG})
	verifier("COLORFGBG illisible", map[string]string{"COLORFGBG": "0;default"}, att{ansi.Sombre, FondDefaut})
	if err := s.EcrireFond(ansi.Clair, false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(s.Config, "fond")); string(b) != "light\n" {
		t.Errorf("fichier fond : %q", b)
	}
	verifier("réglage", map[string]string{"COLORFGBG": "15;0"}, att{ansi.Clair, FondReglage})
	verifier("variable", map[string]string{"AGENTLINE_BACKGROUND": "dark", "COLORFGBG": "0;15"}, att{ansi.Sombre, FondVariable})
	verifier("variable inconnue", map[string]string{"AGENTLINE_BACKGROUND": "gris"}, att{ansi.Clair, FondReglage})
	verifier("variable auto", map[string]string{"AGENTLINE_BACKGROUND": "auto", "COLORFGBG": "15;0"}, att{ansi.Sombre, FondCOLORFGBG})
	if err := s.EcrireFond(ansi.Sombre, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Config, "fond")); err == nil {
		t.Error("auto : le réglage doit disparaître")
	}
	if err := s.EcrireFond(ansi.Sombre, true); err != nil {
		t.Errorf("auto sans réglage : %v", err)
	}
	verifier("réglage retiré", nil, att{ansi.Sombre, FondDefaut})
}

// TestVariantesDeFond : « # fond: clair » et les couleurs suffixées remplacent la correction automatique.
func TestVariantesDeFond(t *testing.T) {
	p := Lire([]byte("# Test\n# tags: bleu\n# fond: clair\nTEXTE=111111 DISCRET=555555 TEXTE_SOMBRE=eeeeee TEXTE_CLAIR=222222\n" +
		"MODELE=0000aa D0=006600\nINTENSITE=(000011 000022 000033 000044 000055)\nINTENSITE_SOMBRE=(aaaaaa bbbbbb cccccc dddddd eeeeee)\n"))
	if !p.Clair || p.Texte != "111111" || p.Autre.Texte != "eeeeee" || p.Autre.Intensite[4] != "eeeeee" {
		t.Fatalf("lecture : %+v", p)
	}
	q := p.PourFond(false)
	if q.Texte != "eeeeee" || q.Intensite != [5]string{"aaaaaa", "bbbbbb", "cccccc", "dddddd", "eeeeee"} {
		t.Errorf("variantes non prises : %+v", q.Couleurs)
	}
	if q.Discret == "555555" || ansi.Contraste(q.Discret, ansi.FondSombre) < bornesDiscret[0] {
		t.Errorf("DISCRET sans variante : à corriger, %s", q.Discret)
	}
	if p.PourFond(true) != p {
		t.Error("sur fond clair, un thème clair ne change pas")
	}
	sombre := Lire([]byte("TEXTE=dddddd TEXTE_CLAIR=333333 TEXTE_SOMBRE=ffffff\n"))
	if sombre.Clair || sombre.Autre.Texte != "333333" || sombre.PourFond(true).Texte != "333333" {
		t.Errorf("thème sombre : %+v", sombre)
	}
}
