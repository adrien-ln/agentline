package theme

import (
	"math"
	"reflect"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
)

// TestEtiquettes : les couleurs franches prennent leur étiquette, les grises « gris » ; un thème garde ses
// deux étiquettes les plus fréquentes.
func TestEtiquettes(t *testing.T) {
	for h, e := range map[string]string{"ff0000": "rouge", "ff8800": "orange", "ffee00": "jaune", "22aa22": "vert",
		"00bcd4": "cyan", "2060ff": "bleu", "9b30d9": "violet", "ff4fa0": "rose", "8a8a8a": "gris"} {
		if g := EtiquetteDe(h); g != e {
			t.Errorf("%s : %s, attendu %s", h, g, e)
		}
	}
	if e := EtiquettesAuto(Secours()); !reflect.DeepEqual(e, []string{"bleu", "violet"}) && !reflect.DeepEqual(e, []string{"violet", "bleu"}) {
		t.Errorf("Tokyo Night : %v", e)
	}
	for _, e := range EtiquettesAuto(Harmoniser("e05050", false)) {
		if !tagConnu(e) {
			t.Errorf("étiquette hors des filtres du sélecteur : %s", e)
		}
	}
}

func tagConnu(e string) bool {
	for _, p := range prototypes {
		if p.tag == e {
			return true
		}
	}
	return e == "gris"
}

// TestSeuils : les seuils et l'intensité prennent la clarté de l'accent (bornée), dans l'ordre vert, jaune,
// orange, rouge.
func TestSeuils(t *testing.T) {
	p := Seuils(Secours(), "bb9af7")
	for k, h := range []string{p.D0, p.D50, p.D75, p.D100} {
		if e := EtiquetteDe(h); e != []string{"vert", "jaune", "orange", "rouge"}[k] {
			t.Errorf("seuil %d : %s (%s)", k, h, e)
		}
	}
	l0, _, _ := ansi.LCh(p.D0)
	l1, _, _ := ansi.LCh(p.D100)
	if math.Abs(l0-l1) > 6 {
		t.Errorf("même clarté attendue : %.0f et %.0f", l0, l1)
	}
}

// TestHarmoniser : un thème complet, lisible sur son fond, sombre comme clair.
func TestHarmoniser(t *testing.T) {
	for _, clair := range []bool{false, true} {
		p := Harmoniser("3fa7d6", clair)
		fond := ansi.FondSombre
		if clair {
			fond = ansi.FondClair
		}
		if p.Clair != clair || p.Texte == "" || p.ModeleFin == "" || p.Intensite[4] == "" || ansi.Contraste(p.Texte, fond) < 4.5 {
			t.Errorf("clair=%v : %+v (contraste du texte %.1f)", clair, p, ansi.Contraste(p.Texte, fond))
		}
	}
}
