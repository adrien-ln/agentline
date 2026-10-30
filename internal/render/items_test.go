package render

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
)

func TestNouvelleLigne(t *testing.T) {
	l := nouvelleLigne(" model effort|| folder  git |ctx ", "·")
	if want := [][]string{{"model", "effort"}, {"folder", "git"}, {"ctx"}}; !reflect.DeepEqual(l.segments, want) || l.sep != "·" {
		t.Errorf("obtenu %+v", l)
	}
	defer func() {
		if recover() == nil {
			t.Error("un item inconnu doit faire paniquer")
		}
	}()
	nouvelleLigne("model inconnu", "·")
}

// TestDispositionDefautComplete : la version longue utilise chaque item du registre une fois, sauf ceux
// qu'on ajoute soi-même (machine), absents.
func TestDispositionDefautComplete(t *testing.T) {
	vus := map[string]int{}
	for _, l := range dispositionDefaut.lignes {
		for _, seg := range l.segments {
			for _, n := range seg {
				vus[n]++
			}
		}
	}
	for n, d := range items {
		if voulu := map[bool]int{false: 1, true: 0}[d.ajoute]; vus[n] != voulu {
			t.Errorf("item %s : %d fois dans la disposition par défaut", n, vus[n])
		}
	}
}

// TestBarre : sur 8 cases, la barre est celle de l'ancienne version (formule recopiée) ; sur n cases,
// elle compte des demi-cases.
func TestBarre(t *testing.T) {
	r := preparer(etat.Session{}, env(t, modes["riche"], t.TempDir()))
	ancienne := func(p int) string {
		u := (p*16 + 50) / 100
		var b strings.Builder
		for i := 0; i < 8; i++ {
			if u > i*2 {
				b.WriteString(ansi.Fg(r.degrade((i*100 + 50) / 8)))
				if u >= (i+1)*2 {
					b.WriteString("━")
				} else {
					b.WriteString("╸")
				}
			} else {
				b.WriteString(r.eteint + "─")
			}
		}
		return b.String() + z
	}
	for p := -20; p <= 130; p++ {
		if a, o := ancienne(p), r.barre(p, largeurBarre); a != o {
			t.Errorf("%d %% : %q au lieu de %q", p, o, a)
		}
	}
	for _, c := range []struct {
		p, n  int
		texte string
	}{{50, 4, "━━──"}, {0, 4, "────"}, {100, 12, "━━━━━━━━━━━━"}, {25, 10, "━━╸───────"}, {42, 1, "╸"}} {
		if v := ansi.Visible(r.barre(c.p, c.n)); v != c.texte {
			t.Errorf("%d %% sur %d : %q, attendu %q", c.p, c.n, v, c.texte)
		}
	}
}
