package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
)

// enAnglais : le test se déroule en anglais (les autres tests du paquet restent en français).
func enAnglais(t *testing.T) {
	t.Helper()
	avant := i18n.Choisir(i18n.EN)
	t.Cleanup(func() { i18n.Choisir(avant) })
}

// TestReferencesAnglaises : la statusline en anglais (tests/golden/en/), sur les cas où la langue se voit
// (état de PR, quotas, plafond, coût), en version longue et courte, et sur quelques thèmes. Générées par
// -maj, puis relues : 7d, $4.98, approved, in review, limit ~, jours en anglais.
func TestReferencesAnglaises(t *testing.T) {
	enAnglais(t)
	dossier := fixtures(t)
	config := t.TempDir()
	en := filepath.Join(racineDepot, "tests", "golden", "en")
	for _, c := range []string{"complet", "tendu", "api", "plafond"} {
		j := casJSON(t, c, dossier)
		for _, nm := range []string{"riche", "ssh"} {
			comparer(t, filepath.Join(en, c+"."+nm+".ans"), Render(j, env(t, modes[nm], config)))
		}
	}
	for _, th := range []string{"nord", "dracula", "gruvbox"} {
		m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": th}}
		comparer(t, filepath.Join(en, "themes", th+".complet.ans"), Render(casJSON(t, "complet", dossier), env(t, m, config)))
	}
	// sur fond clair : un thème clair, et un thème sombre corrigé
	for _, th := range []string{"github-light", "tokyo"} {
		m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": th}, couleurs: ansi.Profil{Fond: ansi.Clair}}
		comparer(t, filepath.Join(en, "clair", th+".tendu.ans"), Render(casJSON(t, "tendu", dossier), env(t, m, config)))
	}
}

// TestLangueDuRendu : en anglais, aucun texte français ; un libellé ou un format réglé l'emporte sur la langue.
func TestLangueDuRendu(t *testing.T) {
	enAnglais(t)
	dossier := fixtures(t)
	config := t.TempDir()
	var tout strings.Builder
	for _, c := range []string{"complet", "tendu", "api", "plafond", "minimal", "hors-git"} {
		for _, nm := range []string{"riche", "ssh"} {
			tout.WriteString(Render(casJSON(t, c, dossier), env(t, modes[nm], config)))
		}
	}
	s := tout.String()
	for _, x := range []string{"7j", "plafond", "approuvée", "en revue", "à corriger", "brouillon", ",98 $"} {
		if strings.Contains(s, x) {
			t.Errorf("%q dans la statusline anglaise", x)
		}
	}
	for _, x := range []string{"7d", "$"} {
		if !strings.Contains(s, x) {
			t.Errorf("%q absent de la statusline anglaise", x)
		}
	}
	st := style.Defaut()
	for k, v := range map[string]string{"7d.label": "7j", "cost.format": "fr", "cost.when": "always"} {
		if err := st.Definir(k, v); err != nil {
			t.Fatal(err)
		}
	}
	e := env(t, modes["riche"], config)
	e.Style = st
	r := Render(casJSON(t, "api", dossier), e)
	if !strings.Contains(r, ",") || !strings.Contains(r, " $") {
		t.Errorf("cost.format=fr en anglais : %q", r)
	}
	if r := Render(casJSON(t, "complet", dossier), e); !strings.Contains(r, "7j") {
		t.Errorf("7d.label=7j en anglais : %q", r)
	}
	st = style.Defaut()
	st.Definir("quota.reset", "always") // la remise à zéro du quota de 7 jours, à plus d'un jour : son jour
	e.Style = st
	if r := Render(casJSON(t, "plafond", dossier), e); !strings.Contains(r, "Sun 08:00") {
		t.Errorf("jour de la remise à zéro en anglais : %q", r)
	}
}
