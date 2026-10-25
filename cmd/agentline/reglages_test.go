package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/theme"
)

// TestMain : aucun test ne lit ni n'écrit les vrais réglages, même s'il oublie configVide : tout part
// d'un dossier temporaire (AGENTLINE_CONFIG, XDG_CONFIG_HOME, HOME), et les anciens noms sont vidés.
// Les textes attendus sont en français (TestAideAnglaise vérifie l'anglais).
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	d, err := os.MkdirTemp("", "agentline-tests-")
	if err != nil {
		panic(err)
	}
	for _, v := range variables {
		for _, a := range anciensNoms("AGENTLINE_" + v) {
			os.Unsetenv(a)
		}
	}
	detecterFond = func() (ansi.Fond, bool) { return ansi.Sombre, false } // rien n'est écrit au terminal
	os.Unsetenv("COLORFGBG")
	os.Setenv("AGENTLINE_CONFIG", filepath.Join(d, "config"))
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(d, "xdg"))
	os.Setenv("HOME", filepath.Join(d, "home"))
	os.Setenv("USERPROFILE", filepath.Join(d, "home"))
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}

// sansConfigImposee : AGENTLINE_CONFIG absent, XDG_CONFIG_HOME vers un dossier temporaire neuf (et une
// migration pas encore faite) ; renvoie ce dossier.
func sansConfigImposee(t *testing.T) string {
	t.Helper()
	configVide(t)
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	os.Unsetenv("AGENTLINE_CONFIG") // tous deux rendus par les t.Setenv de configVide à la fin du test
	os.Unsetenv("SL_CONFIG")
	ancienne := migration
	migration = &etatMigration{}
	t.Cleanup(func() { migration = ancienne })
	return xdg
}

func TestEmplacementsConfig(t *testing.T) {
	xdg := sansConfigImposee(t)
	if n, a := emplacementsConfig(); n != filepath.Join(xdg, "agentline") || a != filepath.Join(xdg, "statusline") {
		t.Errorf("XDG : %s %s", n, a)
	}
	t.Setenv("SL_CONFIG", "/ancien/imposé")
	if n, a := emplacementsConfig(); n != "/ancien/imposé" || a != "" {
		t.Errorf("SL_CONFIG (ancien nom) : %s %q", n, a)
	}
	t.Setenv("AGENTLINE_CONFIG", "/imposé")
	if n, a := emplacementsConfig(); n != "/imposé" || a != "" {
		t.Errorf("AGENTLINE_CONFIG : %s %q", n, a)
	}
}

// TestMigrationAuPremierAcces : les anciens réglages sont copiés au premier accès (ici « theme current »),
// lus ensuite ; doctor et install le disent ; l'ancien dossier reste intact.
func TestMigrationAuPremierAcces(t *testing.T) {
	xdg := sansConfigImposee(t)
	ancien := filepath.Join(xdg, "statusline")
	os.MkdirAll(filepath.Join(ancien, "themes"), 0o755)
	os.WriteFile(filepath.Join(ancien, "defaut"), []byte("nord\n"), 0o644)

	if o, _, _ := lancerTheme(t, "current", t.TempDir()); o != "nord (défaut)\n" {
		t.Errorf("après migration : %q", o)
	}
	if b, err := os.ReadFile(filepath.Join(xdg, "agentline", "defaut")); err != nil || string(b) != "nord\n" {
		t.Errorf("copie : %q %v", b, err)
	}
	if b, _ := os.ReadFile(filepath.Join(ancien, "defaut")); string(b) != "nord\n" {
		t.Error("l'ancien dossier doit rester intact")
	}
	if v, ok := verifierMigration(); !ok || !strings.Contains(v.Detail, "copiés de "+ancien) {
		t.Errorf("doctor : %+v", v)
	}

	// Lancement suivant (nouveau processus) : plus de copie, l'ancien dossier est signalé comme gardé
	migration = &etatMigration{}
	lancerTheme(t, "set", "dracula")
	if b, _ := os.ReadFile(filepath.Join(ancien, "defaut")); string(b) != "nord\n" {
		t.Error("les écritures vont dans le nouveau dossier")
	}
	if v, ok := verifierMigration(); !ok || migration.copie || !strings.Contains(v.Detail, "gardé, plus lu") {
		t.Errorf("doctor, lancement suivant : %+v", v)
	}
}

// TestRenderMigreSilencieusement : l'environnement de la statusline fait la copie, sans erreur possible.
func TestRenderMigreSilencieusement(t *testing.T) {
	xdg := sansConfigImposee(t)
	os.MkdirAll(filepath.Join(xdg, "statusline"), 0o755)
	os.WriteFile(filepath.Join(xdg, "statusline", "defaut"), []byte("nord\n"), 0o644)
	env := environnement() // ce que lit « agentline render »
	if env.Themes.Config != filepath.Join(xdg, "agentline") {
		t.Errorf("dossier : %s", env.Themes.Config)
	}
	if _, err := os.Stat(filepath.Join(xdg, "agentline", "defaut")); err != nil {
		t.Errorf("copie faite par render : %v", err)
	}
}

// TestEmplacementsConfigSansXDG : sans XDG_CONFIG_HOME, les réglages vont dans ~/.config (le dossier
// personnel du système sous Windows, où HOME peut être au format Git Bash).
func TestEmplacementsConfigSansXDG(t *testing.T) {
	sansConfigImposee(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	h := t.TempDir()
	t.Setenv("HOME", h)
	if runtime.GOOS == "windows" {
		h, _ = os.UserHomeDir()
	}
	if n, a := emplacementsConfig(); n != filepath.Join(h, ".config", "agentline") || a != filepath.Join(h, ".config", "statusline") {
		t.Errorf("sans XDG : %s %s", n, a)
	}
}

// TestSignalerFond : un fond détecté différent du réglage est enregistré et annoncé une fois ; rien
// sans détection ni quand il est déjà enregistré.
func TestSignalerFond(t *testing.T) {
	configVide(t)
	t.Setenv("AGENTLINE_BACKGROUND", "")
	s := theme.Sources{Config: t.TempDir()}
	ancien := detecterFond
	t.Cleanup(func() { detecterFond = ancien })
	var sortie bytes.Buffer
	detecterFond = func() (ansi.Fond, bool) { return ansi.Clair, false }
	if signalerFond(s, &sortie); sortie.Len() != 0 {
		t.Errorf("sans détection, rien à dire : %q", sortie.String())
	}
	detecterFond = func() (ansi.Fond, bool) { return ansi.Clair, true }
	signalerFond(s, &sortie)
	if !strings.Contains(sortie.String(), "détecté") {
		t.Errorf("fond détecté : %q", sortie.String())
	}
	if f, ok := s.FondEnregistre(); !ok || f != ansi.Clair {
		t.Errorf("fond enregistré : %v %v", f, ok)
	}
	sortie.Reset()
	if signalerFond(s, &sortie); sortie.Len() != 0 {
		t.Errorf("déjà enregistré : %q", sortie.String())
	}
}
