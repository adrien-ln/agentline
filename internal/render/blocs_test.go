package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/entree/claude"
	"github.com/adrien-ln/agentline/internal/style"
)

// rendreBlocs : le cas rendu avec ces règles de style, sans retirer les couleurs.
func rendreBlocs(t *testing.T, cas, regles string, niveau ansi.Niveau) string {
	t.Helper()
	e := env(t, modes["riche"], t.TempDir())
	e.GitFictif = &GitFictif{Branche: "main"}
	e.Style = style.Lire([]byte(regles))
	e.Couleurs.Niveau = niveau
	return Render(casJSON(t, cas, "/x"), e)
}

// TestBlocs : chaque forme ferme ses segments par son glyphe, l'arrondi ouvre aussi la ligne ; le texte
// d'un bloc perd ses couleurs et prend le fond de son premier item.
func TestBlocs(t *testing.T) {
	for forme, g := range formesBlocs {
		l := rendreBlocs(t, "complet", "sep.style="+forme, ansi.Truecolor)
		v := ansi.Visible(l)
		if strings.Count(v, g[0]) < 2 || g[1] != "" && !strings.HasPrefix(v, g[1]) {
			t.Errorf("%s : glyphes %q absents de %q", forme, g, v)
		}
		if strings.Contains(v, "│") {
			t.Errorf("%s : le séparateur de sep.line1 ne doit plus paraître : %q", forme, v)
		}
		if !strings.Contains(l, ansi.Bg("bb9af7")) { // le fond du modèle (Tokyo Night)
			t.Errorf("%s : fond du modèle absent : %q", forme, l)
		}
	}
	l := rendreBlocs(t, "complet", "sep.style=arrow\nsep.colors=folder=ff8800 model=d0", ansi.Truecolor)
	if !strings.Contains(l, ansi.Bg("ff8800")) || !strings.Contains(l, ansi.Bg("9ece6a")) {
		t.Errorf("sep.colors : fonds ff8800 et d0 attendus : %q", l)
	}
	// en alerte, le segment prend le rouge du thème
	if l := rendreBlocs(t, "tendu", "sep.style=arrow", ansi.Truecolor); !strings.Contains(l, ansi.Bg("f7768e")) {
		t.Errorf("alerte : fond d100 attendu : %q", l)
	}
}

// TestBlocsRepli : sans Nerd Font ou sans couleurs, les segments restent séparés comme d'habitude.
func TestBlocsRepli(t *testing.T) {
	for _, c := range []struct {
		regles string
		niveau ansi.Niveau
	}{
		{"sep.style=round\nicons=text", ansi.Truecolor},
		{"sep.style=round\nicons=none", ansi.Truecolor},
		{"sep.style=round", ansi.SansCouleur},
	} {
		if v := ansi.Visible(rendreBlocs(t, "complet", c.regles, c.niveau)); !strings.Contains(v, " │ ") || strings.Contains(v, "") {
			t.Errorf("%q, niveau %v : rendu habituel attendu : %q", c.regles, c.niveau, v)
		}
	}
}

// TestBlocsPlaces : les places des items tombent sur leur texte dans la ligne en blocs (le sélecteur
// surligne l'item choisi).
func TestBlocsPlaces(t *testing.T) {
	e := env(t, modes["riche"], t.TempDir())
	e.GitFictif = &GitFictif{Branche: "main"}
	e.Style = style.Lire([]byte("sep.style=round"))
	l, places := RenderPlaces(claude.Lire([]byte(casJSON(t, "complet", "/x"))), e)
	lignes := strings.Split(ansi.Visible(l), "\n")
	for _, p := range places {
		texte := string([]rune(lignes[p.Ligne])[p.Debut:p.Fin])
		if voulu := map[string]string{"model": "Opus 5.5", "folder": "projet-propre", "ctx": "ctx", "5h": "5h"}[p.Item]; voulu != "" && !strings.Contains(texte, voulu) {
			t.Errorf("%s : %q à [%d:%d], %q attendu", p.Item, texte, p.Debut, p.Fin, voulu)
		}
	}
}

// TestCouleursBlocs : chaque item a un fond par défaut, un rôle que sep.colors connaît aussi.
func TestCouleursBlocs(t *testing.T) {
	for _, n := range style.Items {
		if r, ok := couleursBlocs[n]; !ok || !slices.Contains(style.Roles, r) {
			t.Errorf("%s : fond par défaut %q, un rôle de style.Roles attendu", n, r)
		}
	}
}
