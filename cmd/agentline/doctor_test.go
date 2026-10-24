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

func TestReglagesOrphelins(t *testing.T) {
	d := t.TempDir()
	s := theme.Sources{Config: d, Integre: agentline.Themes}
	if v := reglagesOrphelins(s); len(v) != 1 || v[0].Niveau != installation.Bon {
		t.Errorf("sans réglages : %+v", v)
	}
	os.WriteFile(filepath.Join(d, "defaut"), []byte("tokio\n"), 0o644)
	os.WriteFile(filepath.Join(d, "projets"), []byte("/home/x/projet\tdisparu\ngitlab.com/x/y\tnord\ngitlab.com/x/z\tefface\n"), 0o644)
	os.WriteFile(filepath.Join(d, "branches"), []byte("main\tnord\nrelease/*\tvieux\n"), 0o644)
	v := reglagesOrphelins(s)
	var corrections []string
	for _, x := range v {
		if x.Niveau != installation.Probleme {
			t.Errorf("%+v", x)
		}
		corrections = append(corrections, x.Correction)
	}
	attendu := []string{
		"agentline theme set tokyo",
		`agentline theme unlink "/home/x/projet"`,
		"dans un clone de gitlab.com/x/z : agentline theme link tokyo (ou agentline theme unlink)",
		`agentline theme unbranch "release/*"`,
	}
	if strings.Join(corrections, "\n") != strings.Join(attendu, "\n") {
		t.Errorf("corrections :\n%s", strings.Join(corrections, "\n"))
	}
}

func TestDoctorDansLAide(t *testing.T) {
	sortie, _, code := lancerAgentline(t, "help")
	if code != 0 || !strings.Contains(sortie, "agentline doctor") {
		t.Errorf("aide :\n%s", sortie)
	}
	if _, erreurs, code := lancerAgentline(t, "doctor", "x"); code != codeUsage || !strings.Contains(erreurs, "argument en trop") {
		t.Errorf("agentline doctor x : %d %s", code, erreurs)
	}
}

// TestNiveauCouleurs : sur la ligne courte, le niveau imposé > AGENTLINE_SSH_RICH (24 bits) > couleurs
// fixes ; en SSH sans ligne courte (par défaut), la détection habituelle.
func TestNiveauCouleurs(t *testing.T) {
	for _, c := range []struct {
		env    map[string]string
		court  bool
		niveau ansi.Niveau
		impose bool
	}{
		{map[string]string{"TERM": "xterm-256color"}, false, ansi.Couleurs256, false},
		{map[string]string{"SSH_CONNECTION": "x", "TERM": "xterm-256color"}, false, ansi.Couleurs256, false},
		{map[string]string{"SSH_CONNECTION": "x", "TERM": "xterm-256color", "AGENTLINE_SSH_RICH": "1"}, true, ansi.Truecolor, false},
		{map[string]string{"SSH_CONNECTION": "x", "AGENTLINE_SSH_RICH": "1", "AGENTLINE_COLORS": "16"}, true, ansi.Couleurs16, true},
		{map[string]string{"SSH_CONNECTION": "x", "NO_COLOR": "1"}, true, ansi.SansCouleur, true},
		{map[string]string{"SSH_CONNECTION": "x", "TERM": "dumb"}, true, ansi.SansCouleur, false},
	} {
		if n, _, i := niveauCouleurs(func(k string) string { return c.env[k] }, c.court); n != c.niveau || i != c.impose {
			t.Errorf("%v : %v %v", c.env, n, i)
		}
	}
}

func TestVerifierCouleurs(t *testing.T) {
	for _, c := range []struct {
		env    map[string]string
		niveau installation.Niveau
		detail string
	}{
		{nil, installation.Bon, "24 bits (défaut)"},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, installation.Avertissement, "256 couleurs (TERM_PROGRAM=Apple_Terminal) : si le terminal affiche le 24 bits, AGENTLINE_COLORS=truecolor"},
		{map[string]string{"AGENTLINE_COLORS": "16"}, installation.Bon, "16 couleurs (AGENTLINE_COLORS)"},
		{map[string]string{"NO_COLOR": "1"}, installation.Bon, "sans couleur (NO_COLOR)"},
		{map[string]string{"AGENTLINE_COLORS": "mille"}, installation.Avertissement, "AGENTLINE_COLORS=mille inconnue, ignorée : 24 bits (défaut)"},
		{map[string]string{"SSH_CONNECTION": "1 2 3 4", "TERM": "xterm-256color"}, installation.Bon, "couleurs ANSI fixes de la ligne courte (SSH)"},
		{map[string]string{"SSH_CONNECTION": "1 2 3 4", "TERM": "xterm-256color", "AGENTLINE_SSH_RICH": "1"}, installation.Bon, "24 bits (AGENTLINE_SSH_RICH)"},
		{map[string]string{"SSH_CONNECTION": "1 2 3 4", "AGENTLINE_SSH_RICH": "1", "AGENTLINE_COLORS": "256"}, installation.Bon, "256 couleurs (AGENTLINE_COLORS)"},
	} {
		v := verifierCouleurs(func(k string) string { return c.env[k] }, c.env["SSH_CONNECTION"] != "") // la ligne courte demandée
		if v.Niveau != c.niveau || v.Detail != c.detail {
			t.Errorf("%v : %v %q", c.env, v.Niveau, v.Detail)
		}
	}
}
