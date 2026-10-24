package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/style"
)

func lancerStyle(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var o, e bytes.Buffer
	code := commandeStyle(args, &o, &e)
	return o.String(), e.String(), code
}

func TestStyleCLI(t *testing.T) {
	configVide(t)
	t.Setenv("AGENTLINE_STYLE", "")
	t.Setenv("SSH_CONNECTION", "")
	fichier := filepath.Join(os.Getenv("AGENTLINE_CONFIG"), style.Fichier)

	if o, _, code := lancerStyle(t, "set", "ctx.width", "12"); code != 0 || o != "✓ ctx.width = 12\n" {
		t.Errorf("set : code %d, %q", code, o)
	}
	if o, _, code := lancerStyle(t, "set", "ctx.bar", "-=."); code != 0 || !strings.Contains(o, "-=.") {
		t.Errorf("set d'une valeur qui commence par - : code %d, %q", code, o)
	}
	if o, _, code := lancerStyle(t, "set", "icon.pr", ""); code != 0 || o != "✓ icon.pr = (vide)\n" {
		t.Errorf("set vide : code %d, %q", code, o)
	}
	if o, _, _ := lancerStyle(t, "set", "line2", "ctx", "|", "5h"); !strings.Contains(o, "ctx | 5h") {
		t.Errorf("set en plusieurs arguments : %q", o)
	}
	b, _ := os.ReadFile(fichier)
	if !strings.Contains(string(b), "ctx.width=12\n") || !strings.Contains(string(b), "icon.pr=\n") {
		t.Errorf("fichier : %q", b)
	}
	o, _, _ := lancerStyle(t, "show")
	if !strings.Contains(o, "ctx.width         12  (réglé)") || !strings.Contains(o, "ctx.alert         85  (défaut)") {
		t.Errorf("show :\n%s", o)
	}
	if n := strings.Count(o, "\n"); n != len(style.Cles()) {
		t.Errorf("show : %d lignes pour %d options", n, len(style.Cles()))
	}
	if o, _, _ := lancerStyle(t, "set", "ctx.width", "8"); o != "✓ ctx.width = 8 (défaut)\n" {
		t.Errorf("set au défaut : %q", o)
	}
	if o, _, _ := lancerStyle(t, "unset", "icon.pr"); !strings.Contains(o, "retour au défaut") {
		t.Errorf("unset : %q", o)
	}
	if o, _, _ := lancerStyle(t, "hide", "7d"); !strings.HasPrefix(o, "✓ 7d masqué") {
		t.Errorf("hide : %q", o)
	}
	if o, _, _ := lancerStyle(t, "preview"); strings.Contains(o, "7j") || !strings.Contains(o, "ctx") {
		t.Errorf("preview : %q", o)
	}
	if o, _, _ := lancerStyle(t, "preview", "--ssh"); strings.Count(o, "\n") != 1 {
		t.Errorf("preview --ssh : %q", o)
	}
	if o, _, _ := lancerStyle(t, "unhide", "7d"); !strings.HasPrefix(o, "✓ 7d affiché") {
		t.Errorf("unhide : %q", o)
	}
	t.Setenv("AGENTLINE_STYLE", "defaut")
	if o, _, _ := lancerStyle(t, "set", "ctx.label", "contexte"); !strings.Contains(o, "⚠ AGENTLINE_STYLE=default") {
		t.Errorf("avertissement AGENTLINE_STYLE : %q", o)
	}
	t.Setenv("AGENTLINE_STYLE", "")
	if o, _, _ := lancerStyle(t, "reset"); !strings.HasPrefix(o, "✓ style par défaut rétabli (3 réglages") {
		t.Errorf("reset : %q", o)
	}
	if _, err := os.Stat(fichier); !os.IsNotExist(err) {
		t.Error("reset doit retirer le fichier")
	}
	if o, _, _ := lancerStyle(t, "keys"); strings.Count(o, "\n") != 2*len(style.Cles()) || !strings.Contains(o, "4–20") {
		t.Errorf("keys :\n%s", o)
	}
}

// TestStyleAnciensNoms : les anciens noms français des options, des items et des valeurs sont acceptés,
// et réglés sous leur nom actuel.
func TestStyleAnciensNoms(t *testing.T) {
	configVide(t)
	t.Setenv("SSH_CONNECTION", "")
	for _, c := range [][3]string{
		{"ctx.largeur", "12", "✓ ctx.width = 12\n"},
		{"modele.nom", "court", "✓ model.name = short\n"},
		{"git.amont", "non", "✓ git.upstream = no\n"},
		{"quota.reset", "non", "✓ quota.reset = never\n"},
		{"ligne2", "ctx | 7j cout", "✓ line2 = ctx | 7d cost\n"},
		{"icone.dossier", "D", "✓ icon.folder = D\n"},
	} {
		if o, e, code := lancerStyle(t, "set", c[0], c[1]); code != 0 || o != c[2] {
			t.Errorf("set %s %s : code %d, %q %s", c[0], c[1], code, o, e)
		}
	}
	if o, _, _ := lancerStyle(t, "hide", "cout"); !strings.HasPrefix(o, "✓ cost masqué") {
		t.Errorf("hide cout : %q", o)
	}
	if o, _, _ := lancerStyle(t, "unset", "ctx.largeur"); !strings.Contains(o, "ctx.width") {
		t.Errorf("unset ctx.largeur : %q", o)
	}
}

func TestStyleErreurs(t *testing.T) {
	configVide(t)
	for _, c := range [][]string{{"set"}, {"set", "ctx.width"}, {"set", "ctx.width", "99"}, {"set", "inconnue", "1"},
		{"set", "ctx.bar", "🚀🚀🚀"}, {"unset"}, {"unset", "zut"}, {"hide", "zut"}, {"hide"}, {"preview", "--x"}, {"bidule"}} {
		if _, e, code := lancerStyle(t, c...); code != codeUsage || e == "" {
			t.Errorf("agentline style %v : code %d, %q", c, code, e)
		}
	}
	if _, e, _ := lancerStyle(t, "set", "ctx.width", "99"); !strings.Contains(e, "un nombre de 4 à 20") {
		t.Errorf("message : %q", e)
	}
	if _, e, _ := lancerStyle(t, "set", "ctx.bar", "🚀🚀🚀"); !strings.Contains(e, "2 colonnes") {
		t.Errorf("glyphe large : %q", e)
	}
	if _, e, _ := lancerStyle(t, "set", "ctx.larg", "9"); !strings.Contains(e, "ctx.width") {
		t.Errorf("suggestion : %q", e)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("AGENTLINE_CONFIG"), style.Fichier)); !os.IsNotExist(err) {
		t.Error("une erreur ne doit rien écrire")
	}
	o, _, code := lancerAgentline(t, "help")
	for _, x := range commandesStyle() {
		if code != 0 || !strings.Contains(o, "agentline style "+x.nom) {
			t.Errorf("agentline help : agentline style %s absente", x.nom)
		}
	}
	if o, _, code := lancerStyle(t, "set", "--help"); code != 0 || !strings.HasPrefix(o, "usage : agentline style set") {
		t.Errorf("set --help : %d %q", code, o)
	}
}

// TestLigneCourteOptionnelle : en SSH, la statusline reste complète tant que ssh.short n'est pas à yes.
func TestLigneCourteOptionnelle(t *testing.T) {
	configVide(t)
	t.Setenv("AGENTLINE_STYLE", "")
	t.Setenv("SSH_CONNECTION", "1.2.3.4 5 6.7.8.9 22")
	if environnement().SSH {
		t.Error("par défaut, la statusline complète en SSH")
	}
	if _, e, code := lancerStyle(t, "set", "ssh.short", "yes"); code != 0 {
		t.Fatalf("set ssh.short : %d %q", code, e)
	}
	if !environnement().SSH {
		t.Error("ssh.short=yes : la ligne courte en SSH")
	}
	t.Setenv("SSH_CONNECTION", "")
	if environnement().SSH {
		t.Error("ssh.short=yes hors SSH : la statusline complète")
	}
}
