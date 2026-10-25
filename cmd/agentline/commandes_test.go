package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/selecteur"
	"github.com/adrien-ln/agentline/internal/style"
)

// TestOptionAvecValeur : une option qui prend une valeur la lit après elle ou après « = » ; une option sans
// valeur la refuse. Rien n'est installé : chaque cas s'arrête sur une erreur d'usage, avant l'installation.
func TestOptionAvecValeur(t *testing.T) {
	for _, c := range []struct {
		args  []string
		texte string
	}{
		{[]string{"install", "--font"}, "il manque la valeur de --font"},
		{[]string{"install", "--font", "comic"}, "police inconnue : comic (au choix : firacode, jetbrainsmono, cascadia, hack, meslo)"},
		{[]string{"install", "--font=comic"}, "police inconnue : comic"},
		{[]string{"install", "--in-place=oui"}, "--in-place ne prend pas de valeur"},
		{[]string{"uninstall", "--font", "hack"}, "option inconnue : --font"},
	} {
		var sortie, erreurs bytes.Buffer
		if code := principale(c.args, &sortie, &erreurs); code != codeUsage || !strings.Contains(erreurs.String(), c.texte) {
			t.Errorf("%v : code %d\n%s", c.args, code, erreurs.String())
		}
	}
	var sortie, erreurs bytes.Buffer
	if principale([]string{"install", "--help"}, &sortie, &erreurs); !strings.Contains(sortie.String(), "--font <name>") {
		t.Errorf("l'aide de install doit montrer la valeur de --font\n%s", sortie.String())
	}
}

// TestCommandeFont : la liste nomme chaque police proposée ; un nom inconnu ou une sous-commande inconnue
// sont des erreurs d'usage, avant toute installation.
func TestCommandeFont(t *testing.T) {
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"font"}, &sortie, &erreurs); code != 0 {
		t.Fatalf("agentline font : code %d\n%s", code, erreurs.String())
	}
	for _, n := range []string{"firacode", "jetbrainsmono", "cascadia", "hack", "meslo"} {
		if !strings.Contains(sortie.String(), n) {
			t.Errorf("la liste doit nommer %s\n%s", n, sortie.String())
		}
	}
	for args, texte := range map[[2]string]string{
		{"set", "comic"}:  "police inconnue : comic",
		{"choose", "x"}:   "commande inconnue : choose",
		{"set", "--font"}: "option inconnue : --font",
	} {
		erreurs.Reset()
		if code := principale([]string{"font", args[0], args[1]}, &sortie, &erreurs); code != codeUsage || !strings.Contains(erreurs.String(), texte) {
			t.Errorf("font %v : code %d\n%s", args, code, erreurs.String())
		}
	}
}

// TestCommandeFontTest : « agentline font test » dessine chaque glyphe d'agentline entre deux barres, puis la
// statusline d'exemple ; rien n'est modifié.
func TestCommandeFontTest(t *testing.T) {
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"font", "test"}, &sortie, &erreurs); code != 0 {
		t.Fatalf("agentline font test : code %d\n%s", code, erreurs.String())
	}
	for _, x := range []string{"|" + style.GlypheDuJeu("nerd", "branch") + "|", "|" + selecteur.GlyphesTouches()[0][1] + "|", "|▟|", "|━|", "ctx"} {
		if !strings.Contains(sortie.String(), x) {
			t.Errorf("%q absent\n%s", x, sortie.String())
		}
	}
}

// TestCommandeThemeNew : un nom invalide, un nom déjà pris ou un thème de départ inconnu sont des erreurs
// d'usage, avant l'ouverture de l'éditeur.
func TestCommandeThemeNew(t *testing.T) {
	for args, texte := range map[[3]string]string{
		{"new", "Ma Nuit", ""}:           "nom de thème invalide : Ma Nuit",
		{"new", "tokyo", ""}:             "un thème s'appelle déjà tokyo",
		{"new", "ma-nuit", "--from=zzz"}: "thème inconnu : zzz",
		{"edit", "zzz", ""}:              "thème inconnu : zzz",
	} {
		var sortie, erreurs bytes.Buffer
		l := []string{"theme", args[0], args[1]}
		if args[2] != "" {
			l = append(l, args[2])
		}
		if code := principale(l, &sortie, &erreurs); code != codeUsage || !strings.Contains(erreurs.String(), texte) {
			t.Errorf("%v : code %d\n%s", l, code, erreurs.String())
		}
	}
}

// TestCommandeFontPreview : hors de Windows Terminal et de VS Code, l'aperçu en direct n'est pas possible et
// le dit, sans rien modifier.
func TestCommandeFontPreview(t *testing.T) {
	t.Setenv("WT_SESSION", "")
	t.Setenv("TERM_PROGRAM", "")
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"font", "preview"}, &sortie, &erreurs); code != codeEchec || !strings.Contains(erreurs.String(), "agentline font test") {
		t.Errorf("code %d\n%s%s", code, sortie.String(), erreurs.String())
	}
}

// TestCommandeThemeImportExport : l'import refuse ce qu'il ne peut pas lire avant d'ouvrir l'éditeur ;
// l'export écrit le thème avec les noms anglais.
func TestCommandeThemeImportExport(t *testing.T) {
	t.Setenv("WT_SESSION", "")
	vide := filepath.Join(t.TempDir(), "vide.json")
	os.WriteFile(vide, []byte("pas un jeu"), 0o644)
	for _, c := range []struct {
		args  []string
		code  int
		texte string
	}{
		{[]string{"theme", "import"}, codeUsage, "il faut un fichier de jeu de couleurs"},
		{[]string{"theme", "import", filepath.Join(t.TempDir(), "absent.json")}, codeEchec, "absent.json"},
		{[]string{"theme", "import", vide}, codeEchec, "format inconnu"},
		{[]string{"theme", "import", "--terminal"}, codeEchec, "Windows Terminal"},
	} {
		var sortie, erreurs bytes.Buffer
		if code := principale(c.args, &sortie, &erreurs); code != c.code || !strings.Contains(erreurs.String(), c.texte) {
			t.Errorf("%v : code %d\n%s", c.args, code, erreurs.String())
		}
	}
	var sortie, erreurs bytes.Buffer
	if code := principale([]string{"theme", "export", "tokyo"}, &sortie, &erreurs); code != 0 || !strings.HasPrefix(sortie.String(), "# Tokyo Night\n# tags: ") || !strings.Contains(sortie.String(), "TEXT=c0caf5") {
		t.Errorf("export : code %d\n%s%s", code, sortie.String(), erreurs.String())
	}
}

// TestStyleProject : --project règle le style du projet d'ici, sans toucher au style général ; show dit d'où
// vient chaque valeur ; unset et reset --project rendent la main au style général.
func TestStyleProject(t *testing.T) {
	lancerOK := func(args ...string) string {
		t.Helper()
		var sortie, erreurs bytes.Buffer
		if code := principale(args, &sortie, &erreurs); code != 0 {
			t.Fatalf("%v : code %d\n%s", args, code, erreurs.String())
		}
		return sortie.String()
	}
	lancerOK("style", "reset")
	if s := lancerOK("style", "set", "--project", "ctx.width", "12"); !strings.Contains(s, "ctx.width = 12 (ce projet)") {
		t.Errorf("set --project : %q", s)
	}
	if styleFichier().Regle("ctx.width") {
		t.Error("le style général ne doit pas changer")
	}
	if s := lancerOK("style", "show"); !strings.Contains(s, "ctx.width         12  (ce projet)") {
		t.Errorf("show :\n%s", s)
	}
	if s := lancerOK("style", "unset", "--project", "ctx.width"); !strings.Contains(s, "la valeur générale pour ce projet (8)") {
		t.Errorf("unset --project : %q", s)
	}
	lancerOK("style", "hide", "--project", "pr")
	if s := lancerOK("style", "reset", "--project"); !strings.Contains(s, "style du projet retiré (1 réglage)") {
		t.Errorf("reset --project : %q", s)
	}
}
