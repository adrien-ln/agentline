package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/style"
)

func TestLangageDe(t *testing.T) {
	d := t.TempDir()
	if _, ok := langageDe(d); ok {
		t.Error("dossier vide : aucun langage attendu")
	}
	for _, f := range []string{"package.json", "tsconfig.json"} {
		if err := os.WriteFile(filepath.Join(d, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if l, _ := langageDe(d); l.icone != "" {
		t.Errorf("tsconfig.json et package.json : %q, TypeScript attendu", l.icone)
	}
}

func TestForgeDe(t *testing.T) {
	for url, voulu := range map[string]string{
		"https://github.com/a/b/pull/3":                     "",
		"https://gitlab.com/test/projet/-/merge_requests/4": "",
		"https://bitbucket.org/a/b/pull-requests/1":         "",
		"https://gitea.example.com/github/b/pulls/2":        "", // « github » dans le chemin, pas dans l'hôte
		"": "",
	} {
		if f, _ := forgeDe(url); f.icone != voulu {
			t.Errorf("%s : %q, attendu %q", url, f.icone, voulu)
		}
	}
}

// TestIconesProjet : folder.icon=language et pr.icon=forge, discrets par défaut, dans leur couleur avec
// project.color=brand ; sans Nerd Font, les icônes habituelles restent.
func TestIconesProjet(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "projet-propre"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "projet-propre", "go.mod"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	rendu := func(regles string) string {
		e := env(t, modes["riche"], t.TempDir())
		e.GitFictif = &GitFictif{Branche: "main"}
		e.Style = style.Lire([]byte(regles))
		return Render(casJSON(t, "complet", filepath.ToSlash(d)), e)
	}
	l := rendu("folder.icon=language\npr.icon=forge")
	v := ansi.Visible(l)
	for _, x := range []string{" projet-propre", " #42"} {
		if !strings.Contains(v, x) {
			t.Errorf("%q absent de %q", x, v)
		}
	}
	if strings.Contains(v, "") || strings.Contains(v, "") {
		t.Errorf("icônes du dossier ou de la PR encore là : %q", v)
	}
	discret := l[strings.LastIndex(l[:strings.Index(l, "")], "\x1b["):strings.Index(l, "")]
	if !strings.Contains(l, discret+"") {
		t.Errorf("langage et forge : même couleur discrète attendue (%q)", discret)
	}
	if b := rendu("folder.icon=language\npr.icon=forge\nproject.color=brand"); !strings.Contains(b, "38;2;0;173;216m") || !strings.Contains(b, "38;2;252;109;38m") {
		t.Errorf("project.color=brand : couleurs de Go et de GitLab attendues\n%q", b)
	}
	if v := ansi.Visible(rendu("folder.icon=language\npr.icon=forge\nicons=text")); !strings.Contains(v, "▸ projet-propre") || !strings.Contains(v, "PR #42") {
		t.Errorf("icons=text : icônes en texte attendues : %q", v)
	}
}
