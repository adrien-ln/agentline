package render

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Les sorties de référence (tests/golden/) ont été produites par l'ancienne version bash (statusline.sh),
// puis vérifiées identiques au rendu Go sur tous les thèmes avant son retrait. Après un changement de
// rendu VOULU : go test ./internal/render -run 'TestReferences' -maj, puis relis le diff des .ans.
var maj = flag.Bool("maj", false, "réécrire les sorties de référence de tests/golden/")

// Heure figée, fuseau UTC, dépôts de test identiques à ceux qui ont produit les références.
const slNow = 1800000000

var racineDepot = filepath.Join("..", "..")

func git(t *testing.T, dossier string, args ...string) {
	t.Helper()
	base := []string{"-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "init.defaultBranch=main", "-c", "core.autocrlf=false"}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dossier
	if sortie, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v : %v\n%s", args, err, sortie)
	}
}

func ecrire(t *testing.T, f, contenu string) {
	t.Helper()
	if err := os.WriteFile(f, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixtures : « projet-propre » (remote, à jour), « projet-sale » (+1 ●1 ?1, sans remote), « hors-git ».
func fixtures(t *testing.T) string {
	t.Helper()
	b := t.TempDir()
	propre, sale := filepath.Join(b, "projet-propre"), filepath.Join(b, "projet-sale")
	for _, d := range []string{propre, sale, filepath.Join(b, "hors-git")} {
		os.MkdirAll(d, 0o755)
	}
	git(t, propre, "init", "-q")
	ecrire(t, filepath.Join(propre, "a.txt"), "a\n")
	git(t, propre, "add", "a.txt")
	git(t, propre, "commit", "-qm", "init")
	git(t, propre, "remote", "add", "origin", "git@gitlab.com:Test/Projet-Propre.git")
	git(t, sale, "init", "-q")
	ecrire(t, filepath.Join(sale, "a.txt"), "a\n")
	ecrire(t, filepath.Join(sale, "b.txt"), "b\n")
	git(t, sale, "add", "a.txt", "b.txt")
	git(t, sale, "commit", "-qm", "init")
	ecrire(t, filepath.Join(sale, "a.txt"), "a\nmodif\n")
	ecrire(t, filepath.Join(sale, "c.txt"), "c\n")
	git(t, sale, "add", "c.txt")
	ecrire(t, filepath.Join(sale, "d.txt"), "neuf\n")
	return filepath.ToSlash(b)
}

var delaiCas = regexp.MustCompile(`@T\+([0-9]+)@`)

func casJSON(t testing.TB, nom, dossier string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(racineDepot, "tests", "cas", nom+".json"))
	if err != nil {
		t.Fatal(err)
	}
	j := strings.ReplaceAll(string(b), "@DIR@", dossier)
	return delaiCas.ReplaceAllStringFunc(j, func(m string) string {
		n, _ := strconv.ParseInt(delaiCas.FindStringSubmatch(m)[1], 10, 64)
		return strconv.FormatInt(slNow+n, 10)
	})
}

type mode struct {
	colonnes      int
	ssh, sshRiche bool
	vars          map[string]string
	couleurs      ansi.Profil
	imposees      bool // AGENTLINE_COLORS ou NO_COLOR
}

var modes = map[string]mode{
	"riche":      {colonnes: 200},
	"etroit":     {colonnes: 60},
	"catppuccin": {colonnes: 200, vars: map[string]string{"AGENTLINE_PALETTE": "catppuccin"}},
	"ssh":        {colonnes: 200, ssh: true},
	"ssh-riche":  {colonnes: 200, ssh: true, sshRiche: true},
}

func env(t testing.TB, m mode, config string) Env {
	return Env{
		Now: slNow, Loc: time.UTC, Columns: m.colonnes, SSH: m.ssh, SSHRiche: m.sshRiche,
		Couleurs: m.couleurs, CouleursImposees: m.imposees,
		TempDir: t.TempDir(), GitDelai: 5 * time.Second,
		Themes: theme.Sources{Config: config, Integre: agentline.Themes, Windows: runtime.GOOS == "windows"},
		Getenv: func(k string) string { return m.vars[k] },
	}
}

// comparer : la sortie face au fichier de référence (ou le réécrit avec -maj).
func comparer(t *testing.T, fichier, obtenu string) {
	t.Helper()
	if *maj {
		os.MkdirAll(filepath.Dir(fichier), 0o755)
		ecrire(t, fichier, obtenu)
		return
	}
	attendu, err := os.ReadFile(fichier)
	if err != nil {
		t.Fatalf("%v (première fois : go test ./internal/render -maj)", err)
	}
	if obtenu != string(attendu) {
		t.Errorf("%s : différent de la référence\nattendu : %q\nobtenu  : %q", filepath.Base(fichier), attendu, obtenu)
	}
}

// TestReferences : les 30 sorties de tests/golden/ (6 cas × 5 modes), octet pour octet.
func TestReferences(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	cas, _ := filepath.Glob(filepath.Join(racineDepot, "tests", "cas", "*.json"))
	if len(cas) == 0 {
		t.Fatal("aucun cas dans tests/cas")
	}
	for _, c := range cas {
		nom := strings.TrimSuffix(filepath.Base(c), ".json")
		j := casJSON(t, nom, dossier)
		for nm, m := range modes {
			comparer(t, filepath.Join(racineDepot, "tests", "golden", nom+"."+nm+".ans"), Render(j, env(t, m, config)))
		}
	}
}

// TestReferencesThemes : chaque thème sur 3 cas (153 sorties de tests/golden/themes/).
func TestReferencesThemes(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	themes, _ := filepath.Glob(filepath.Join(racineDepot, "themes", "*.theme"))
	if len(themes) < 40 {
		t.Fatalf("%d thèmes seulement", len(themes))
	}
	for _, f := range themes {
		nom := strings.TrimSuffix(filepath.Base(f), ".theme")
		for _, c := range []string{"complet", "tendu", "api"} {
			m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": nom}}
			comparer(t, filepath.Join(racineDepot, "tests", "golden", "themes", nom+"."+c+".ans"), Render(casJSON(t, c, dossier), env(t, m, config)))
		}
	}
}

// themesNiveaux : les thèmes des références par niveau de couleurs et par fond (quelques-uns, variés).
var themesNiveaux = []string{"tokyo", "dracula", "gruvbox"}

// themesClairs : des thèmes faits pour un fond clair, rendus tels quels sur fond clair.
var themesClairs = []string{"tokyo-day", "solarized-light", "github-light"}

// TestReferencesNiveaux : les mêmes cas en 256 couleurs, en 16 couleurs et sans couleur
// (tests/golden/256/, 16/, none/), plus la ligne courte de SSH avec un niveau imposé. Références
// produites par ce test (-maj), puis relues : codes 38;5 et 30-37/90-97, gras et italique gardés sans couleur.
func TestReferencesNiveaux(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	for _, niv := range []ansi.Niveau{ansi.Couleurs256, ansi.Couleurs16, ansi.SansCouleur} {
		rep := filepath.Join(racineDepot, "tests", "golden", niv.String())
		for _, nom := range themesNiveaux {
			for _, c := range []string{"complet", "tendu"} {
				m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": nom}, couleurs: ansi.Profil{Niveau: niv}}
				comparer(t, filepath.Join(rep, nom+"."+c+".ans"), Render(casJSON(t, c, dossier), env(t, m, config)))
			}
		}
		m := mode{colonnes: 200, ssh: true, couleurs: ansi.Profil{Niveau: niv}, imposees: true}
		comparer(t, filepath.Join(rep, "ssh.complet.ans"), Render(casJSON(t, "complet", dossier), env(t, m, config)))
	}
}

// TestReferencesClair : sur fond clair (tests/golden/clair/), des thèmes sombres corrigés, la ligne courte
// de SSH (couleurs ANSI sombres) et un cas en 16 couleurs. Références produites par ce test (-maj), relues :
// mêmes teintes que sur fond sombre, plus foncées ; la ligne courte en 30-37.
func TestReferencesClair(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	rep := filepath.Join(racineDepot, "tests", "golden", "clair")
	clair := ansi.Profil{Fond: ansi.Clair}
	for _, nom := range append(themesNiveaux, themesClairs...) {
		for _, c := range []string{"complet", "tendu"} {
			m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": nom}, couleurs: clair}
			comparer(t, filepath.Join(rep, nom+"."+c+".ans"), Render(casJSON(t, c, dossier), env(t, m, config)))
		}
	}
	comparer(t, filepath.Join(rep, "ssh.complet.ans"), Render(casJSON(t, "complet", dossier), env(t, mode{colonnes: 200, ssh: true, couleurs: clair}, config)))
	m := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": "tokyo"}, couleurs: ansi.Profil{Niveau: ansi.Couleurs16, Fond: ansi.Clair}}
	comparer(t, filepath.Join(rep, "16.tokyo.complet.ans"), Render(casJSON(t, "complet", dossier), env(t, m, config)))
}

// TestSansCouleur : sans couleur, plus aucune séquence de couleur, mais le gras reste (no-color.org).
func TestSansCouleur(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	couleur := regexp.MustCompile(`\x1b\[[0-9;]*(3[0-9]|4[0-9]|9[0-7]|10[0-7])(;[0-9;]*)?m`)
	for _, m := range []mode{
		{colonnes: 200, couleurs: ansi.Profil{Niveau: ansi.SansCouleur}},
		{colonnes: 200, ssh: true, couleurs: ansi.Profil{Niveau: ansi.SansCouleur}, imposees: true},
		{colonnes: 200, ssh: true, couleurs: ansi.Profil{Niveau: ansi.SansCouleur}}, // détecté (TERM=dumb) : la palette fixe, sans couleur
	} {
		s := Render(casJSON(t, "complet", dossier), env(t, m, config))
		if couleur.MatchString(s) {
			t.Errorf("%+v : une couleur reste : %q", m, s)
		}
		if !strings.Contains(s, "\x1b[1m") {
			t.Errorf("%+v : le gras a disparu : %q", m, s)
		}
	}
}

// TestThemeDuProjet : le thème associé au projet (par remote, ou par chemin), sinon celui par défaut.
func TestThemeDuProjet(t *testing.T) {
	dossier := fixtures(t)
	config := t.TempDir()
	s := theme.Sources{Config: config, Windows: runtime.GOOS == "windows"}
	ecrire(t, filepath.Join(config, "projets"),
		s.ProjetID(dossier+"/projet-propre")+"\tnord\n"+s.ProjetID(dossier+"/projet-sale")+"\tdracula\n")
	ecrire(t, filepath.Join(config, "defaut"), "gruvbox\n")
	for c, attendu := range map[string]string{"complet": "nord", "tendu": "dracula", "hors-git": "gruvbox"} {
		j := casJSON(t, c, dossier)
		force := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": attendu}}
		if obtenu, voulu := Render(j, env(t, modes["riche"], config)), Render(j, env(t, force, t.TempDir())); obtenu != voulu {
			t.Errorf("cas %s : le rendu devrait être celui du thème %s", c, attendu)
		}
	}
}

func TestThemeDeBranche(t *testing.T) {
	dossier := fixtures(t)
	git(t, dossier+"/projet-sale", "checkout", "-q", "-b", "release/1.2")
	config := t.TempDir()
	s := theme.Sources{Config: config, Windows: runtime.GOOS == "windows"}
	ecrire(t, filepath.Join(config, "projets"),
		s.ProjetID(dossier+"/projet-propre")+"\tnord\n"+s.ProjetID(dossier+"/projet-sale")+"\tdracula\n")
	ecrire(t, filepath.Join(config, "defaut"), "gruvbox\n")
	ecrire(t, filepath.Join(config, "branches"), "ma*\tmonokai\nmain\tkanagawa\nrelease/*\tsolarized\n")
	// la branche passe avant le projet ; le nom exact avant le motif ; hors Git, rien ne change
	for c, attendu := range map[string]string{"complet": "kanagawa", "tendu": "solarized", "hors-git": "gruvbox"} {
		j := casJSON(t, c, dossier)
		force := mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": attendu}}
		if obtenu, voulu := Render(j, env(t, modes["riche"], config)), Render(j, env(t, force, t.TempDir())); obtenu != voulu {
			t.Errorf("cas %s : le rendu devrait être celui du thème %s", c, attendu)
		}
	}
}

// BenchmarkRender : le rendu du cas « complet », état git imposé (ni dépôt ni processus git), pour comparer
// deux versions : go test ./internal/render -run '^$' -bench Render -benchmem
func BenchmarkRender(b *testing.B) {
	j := casJSON(b, "complet", filepath.ToSlash(b.TempDir()))
	config := b.TempDir()
	niveaux := map[string]mode{
		"256": {colonnes: 200, couleurs: ansi.Profil{Niveau: ansi.Couleurs256}},
		"16":  {colonnes: 200, couleurs: ansi.Profil{Niveau: ansi.Couleurs16}},
	}
	for _, nm := range []string{"riche", "ssh", "256", "16"} {
		b.Run(nm, func(b *testing.B) {
			m, ok := modes[nm]
			if !ok {
				m = niveaux[nm]
			}
			e := env(b, m, config)
			e.GitFictif = &GitFictif{Branche: "main", Neuf: 1}
			for b.Loop() {
				Render(j, e)
			}
		})
	}
}

// TestExempleCommeJSON : la session fictive des aperçus donne le même rendu que l'ancien JSON fictif.
func TestExempleCommeJSON(t *testing.T) {
	config := t.TempDir()
	for _, ctx := range []int{0, 42, 91} {
		for _, d := range []string{"/apercu/mon-projet", filepath.ToSlash(t.TempDir())} {
			e := env(t, modes["riche"], config)
			e.GitFictif = &GitFictif{Branche: "main", Neuf: 1}
			j := fmt.Sprintf(`{"model":{"display_name":"Opus 5.5"},"effort":{"level":"high"},"workspace":{"current_dir":%q,"project_dir":%q},"pr":{"number":12,"review_state":"pending"},"context_window":{"used_percentage":%d},"rate_limits":{"five_hour":{"used_percentage":61,"resets_at":%d},"seven_day":{"used_percentage":23,"resets_at":%d}}}`,
				d, d, ctx, e.Now+3600, e.Now+300000)
			s := etat.Exemple(e.Now, d, ctx)
			s.PR = etat.PR{Num: "12", Revue: "pending"}
			if a, o := Render(j, e), RenderSession(s, e); a != o {
				t.Errorf("ctx %d, %s :\nattendu %q\nobtenu  %q", ctx, d, a, o)
			}
		}
	}
}
