package selecteur

import (
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/theme"
)

// ouvrirAvec : le sélecteur avec ce profil de couleurs (niveau et fond).
func ouvrirAvec(t *testing.T, dossier string, p ansi.Profil) *Modele {
	t.Helper()
	s := theme.Sources{Config: t.TempDir(), Integre: agentline.Themes, Windows: runtime.GOOS == "windows"}
	getenv := func(string) string { return "" }
	e := render.Env{Now: 1800000000, Loc: time.UTC, TempDir: t.TempDir(), Themes: s, Getenv: getenv, Couleurs: p}
	m := Nouveau(s, e, filepath.ToSlash(dossier), getenv)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 34})
	m.t = 9000 // une scène en cours : diff, outils, réflexion
	return m
}

// TestNiveauxDansLeSelecteur : à chaque niveau, l'écran entier (logo, pastilles, Pip, scènes, statusline)
// n'a que des couleurs de ce niveau ; le texte visible ne change pas.
func TestNiveauxDansLeSelecteur(t *testing.T) {
	vingtQuatre := regexp.MustCompile(`\x1b\[[0-9;]*[34]8;2;`)
	deuxCentCinquanteSix := regexp.MustCompile(`\x1b\[[0-9;]*[34]8;5;`)
	couleur := regexp.MustCompile(`\x1b\[[0-9;]*(3[0-9]|4[0-9]|9[0-7]|10[0-7])(;[0-9;]*)?m`)
	d := t.TempDir()
	ref := ecran(ouvrirAvec(t, d, ansi.Profil{}))
	for _, n := range []ansi.Niveau{ansi.Couleurs256, ansi.Couleurs16, ansi.SansCouleur} {
		v := ouvrirAvec(t, d, ansi.Profil{Niveau: n}).View()
		if vingtQuatre.MatchString(v) {
			t.Errorf("%v : une couleur 24 bits reste", n)
		}
		if n != ansi.Couleurs256 && deuxCentCinquanteSix.MatchString(v) {
			t.Errorf("%v : une couleur 256 reste", n)
		}
		if n == ansi.SansCouleur && couleur.MatchString(v) {
			t.Errorf("sans couleur : une couleur reste")
		}
		if n == ansi.Couleurs256 && !deuxCentCinquanteSix.MatchString(v) {
			t.Errorf("256 : aucune couleur 256")
		}
		if e := couleurs.ReplaceAllString(v, ""); e != ref {
			a, b := strings.Split(ref, "\n"), strings.Split(e, "\n")
			for k := range min(len(a), len(b)) {
				if a[k] != b[k] {
					t.Errorf("%v : le texte visible a changé, ligne %d :\n%q\n%q", n, k, a[k], b[k])
					break
				}
			}
		}
	}
	if v := ouvrirAvec(t, d, ansi.Profil{}).View(); !strings.Contains(v, "38;2;") {
		t.Error("24 bits : plus aucune couleur 24 bits")
	}
}

// TestFondClairDansLeSelecteur : sur fond clair, la mini-fenêtre est peinte sur le fond clair de référence,
// sans le texte blanc des scènes, avec les couleurs du thème corrigées ; le texte visible ne change pas.
func TestFondClairDansLeSelecteur(t *testing.T) {
	d := t.TempDir()
	sombre := ouvrirAvec(t, d, ansi.Profil{})
	m := ouvrirAvec(t, d, ansi.Profil{Fond: ansi.Clair})
	v := m.View()
	if !strings.Contains(v, ansi.Bg(ansi.FondClair)) {
		t.Error("la mini-fenêtre n'est pas sur fond clair")
	}
	if strings.Contains(sombre.View(), ansi.Bg(ansi.FondClair)) {
		t.Error("fond sombre : rien n'est peint")
	}
	for _, l := range m.fenetre("tokyo", 70, 16, "") {
		if strings.Contains(l, cTexte) {
			t.Errorf("texte blanc des scènes sur fond clair : %q", l)
		}
	}
	p, _ := m.sources.Charger("tokyo")
	if m.palette("tokyo") != p.PourFond(true) {
		t.Error("la palette du sélecteur n'est pas corrigée pour le fond clair")
	}
	if couleurs.ReplaceAllString(v, "") != ecran(sombre) {
		t.Error("le texte visible a changé")
	}
}

// TestFiltreClairs : le filtre « clairs » réunit les thèmes faits pour un fond clair, et seulement eux.
func TestFiltreClairs(t *testing.T) {
	m := ouvrirAvec(t, t.TempDir(), ansi.Profil{})
	n := 0
	for k, nom := range m.noms {
		p, _ := m.sources.Charger(nom)
		if p.Clair != m.aLeTag(k, filtreClairs) {
			t.Errorf("%s : clair %v, filtre %v", nom, p.Clair, m.aLeTag(k, filtreClairs))
		}
		if p.Clair {
			n++
		}
	}
	a := false
	for _, f := range m.filtres {
		a = a || f == filtreClairs
	}
	if a != (n > 0) {
		t.Errorf("filtre clairs proposé : %v, thèmes clairs : %d", a, n)
	}
}
