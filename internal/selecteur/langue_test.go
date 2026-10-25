package selecteur

import (
	"os"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// TestMain : les textes attendus par les tests sont en français (TestAnglais vérifie l'anglais).
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	os.Exit(m.Run())
}

// TestAnglais : en-tête, liste, pied, aide, validation et onglet Statusline en anglais.
func TestAnglais(t *testing.T) {
	avant := i18n.Choisir(i18n.EN)
	defer i18n.Choisir(avant)
	m, _ := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "main")
	m.Update(tea.WindowSizeMsg{Width: 150, Height: 42}) // l'onglet Statusline entier, sous le logo
	contient := func(quoi, e string, attendus ...string) {
		t.Helper()
		for _, a := range attendus {
			if !strings.Contains(e, a) {
				t.Errorf("%s : %q absent\n%s", quoi, a, e)
			}
		}
	}
	e := ecran(m)
	contient("liste", e, "in effect: Tokyo Night (built-in)", strconv.Itoa(len(m.noms))+" themes", "◂ all ▸", "Enter choose", "? help", "Claude Code · my-project", "> run the tests")
	appuyer(m, "→")
	contient("filtre", ecran(m), "◂ popular ▸")
	appuyer(m, "?")
	contient("aide", ecran(m), "Themes: list", "Themes: search (/)", "browse", "Esc ?")
	appuyer(m, "échap", "↓", "entrée", "b")
	contient("brouillon", ecran(m), "Not saved: theme", "for the branch main", "s save · u cancel", "scope: the branch main")
	appuyer(m, "échap", "tab")
	m.os.st.Definir("ssh.short", "yes")
	m.os.changer()
	contient("onglet Statusline", ecran(m), " Themes ", "▲ model · Line 1", "Preview · full (v)", "SSH Opus", "s save", "Enter menu")
	appuyer(m, "↑")
	contient("ligne SSH", ecran(m), "SSH line")
	appuyer(m, "g")
	contient("options générales", ecran(m), "Options · general", "icons")
	appuyer(m, "échap", "↓", "x", "+") // tous les items sont dans les lignes : en retirer un d'abord
	contient("ajouter", ecran(m), "Add to line 1")
	appuyer(m, "échap")
	if s := m.Message(); !strings.HasPrefix(s, "Nothing changed: Tokyo Night stays in effect (built-in).") {
		t.Errorf("message de sortie : %q", s)
	}
	m.enRecherche, m.recherche = false, "blue" // la recherche trouve les tags en anglais
	m.onglet = 0
	m.calculerVue()
	if len(m.vue) == 0 {
		t.Error("« blue » ne trouve aucun thème")
	}
}
