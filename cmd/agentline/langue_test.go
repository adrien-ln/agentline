package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
)

// enAnglais : le test se déroule en anglais (les autres tests restent en français).
func enAnglais(t *testing.T) {
	t.Helper()
	avant := i18n.Choisir(i18n.EN)
	t.Cleanup(func() { i18n.Choisir(avant) })
}

// pasDeFrancais : ni typographie française (« », espace avant « : »), ni lettres accentuées.
func pasDeFrancais(t *testing.T, quoi, s string) {
	t.Helper()
	for _, x := range []string{"«", "»", " : ", " ; ", "é", "è", "à", "ê"} {
		if strings.Contains(s, x) {
			i := strings.Index(s, x)
			t.Errorf("%s : %q en anglais, près de %q", quoi, x, s[max(i-40, 0):min(i+40, len(s))])
		}
	}
}

// TestAnciensNoms : les noms français des options et des sous-commandes sont encore acceptés, pas affichés.
func TestAnciensNoms(t *testing.T) {
	var vues map[string]bool
	c := commande{nom: "install", options: optionsInstallation(), faire: func(a *appel) int { vues = a.opts; return 0 }}
	var sortie, erreurs bytes.Buffer
	for args, attendu := range map[[2]string][2]string{
		{"--sur-place", "--sans-police"}: {"--in-place", "--no-font"},
		{"--in-place", "--no-font"}:      {"--in-place", "--no-font"},
	} {
		if code := lancer(c, "agentline", args[:], &sortie, &erreurs); code != 0 || !vues[attendu[0]] || !vues[attendu[1]] || len(vues) != 2 {
			t.Errorf("%v : code %d, options %v (%s)", args, code, vues, erreurs.String())
		}
	}
	for _, sous := range []string{"theme", "style"} {
		for _, aide := range []string{"help", "aide"} {
			sortie.Reset()
			if code := principale([]string{sous, aide}, &sortie, &erreurs); code != 0 || !strings.Contains(sortie.String(), "agentline "+sous+" help") ||
				strings.Contains(sortie.String(), sous+" aide") {
				t.Errorf("agentline %s %s : code %d\n%s", sous, aide, code, sortie.String())
			}
		}
	}
}

// TestCouleursEnAnglais : fond et niveau de couleurs (theme background, doctor) en anglais.
func TestCouleursEnAnglais(t *testing.T) {
	enAnglais(t)
	configVide(t)
	t.Setenv("AGENTLINE_BACKGROUND", "")
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"theme", "background", "light"}, &sortie, &erreurs); code != 0 || sortie.String() != "✓ background → light\n" {
		t.Errorf("theme background light : %d %q %q", code, sortie.String(), erreurs.String())
	}
	sortie.Reset()
	if code := principale([]string{"theme", "background", "gris"}, &sortie, &erreurs); code != codeUsage ||
		!strings.Contains(erreurs.String(), "unknown background: gris (dark, light or auto)") {
		t.Errorf("fond inconnu : %d %q", code, erreurs.String())
	}
	f := verifierFond(sources(), getenv, func() (ansi.Fond, bool) { return ansi.Sombre, true })
	c := verifierCouleurs(func(k string) string {
		return map[string]string{"AGENTLINE_COLORS": "zut", "TERM": "xterm-256color"}[k]
	}, false)
	texte := f.Sujet + " " + f.Detail + " " + f.Correction + " " + c.Sujet + " " + c.Detail + " " + c.Correction
	for _, x := range []string{"Background", "light (setting ", "the terminal has a dark background", "Colors", "AGENTLINE_COLORS=zut unknown"} {
		if !strings.Contains(texte, x) {
			t.Errorf("%q absent : %q", x, texte)
		}
	}
	pasDeFrancais(t, "doctor", texte)
}

func TestAideAnglaise(t *testing.T) {
	enAnglais(t)
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"help"}, &sortie, &erreurs); code != 0 {
		t.Fatalf("code %d", code)
	}
	if !strings.HasPrefix(sortie.String(), "agentline: the Claude Code statusline and its themes.\n\nCommands:\n") {
		t.Errorf("début de l'aide : %q", sortie.String()[:80])
	}
	pasDeFrancais(t, "agentline help", sortie.String())
	for _, args := range [][]string{{"install", "--help"}, {"theme", "set", "--help"}, {"style", "keys"}} {
		sortie.Reset()
		principale(args, &sortie, &erreurs)
		pasDeFrancais(t, strings.Join(args, " "), sortie.String())
	}
	erreurs.Reset()
	if code := principale([]string{"themes"}, &sortie, &erreurs); code != codeUsage ||
		erreurs.String() != "agentline: unknown command: themes (see \"agentline help\")\n" {
		t.Errorf("commande inconnue : %d %q", code, erreurs.String())
	}
}
