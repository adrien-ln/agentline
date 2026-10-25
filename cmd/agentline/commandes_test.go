package main

import (
	"bytes"
	"strings"
	"testing"
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
