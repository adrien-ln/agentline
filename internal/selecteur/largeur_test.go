package selecteur

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/theme"
)

// TestLargeurCellules : la largeur est comptée en colonnes du terminal, pas en caractères.
func TestLargeurCellules(t *testing.T) {
	for s, n := range map[string]int{"abc": 3, "日本": 4, "😀": 2, gras + "Thème" + z: 5, "é": 1, "╭─╮▝▜█": 6} {
		if l := largeur(s); l != n {
			t.Errorf("largeur(%q) = %d, attendu %d", s, l, n)
		}
	}
	if c := couper("日本語", 3); largeur(c) != 2 {
		t.Errorf("couper ne doit pas couper un caractère large en deux : %q (%d colonnes)", c, largeur(c))
	}
	for _, l := range replier([]string{"[p]", "日本語のブランチ名がとても長い"}, "  ", 9) {
		if largeur(l) > 9 {
			t.Errorf("replier : %q fait %d colonnes", l, largeur(l))
		}
	}
}

// TestTitreLarge : un thème au titre CJK ou emoji ne décale pas la liste ni le cadre de la fenêtre.
func TestTitreLarge(t *testing.T) {
	m, _ := ouvrir(t, t.TempDir(), nil, func(s theme.Sources) {
		b, _ := s.Fichier("tokyo")
		os.MkdirAll(filepath.Join(s.Config, "themes"), 0o755)
		os.WriteFile(filepath.Join(s.Config, "themes", "zz-cjk.theme"), []byte(strings.Replace(string(b), "# Tokyo Night", "# 東京の夜 🌃", 1)), 0o644)
	})
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	appuyer(m, "fin")
	if m.titres[m.i] != "東京の夜 🌃" {
		t.Fatalf("thème de test absent : %q", m.titres[m.i])
	}
	e := ecran(m)
	if w := lignesFenetre(e); len(w) != 1 {
		t.Errorf("bordures décalées : %v", w)
	}
	// la colonne du cadre de la fenêtre (« │ ») est la même sur la ligne du thème large et sur les autres
	colonnes := map[int]bool{}
	vu := false
	for _, l := range strings.Split(e, "\n") {
		if i := strings.Index(l, "│"); i >= 0 {
			colonnes[largeur(l[:i])] = true
			vu = vu || strings.Contains(l, "東京")
		}
	}
	if !vu || len(colonnes) != 1 {
		t.Errorf("cadre de la fenêtre décalé par le titre large (colonnes %v, ligne trouvée %v) :\n%s", colonnes, vu, e)
	}
}
