package theme

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
)

var (
	ligneCouleurs  = regexp.MustCompile(`^([A-Z0-9_]+=[0-9a-f]{6}( |$))+$`)
	ligneIntensite = regexp.MustCompile(`^INTENSITE(_CLAIR|_SOMBRE)?=\(([0-9a-f]{6} ){4}[0-9a-f]{6}\)$`)
	// « populaire » et les couleurs dominantes seulement (pas de tons) : ce sont les filtres du sélecteur
	tagsConnus = map[string]bool{"populaire": true, "rouge": true, "orange": true, "jaune": true, "vert": true,
		"cyan": true, "bleu": true, "violet": true, "rose": true, "gris": true}
)

// TestFormatThemes : chaque thème a le format strict attendu, et toutes ses couleurs.
func TestFormatThemes(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	noms := s.Noms()
	if len(noms) < 40 {
		t.Fatalf("%d thèmes seulement", len(noms))
	}
	for _, n := range noms {
		b, _ := s.Fichier(n)
		l := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if !strings.HasPrefix(l[0], "# ") || len(l[0]) < 3 {
			t.Errorf("%s : la première ligne doit être « # Nom affiché »", n)
		}
		debut := 1
		if strings.HasPrefix(l[1], "# tags: ") {
			for _, g := range strings.Fields(strings.ReplaceAll(strings.TrimPrefix(l[1], "# tags: "), ",", " ")) {
				if !tagsConnus[g] {
					t.Errorf("%s : tag inconnu « %s »", n, g)
				}
			}
			debut = 2
		}
		if debut < len(l) && strings.HasPrefix(l[debut], "# fond:") {
			if l[debut] != "# fond: clair" {
				t.Errorf("%s : seule « # fond: clair » est permise (sombre : sans la ligne)", n)
			}
			debut++
		}
		for _, x := range l[debut:] {
			if !ligneCouleurs.MatchString(x) && !ligneIntensite.MatchString(x) {
				t.Errorf("%s : ligne non autorisée : %s", n, x)
			}
		}
		p, ok := s.Charger(n)
		if !ok {
			t.Fatalf("%s : chargement impossible", n)
		}
		for nom, c := range map[string]string{"TEXTE": p.Texte, "DISCRET": p.Discret, "SEP": p.Sep, "ETEINT": p.Eteint,
			"MODELE": p.Modele, "DOSSIER": p.Dossier, "LIEU": p.Lieu, "D0": p.D0, "D50": p.D50, "D75": p.D75, "D100": p.D100, "INTENSITE[4]": p.Intensite[4]} {
			if c == "" {
				t.Errorf("%s : %s manquante", n, nom)
			}
		}
	}
}

func TestNomsInvalidesRefuses(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	for _, n := range []string{"../theme", "Tokyo", "", "-a", "a/b"} {
		if _, ok := s.Fichier(n); ok {
			t.Errorf("nom accepté : %q", n)
		}
	}
	if c := s.Choisir("", func(k string) string { return map[string]string{"AGENTLINE_PALETTE": "nexiste-pas"}[k] }); c.Nom != "tokyo" || c.Source != "intégré" {
		t.Errorf("AGENTLINE_PALETTE inconnue : %s (%s)", c.Nom, c.Source)
	}
	if c := (Sources{}).Choisir("", func(string) string { return "" }); c.Palette != Secours() {
		t.Error("sans aucun thème : la palette de secours doit être utilisée")
	}
}

func TestRemotes(t *testing.T) {
	s := Sources{}
	for _, u := range []string{"git@gitlab.com:Example-Org/My-Repo.git", "https://gitlab.com/example-org/my-repo",
		"https://moi@gitlab.com/example-org/my-repo.git/", "ssh://git@gitlab.com:2222/example-org/my-repo.git"} {
		if id := s.RemoteNormaliser(u); id != "gitlab.com/example-org/my-repo" {
			t.Errorf("%s -> %s", u, id)
		}
	}
	w := Sources{Windows: true}
	if a, b := w.CheminNormaliser("/c/Users/Moi/Projet/"), w.CheminNormaliser(`C:\Users\moi\projet`); a != b || a != "c:/users/moi/projet" {
		t.Errorf("chemins Windows : %q et %q", a, b)
	}
}

func git(t *testing.T, d string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	c.Dir = d
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v : %v\n%s", args, err, out)
	}
}

func TestProjetID(t *testing.T) {
	b := t.TempDir()
	d := filepath.Join(b, "depot")
	os.MkdirAll(filepath.Join(d, "sous", "dossier"), 0o755)
	git(t, d, "init", "-q")
	os.WriteFile(filepath.Join(d, "a"), []byte("a"), 0o644)
	git(t, d, "add", "a")
	git(t, d, "commit", "-qm", "a")
	s := Sources{Windows: runtime.GOOS == "windows"}
	if id := s.ProjetID(filepath.ToSlash(d)); id != s.CheminNormaliser(filepath.ToSlash(d)) {
		t.Errorf("sans remote : %s", id)
	}
	git(t, d, "remote", "add", "origin", "git@gitlab.com:Test/Projet.git")
	for nom, dossier := range map[string]string{"racine": d, "sous-dossier": filepath.Join(d, "sous", "dossier")} {
		if id := s.ProjetID(filepath.ToSlash(dossier)); id != "gitlab.com/test/projet" {
			t.Errorf("%s : %s", nom, id)
		}
	}
	wt := filepath.Join(b, "wt")
	git(t, d, "worktree", "add", "-q", wt)
	if id := s.ProjetID(filepath.ToSlash(wt)); id != "gitlab.com/test/projet" {
		t.Errorf("worktree : %s", id)
	}
}

func TestReglages(t *testing.T) {
	s := Sources{Config: t.TempDir(), Integre: agentline.Themes}
	rien := func(string) string { return "" }
	if c := s.Choisir("x", rien); c.Nom != "tokyo" || c.Source != "intégré" {
		t.Errorf("rien de réglé : %s (%s)", c.Nom, c.Source)
	}
	s.EcrireDefaut("dracula")
	if c := s.Choisir("x", rien); c.Nom != "dracula" || c.Source != "défaut" {
		t.Errorf("défaut : %s (%s)", c.Nom, c.Source)
	}
	id := s.ProjetID("x")
	s.Associer(id, "nord")
	s.Associer(id, "monokai") // remplace
	if a := s.Associations(); len(a) != 1 || a[0][1] != "monokai" {
		t.Errorf("associations : %v", a)
	}
	if c := s.Choisir("x", rien); c.Nom != "monokai" || c.Source != "projet" {
		t.Errorf("projet > défaut : %s (%s)", c.Nom, c.Source)
	}
	if c := s.Choisir("x", func(k string) string { return map[string]string{"AGENTLINE_THEME": "solarized"}[k] }); c.Nom != "solarized" {
		t.Errorf("AGENTLINE_THEME doit tout forcer : %s", c.Nom)
	}
	s.Associer(id, "supprime")
	if c := s.Choisir("x", rien); c.Nom != "dracula" {
		t.Errorf("thème de projet disparu -> défaut : %s", c.Nom)
	}
	s.Associer(id, "")
	if len(s.Associations()) != 0 {
		t.Error("l'association doit être retirée")
	}
}

func TestBranches(t *testing.T) {
	s := Sources{Config: t.TempDir(), Integre: agentline.Themes}
	b := t.TempDir()
	d := filepath.Join(b, "depot")
	os.MkdirAll(d, 0o755)
	git(t, d, "init", "-q")
	os.WriteFile(filepath.Join(d, "a"), []byte("a"), 0o644)
	git(t, d, "add", "a")
	git(t, d, "commit", "-qm", "a")
	git(t, d, "checkout", "-q", "-b", "feat/x")
	dossier := filepath.ToSlash(d)
	if br := BrancheDe(dossier); br != "feat/x" {
		t.Fatalf("branche : %q", br)
	}
	wt := filepath.Join(b, "wt")
	git(t, d, "worktree", "add", "-q", "-b", "release/2", wt)
	if br := BrancheDe(filepath.ToSlash(wt)); br != "release/2" {
		t.Errorf("worktree : %q", br)
	}
	rien := func(string) string { return "" }
	s.Associer(s.ProjetID(dossier), "nord")
	s.RegleBranche("main", "monokai")
	if c := s.Choisir(dossier, rien); c.Nom != "nord" || c.Source != "projet" {
		t.Errorf("aucune règle pour feat/x : %s (%s)", c.Nom, c.Source)
	}
	s.RegleBranche("*", "dracula")
	s.RegleBranche("feat/*", "solarized")
	if c := s.Choisir(dossier, rien); c.Nom != "solarized" || c.Source != "branche" {
		t.Errorf("branche > projet, motif le plus long : %s (%s)", c.Nom, c.Source)
	}
	s.RegleBranche("feat/x", "kanagawa")
	if c := s.Choisir(dossier, rien); c.Nom != "kanagawa" {
		t.Errorf("le nom exact passe avant les motifs : %s", c.Nom)
	}
	if c := s.Choisir(dossier, func(k string) string { return map[string]string{"AGENTLINE_THEME": "tokyo"}[k] }); c.Nom != "tokyo" {
		t.Errorf("AGENTLINE_THEME passe avant la branche : %s", c.Nom)
	}
	if c := s.ChoisirSurBranche(dossier, "release/2", rien); c.Nom != "dracula" {
		t.Errorf("« * » doit couvrir toutes les branches, « / » compris : %s", c.Nom)
	}
	for m, br := range map[string]string{"feat/*": "feat/a/b", "v?.?": "v1.2", "a.b": "a.b"} {
		if !Correspond(m, br) {
			t.Errorf("%s doit couvrir %s", m, br)
		}
	}
	for m, br := range map[string]string{"feat/*": "feature", "a.b": "axb", "main": "mainline", "v?": "v12"} {
		if Correspond(m, br) {
			t.Errorf("%s ne doit pas couvrir %s", m, br)
		}
	}
	s.RegleBranche("feat/x", "")
	s.RegleBranche("feat/*", "")
	s.RegleBranche("*", "")
	if r := s.ReglesBranches(); len(r) != 1 || r[0] != [2]string{"main", "monokai"} {
		t.Errorf("règles après retraits : %v", r)
	}
	for m, ok := range map[string]bool{"main": true, "release/*": true, " ": false, "": false, "a\tb": false} {
		if MotifValide(m) != ok {
			t.Errorf("MotifValide(%q) : %v", m, !ok)
		}
	}
}

// TestEcrituresAtomiques : le thème par défaut et le fond sont écrits d'un coup (aucun fichier temporaire
// ne reste, le contenu est complet).
func TestEcrituresAtomiques(t *testing.T) {
	s := Sources{Config: t.TempDir()}
	if err := s.EcrireDefaut("nord"); err != nil {
		t.Fatal(err)
	}
	if err := s.EcrireFond(ansi.Clair, false); err != nil {
		t.Fatal(err)
	}
	e, _ := os.ReadDir(s.Config)
	var noms []string
	for _, x := range e {
		noms = append(noms, x.Name())
	}
	if strings.Join(noms, " ") != "defaut fond" {
		t.Errorf("fichiers : %v", noms)
	}
	if b, _ := os.ReadFile(filepath.Join(s.Config, "defaut")); string(b) != "nord\n" {
		t.Errorf("defaut : %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(s.Config, "fond")); string(b) != "light\n" {
		t.Errorf("fond : %q", b)
	}
}
