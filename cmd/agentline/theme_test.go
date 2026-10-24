package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func lancerTheme(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var o, e bytes.Buffer
	code := commandeTheme(args, &o, &e)
	return o.String(), e.String(), code
}

func lancerAgentline(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var o, e bytes.Buffer
	code := principale(args, &o, &e)
	return o.String(), e.String(), code
}

// configVide : des réglages vides, et aucune variable (ni son ancien nom SL_*) de l'environnement réel.
func configVide(t *testing.T) {
	t.Setenv("AGENTLINE_CONFIG", t.TempDir())
	for _, v := range variables {
		if v != "CONFIG" {
			t.Setenv("AGENTLINE_"+v, "")
		}
		for _, a := range anciensNoms("AGENTLINE_" + v) {
			t.Setenv(a, "")
		}
	}
}

func TestSousCommandes(t *testing.T) {
	configVide(t)
	d := t.TempDir()

	if o, _, _ := lancerTheme(t, "current", d); o != "tokyo (intégré)\n" {
		t.Errorf("current : %q", o)
	}
	if _, e, code := lancerTheme(t, "set", "nexiste"); code != codeUsage || !strings.Contains(e, "thème inconnu") {
		t.Errorf("set d'un thème inconnu : code %d, %q", code, e)
	}
	if o, _, code := lancerTheme(t, "set", "dracula"); code != 0 || o != "✓ thème par défaut → Dracula (dracula)\n" {
		t.Errorf("set : code %d, %q", code, o)
	}
	if o, _, _ := lancerTheme(t, "current", d); o != "dracula (défaut)\n" {
		t.Errorf("set puis current : %q", o)
	}
	if o, _, _ := lancerTheme(t, "link", "nord", d); !strings.HasPrefix(o, "✓ projet ") || !strings.HasSuffix(o, " → Nord (nord)\n") {
		t.Errorf("link : %q", o)
	}
	if o, _, _ := lancerTheme(t, "current", d); !strings.HasPrefix(o, "nord (projet ") {
		t.Errorf("link puis current : %q", o)
	}
	if o, _, _ := lancerTheme(t, "links"); !strings.Contains(o, "nord") {
		t.Errorf("links : %q", o)
	}
	if o, _, _ := lancerTheme(t, "unlink", d); !strings.Contains(o, "retirée") {
		t.Errorf("unlink : %q", o)
	}
	if o, _, _ := lancerTheme(t, "current", d); o != "dracula (défaut)\n" {
		t.Errorf("unlink puis current : %q", o)
	}
	o, _, _ := lancerTheme(t, "list")
	if n := strings.Count(o, "\n"); n != len(sources().Noms()) || strings.Count(o, "●") != 1 {
		t.Errorf("list : %d lignes pour %d thèmes, %d thèmes marqués", n, len(sources().Noms()), strings.Count(o, "●"))
	}
	if _, e, code := lancerTheme(t, "inconnue"); code != codeUsage || !strings.Contains(e, "commande inconnue") {
		t.Errorf("commande inconnue : code %d, %q", code, e)
	}
	if o, _, _ := lancerTheme(t, "preview", "nord"); !strings.Contains(o, "ctx") {
		t.Errorf("preview : %q", o)
	}
}

// TestThemeParTitre : le thème se donne par son nom ou son titre, casse indifférente ; sinon une suggestion.
func TestThemeParTitre(t *testing.T) {
	configVide(t)
	for _, saisie := range []string{"Tokyo Night", "tokyo night", "TOKYO"} {
		if o, e, code := lancerTheme(t, "set", saisie); code != 0 || !strings.Contains(o, "(tokyo)") {
			t.Errorf("set %q : code %d, %q %q", saisie, code, o, e)
		}
	}
	if _, e, code := lancerTheme(t, "set", "tokio"); code != codeUsage || !strings.Contains(e, "« tokyo »") {
		t.Errorf("faute de frappe : code %d, suggestion attendue : %q", code, e)
	}
	if _, e, _ := lancerTheme(t, "preview", "zzzzzzzz"); strings.Contains(e, "voulais dire") {
		t.Errorf("rien d'assez proche : pas de suggestion attendue : %q", e)
	}
}

// TestCodesEtAide : 2 pour toute erreur d'usage, l'aide générée par la table des commandes.
func TestCodesEtAide(t *testing.T) {
	configVide(t)
	absent := filepath.Join(t.TempDir(), "n", "existe", "pas")
	for _, c := range [][]string{{"set"}, {"set", "nord", "en-trop"}, {"link", "nord", absent}, {"current", absent},
		{"unlink", absent}, {"current", "--inconnue"}, {"branch", "main"}, {"branch", " ", "nord"}} {
		if _, e, code := lancerTheme(t, c...); code != codeUsage || e == "" {
			t.Errorf("agentline theme %v : code %d, attendu %d, %q", c, code, codeUsage, e)
		}
	}
	if _, e, _ := lancerTheme(t, "link", "nord", absent); !strings.Contains(e, "dossier introuvable") {
		t.Errorf("dossier inexistant : %q", e)
	}
	if _, e, _ := lancerTheme(t, "set"); !strings.Contains(e, "usage : agentline theme set <thème>") {
		t.Errorf("l'erreur d'usage doit rappeler l'usage : %q", e)
	}
	for _, c := range [][]string{{"bidule"}, {"install", "--bidule"}, {"version", "en-trop"}} {
		if _, _, code := lancerAgentline(t, c...); code != codeUsage {
			t.Errorf("agentline %v : code %d, attendu %d", c, code, codeUsage)
		}
	}
	if _, e, code := lancerAgentline(t); code != codeUsage || !strings.Contains(e, "agentline theme link") {
		t.Errorf("agentline seul, hors console : code %d, l'aide doit énumérer les sous-commandes de theme : %q", code, e)
	}
	for _, c := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		o, _, code := lancerAgentline(t, c...)
		if code != 0 {
			t.Errorf("agentline %v : code %d", c, code)
		}
		for _, x := range append(commandesAgentline(), commandesTheme()...) {
			if !x.cachee && !strings.Contains(o, " "+x.nom) {
				t.Errorf("agentline %v : %s absente de l'aide", c, x.nom)
			}
		}
	}
	if o, _, code := lancerAgentline(t, "install", "--help"); code != 0 || !strings.Contains(o, "--in-place") || !strings.Contains(o, "--no-font") || strings.Contains(o, "--sur-place") || strings.Contains(o, "--sans-profil") {
		t.Errorf("agentline install --help : code %d, %q", code, o)
	}
	if o, _, code := lancerTheme(t, "set", "--help"); code != 0 || !strings.HasPrefix(o, "usage : agentline theme set") {
		t.Errorf("agentline theme set --help : code %d, %q", code, o)
	}
	if o, _, _ := lancerTheme(t, "aide"); !strings.Contains(o, "Priorité") {
		t.Errorf("agentline theme aide : %q", o)
	}
}

func depot(t *testing.T, branche string) string {
	t.Helper()
	d := t.TempDir()
	for _, a := range [][]string{{"init", "-q"}, {"checkout", "-q", "-b", branche}} {
		c := exec.Command("git", append([]string{"-c", "init.defaultBranch=main"}, a...)...)
		c.Dir = d
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v : %v\n%s", a, err, out)
		}
	}
	return d
}

func TestSousCommandesBranches(t *testing.T) {
	configVide(t)
	d := depot(t, "release/3")
	if o, _, _ := lancerTheme(t, "branches"); o != "Aucune règle de branche.\n" {
		t.Errorf("branches sans règle : %q", o)
	}
	if _, e, code := lancerTheme(t, "branch", "main", "nexiste"); code == 0 || !strings.Contains(e, "thème inconnu") {
		t.Errorf("branch vers un thème inconnu : code %d, %q", code, e)
	}
	if _, e, code := lancerTheme(t, "branch", "main"); code == 0 || !strings.Contains(e, "usage") {
		t.Errorf("branch sans thème : code %d, %q", code, e)
	}
	if o, _, _ := lancerTheme(t, "branch", "release/*", "solarized"); o != "✓ branche release/* → Solarized (solarized)\n" {
		t.Errorf("branch : %q", o)
	}
	lancerTheme(t, "link", "nord", d)
	if o, _, _ := lancerTheme(t, "current", d); o != "solarized (branche release/3)\n" {
		t.Errorf("branch puis current : %q", o)
	}
	if o, _, _ := lancerTheme(t, "branches"); !strings.Contains(o, "release/*") || !strings.Contains(o, "solarized") {
		t.Errorf("branches : %q", o)
	}
	lancerTheme(t, "unbranch", "release/*")
	if o, _, _ := lancerTheme(t, "current", d); !strings.HasPrefix(o, "nord (projet ") {
		t.Errorf("unbranch puis current : %q", o)
	}
	if o, _, _ := lancerTheme(t, "unbranch", "release/*"); o != "Aucune règle pour release/*\n" {
		t.Errorf("unbranch d'une règle absente : %q", o)
	}
}

// TestCourantPourquoi : current --why montre toute la chaîne, l'étape active et celles qu'elle masque.
func TestCourantPourquoi(t *testing.T) {
	configVide(t)
	d := depot(t, "main")
	lancerTheme(t, "set", "dracula")
	lancerTheme(t, "branch", "main", "solarized")
	t.Setenv("AGENTLINE_PALETTE", "nexiste-pas")
	o, _, code := lancerTheme(t, "current", "--why", d)
	l := strings.Split(strings.TrimRight(o, "\n"), "\n")
	if code != 0 || len(l) != 7 || l[0] != "solarized (branche main)" {
		t.Fatalf("current --why : code %d\n%s", code, o)
	}
	for k, attendu := range map[int][]string{1: {"AGENTLINE_THEME", "—"}, 2: {"●", "branche main", "Solarized"}, 4: {"défaut", "Dracula", "masqué"},
		5: {"AGENTLINE_PALETTE", "introuvable"}, 6: {"intégré", "masqué"}} {
		for _, x := range attendu {
			if !strings.Contains(l[k], x) {
				t.Errorf("ligne %d : %q attendu dans %q", k, x, l[k])
			}
		}
	}
}

// TestAvertissementMasque : un réglage qui ne sera pas visible ici est signalé.
func TestAvertissementMasque(t *testing.T) {
	configVide(t)
	d := depot(t, "main")
	t.Chdir(d)
	lancerTheme(t, "branch", "main", "solarized")
	if o, _, _ := lancerTheme(t, "link", "nord"); !strings.Contains(o, "⚠ masqué ici par la branche main") {
		t.Errorf("link masqué par la branche : %q", o)
	}
	if o, _, _ := lancerTheme(t, "set", "dracula"); !strings.Contains(o, "⚠") {
		t.Errorf("set masqué : %q", o)
	}
	if o, _, _ := lancerTheme(t, "branch", "*", "kanagawa"); !strings.Contains(o, "plus précise") {
		t.Errorf("règle masquée par une règle plus précise : %q", o)
	}
	t.Setenv("AGENTLINE_THEME", "tokyo")
	if o, _, _ := lancerTheme(t, "branch", "feat/*", "nord"); !strings.Contains(o, "AGENTLINE_THEME=tokyo") {
		t.Errorf("règle masquée par AGENTLINE_THEME : %q", o)
	}
	t.Setenv("AGENTLINE_THEME", "")
	lancerTheme(t, "unbranch", "main")
	lancerTheme(t, "unbranch", "*")
	if o, _, _ := lancerTheme(t, "link", "nord"); strings.Contains(o, "⚠") {
		t.Errorf("rien ne masque le projet : %q", o)
	}
}

// TestSansArgument : dans une console, « agentline » seul ouvre le sélecteur ; hors console, l'aide (code 2).
func TestSansArgument(t *testing.T) {
	configVide(t)
	ancienI, ancienO := interactif, ouvrirSelecteur
	t.Cleanup(func() { interactif, ouvrirSelecteur = ancienI, ancienO })
	ouvert := false
	ouvrirSelecteur = func(a *appel) int { ouvert = true; return 0 }

	interactif = func() bool { return true }
	if o, e, code := lancerAgentline(t); code != 0 || !ouvert || o != "" || e != "" {
		t.Errorf("console : code %d, sélecteur ouvert %v, %q %q", code, ouvert, o, e)
	}
	ouvert = false
	interactif = func() bool { return false }
	if _, e, code := lancerAgentline(t); code != codeUsage || ouvert || !strings.Contains(e, "sans commande : le sélecteur") {
		t.Errorf("hors console : code %d, sélecteur ouvert %v, %q", code, ouvert, e)
	}
}
