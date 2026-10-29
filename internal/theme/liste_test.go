package theme

import (
	"slices"
	"testing"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
)

var familles = []string{"populaire", "basique", "création"}

// TestFamilles : chaque thème intégré est d'une famille, et d'une seule.
func TestFamilles(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	for _, n := range s.Noms() {
		_, tags := s.Entete(n)
		k := 0
		for _, g := range tags {
			if slices.Contains(familles, g) {
				k++
			}
		}
		if k != 1 {
			t.Errorf("%s : %d familles dans %v, une attendue", n, k, tags)
		}
	}
}

// TestEtiquettesCoherentes : une étiquette de couleur nomme la teinte d'une des couleurs principales du
// thème (modèle, fin du dégradé, dossier, branche), et un thème en a au moins une.
func TestEtiquettesCoherentes(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	for _, n := range s.Noms() {
		p, _ := s.Charger(n)
		_, tags := s.Entete(n)
		teintes := map[string]bool{}
		for _, h := range []string{p.Modele, p.ModeleFin, p.Dossier, p.Lieu} {
			if h != "" {
				teintes[EtiquetteDe(h)] = true
			}
		}
		couleurs := 0
		for _, g := range tags {
			if slices.Contains(familles, g) {
				continue
			}
			couleurs++
			if !teintes[g] {
				t.Errorf("%s : étiquette « %s », aucune couleur principale de cette teinte (%v)", n, g, teintes)
			}
		}
		if couleurs == 0 {
			t.Errorf("%s : aucune étiquette de couleur", n)
		}
	}
}

// TestPasDeDoublons : deux thèmes du même fond, dont l'un au moins n'est pas « populaire », ne se
// ressemblent pas : l'écart moyen de leurs couleurs principales dépasse un seuil. Les thèmes populaires
// gardent leurs couleurs officielles, même proches entre eux (Tokyo Night et Catppuccin, par exemple).
func TestPasDeDoublons(t *testing.T) {
	const seuil = 15
	s := Sources{Integre: agentline.Themes}
	noms := s.Noms()
	type th struct {
		p         Palette
		populaire bool
	}
	ths := make([]th, len(noms))
	for i, n := range noms {
		p, _ := s.Charger(n)
		_, tags := s.Entete(n)
		ths[i] = th{p, slices.Contains(tags, "populaire")}
	}
	for i := range noms {
		for j := i + 1; j < len(noms); j++ {
			a, b := ths[i], ths[j]
			if a.p.Clair != b.p.Clair || a.populaire && b.populaire {
				continue
			}
			d := (ansi.Ecart(a.p.Modele, b.p.Modele) + ansi.Ecart(a.p.Dossier, b.p.Dossier) + ansi.Ecart(a.p.Lieu, b.p.Lieu)) / 3
			if d < seuil {
				t.Errorf("%s et %s se ressemblent trop (écart moyen %.1f < %d)", noms[i], noms[j], d, seuil)
			}
		}
	}
}

// TestRemplaces : un thème retiré n'existe plus, son remplaçant oui ; le nommer donne le remplaçant, sauf
// si l'utilisateur a un thème de ce nom.
func TestRemplaces(t *testing.T) {
	s := Sources{Integre: agentline.Themes, Config: t.TempDir()}
	noms := s.Noms()
	for ancien, nouveau := range Remplaces {
		if slices.Contains(noms, ancien) {
			t.Errorf("%s : encore intégré, il ne doit plus être un remplacement", ancien)
		}
		if !slices.Contains(noms, nouveau) {
			t.Errorf("%s -> %s : remplaçant absent", ancien, nouveau)
		}
		if n, ok := s.Trouver(ancien); !ok || n != nouveau {
			t.Errorf("Trouver(%s) = %s, %v ; attendu %s", ancien, n, ok, nouveau)
		}
		a, _ := s.Charger(ancien)
		b, _ := s.Charger(nouveau)
		if a != b {
			t.Errorf("Charger(%s) : la palette de %s attendue", ancien, nouveau)
		}
	}
	// un ancien réglage qui nomme un thème retiré : la chaîne de priorité donne le remplaçant
	if err := s.EcrireDefaut("miami"); err != nil {
		t.Fatal(err)
	}
	if c := s.Choisir("", func(string) string { return "" }); c.Nom != "vaporwave" {
		t.Errorf("défaut miami : %s retenu, vaporwave attendu", c.Nom)
	}
	if err := s.EcrireTheme("miami", Format(Secours(), "Mon Miami", nil)); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Trouver("miami"); n != "miami" {
		t.Errorf("thème de l'utilisateur nommé miami : Trouver donne %s", n)
	}
	if p, _ := s.Charger("miami"); p != Secours() {
		t.Error("thème de l'utilisateur nommé miami : sa palette attendue")
	}
}
