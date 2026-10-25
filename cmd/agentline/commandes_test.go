package main

import (
	"bytes"
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
