package theme

import (
	"testing"

	"github.com/adrien-ln/agentline"
)

// TestResoudre : toute la chaîne, l'étape active et celles qu'elle masque ; Choisir en est le début.
func TestResoudre(t *testing.T) {
	s := Sources{Config: t.TempDir(), Integre: agentline.Themes}
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	resoudre := func() []Etape { return s.Resoudre("x", "main", getenv) }
	verifier := func(quoi, nom, source string) []Etape {
		t.Helper()
		e := resoudre()
		if len(e) != len(Ordre) {
			t.Fatalf("%s : %d étapes, attendu une par source (%d)", quoi, len(e), len(Ordre))
		}
		actives := 0
		for k, x := range e {
			if x.Source != Ordre[k] {
				t.Errorf("%s : étape %d = %s, attendu %s", quoi, k, x.Source, Ordre[k])
			}
			if x.Active {
				actives++
			}
		}
		a := Active(e)
		if actives != 1 || a.Nom != nom || a.Source != source {
			t.Errorf("%s : %d étape(s) active(s), %s (%s), attendu %s (%s)", quoi, actives, a.Nom, a.Source, nom, source)
		}
		if c := s.ChoisirSurBranche("x", "main", getenv); c.Nom != a.Nom || c.Source != a.Source || c.Palette != a.Palette {
			t.Errorf("%s : Choisir donne %s (%s), Resoudre %s (%s)", quoi, c.Nom, c.Source, a.Nom, a.Source)
		}
		return e
	}

	e := verifier("rien de réglé", "tokyo", SourceIntegre)
	if e[Rang(SourceBranche)].Cle != "main" || e[Rang(SourceProjet)].Cle != s.ProjetID("x") {
		t.Errorf("la branche et le projet doivent être connus même sans réglage : %+v", e)
	}
	if _, ok := MasquePar(e, SourceDefaut); ok {
		t.Error("rien de réglé : un défaut ne doit pas être masqué")
	}

	s.EcrireDefaut("dracula")
	env["AGENTLINE_PALETTE"] = "nord"
	e = verifier("défaut", "dracula", SourceDefaut)
	if p := e[Rang(SourcePalette)]; p.Nom != "nord" || !p.Existe || p.Active {
		t.Errorf("AGENTLINE_PALETTE doit être vue, et masquée : %+v", p)
	}

	s.Associer(s.ProjetID("x"), "monokai")
	verifier("projet", "monokai", SourceProjet)
	s.RegleBranche("*", "solarized")
	e = verifier("branche", "solarized", SourceBranche)
	if b := e[Rang(SourceBranche)]; b.Motif != "*" || b.Libelle() != "branche main (règle *)" {
		t.Errorf("règle de branche : %+v, %q", b, b.Libelle())
	}
	if m, ok := MasquePar(e, SourceProjet); !ok || m.Source != SourceBranche {
		t.Errorf("le projet doit être masqué par la branche : %+v", m)
	}
	if _, ok := MasquePar(e, SourceBranche); ok {
		t.Error("une règle de branche ne doit pas être masquée sans AGENTLINE_THEME")
	}

	env["AGENTLINE_THEME"] = "nexiste-pas" // un thème inconnu est ignoré, sans masquer les autres
	e = verifier("AGENTLINE_THEME inconnu", "solarized", SourceBranche)
	if f := e[Rang(SourceForce)]; f.Nom != "nexiste-pas" || f.Existe {
		t.Errorf("AGENTLINE_THEME inconnu : %+v", f)
	}
	env["AGENTLINE_THEME"] = "kanagawa"
	e = verifier("AGENTLINE_THEME", "kanagawa", SourceForce)
	for _, src := range Ordre[1:] {
		if m, ok := MasquePar(e, src); !ok || m.Source != SourceForce {
			t.Errorf("AGENTLINE_THEME doit masquer %s", src)
		}
	}

	if e := (Sources{}).Resoudre("", "", func(string) string { return "" }); Active(e).Palette != Secours() {
		t.Error("sans aucun thème, la palette de secours doit l'emporter")
	}
}
