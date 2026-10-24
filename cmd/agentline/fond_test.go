package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/installation"
	"github.com/adrien-ln/agentline/internal/theme"
)

func TestThemeBackground(t *testing.T) {
	configVide(t)
	config := os.Getenv("AGENTLINE_CONFIG")
	if o, _, code := lancerTheme(t, "background"); code != 0 || o != "dark (par défaut)\n" {
		t.Errorf("sans réglage : %d %q", code, o)
	}
	if o, _, code := lancerTheme(t, "background", "light"); code != 0 || o != "✓ fond → light\n" {
		t.Errorf("light : %d %q", code, o)
	}
	if b, _ := os.ReadFile(filepath.Join(config, "fond")); string(b) != "light\n" {
		t.Errorf("fichier fond : %q", b)
	}
	if o, _, _ := lancerTheme(t, "background"); o != "light (réglage "+config+"/fond)\n" {
		t.Errorf("après light : %q", o)
	}
	if e := environnement(); e.Couleurs.Fond != ansi.Clair {
		t.Error("la statusline doit lire le réglage")
	}
	t.Setenv("AGENTLINE_BACKGROUND", "dark")
	if o, _, _ := lancerTheme(t, "background", "light"); !strings.Contains(o, "⚠ AGENTLINE_BACKGROUND=dark l'emporte") {
		t.Errorf("variable contraire : %q", o)
	}
	if o, _, _ := lancerTheme(t, "background"); o != "dark (AGENTLINE_BACKGROUND)\n" {
		t.Errorf("variable : %q", o)
	}
	t.Setenv("AGENTLINE_BACKGROUND", "")
	t.Setenv("COLORFGBG", "0;15")
	if o, _, code := lancerTheme(t, "background", "auto"); code != 0 || o != "✓ fond : détection automatique (light, COLORFGBG)\n" {
		t.Errorf("auto : %d %q", code, o)
	}
	if _, err := os.Stat(filepath.Join(config, "fond")); err == nil {
		t.Error("auto : le réglage doit disparaître")
	}
	if _, e, code := lancerTheme(t, "background", "gris"); code != codeUsage || !strings.Contains(e, "fond inconnu") {
		t.Errorf("valeur inconnue : %d %q", code, e)
	}
	if o, _, _ := lancerTheme(t, "aide"); !strings.Contains(o, "agentline theme background [dark|light|auto]") {
		t.Errorf("aide : %q", o)
	}
}

// TestFondDetecte : le fond détecté n'est enregistré que sans variable ni réglage, et s'il change quelque chose.
func TestFondDetecte(t *testing.T) {
	configVide(t)
	s := theme.Sources{Config: t.TempDir(), Integre: agentline.Themes}
	if m := enregistrerFondDetecte(s, ansi.Sombre); m != "" {
		t.Errorf("sombre détecté, sombre par défaut : rien à faire (%q)", m)
	}
	if m := enregistrerFondDetecte(s, ansi.Clair); !strings.HasPrefix(m, "Fond clair détecté") {
		t.Errorf("clair détecté : %q", m)
	}
	if f, ok := s.FondEnregistre(); !ok || f != ansi.Clair {
		t.Error("clair détecté : pas enregistré")
	}
	if m := enregistrerFondDetecte(s, ansi.Sombre); m != "" {
		t.Errorf("un réglage existe : la détection ne le change pas (%q)", m)
	}
	s2 := theme.Sources{Config: t.TempDir()}
	t.Setenv("AGENTLINE_BACKGROUND", "dark")
	if m := enregistrerFondDetecte(s2, ansi.Clair); m != "" {
		t.Errorf("variable définie : rien à enregistrer (%q)", m)
	}
}

func TestVerifierFond(t *testing.T) {
	s := theme.Sources{Config: t.TempDir()}
	sans := func() (ansi.Fond, bool) { return ansi.Sombre, false }
	clair := func() (ansi.Fond, bool) { return ansi.Clair, true }
	vide := func(string) string { return "" }
	if v := verifierFond(s, vide, sans); v.Niveau != installation.Bon || v.Detail != "sombre (par défaut)" {
		t.Errorf("%+v", v)
	}
	if v := verifierFond(s, vide, clair); v.Niveau != installation.Avertissement || v.Correction != "agentline theme background light" {
		t.Errorf("%+v", v)
	}
	env := func(k string) string { return map[string]string{"AGENTLINE_BACKGROUND": "dark"}[k] }
	if v := verifierFond(s, env, clair); v.Correction != "AGENTLINE_BACKGROUND=light" || v.Detail != "sombre (AGENTLINE_BACKGROUND) : le terminal a un fond clair" {
		t.Errorf("%+v", v)
	}
}
