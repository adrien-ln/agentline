package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/installation"
	"github.com/adrien-ln/agentline/internal/theme"
)

// emplacementsConfig : le dossier des réglages, et l'ancien (celui de « statusline », avant le
// renommage) d'où le copier au premier accès ; ancien vide quand le dossier est imposé par
// AGENTLINE_CONFIG (ou SL_CONFIG) : il est alors pris tel quel.
func emplacementsConfig() (nouveau, ancien string) {
	if d := getenv("AGENTLINE_CONFIG"); d != "" {
		return d, ""
	}
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		h := getenv("HOME")
		if runtime.GOOS == "windows" || h == "" { // sous Windows, HOME peut être au format Git Bash (/c/…)
			h, _ = os.UserHomeDir()
		}
		base = filepath.Join(h, ".config")
	}
	return filepath.Join(base, "agentline"), filepath.Join(base, "statusline")
}

// migration : ce que le premier accès aux réglages a fait de l'ancien dossier, pour doctor et install.
type etatMigration struct {
	une             sync.Once
	ancien, nouveau string
	copie           bool
	err             error
}

var migration = &etatMigration{}

// dossierConfig : le dossier des réglages ; au premier appel, l'ancien dossier y est copié s'il le faut
// (silencieusement : la statusline n'affiche jamais d'erreur).
func dossierConfig() string {
	d, a := emplacementsConfig()
	if a != "" {
		migration.une.Do(func() {
			migration.ancien, migration.nouveau = a, d
			migration.copie, migration.err = theme.Migrer(a, d)
		})
	}
	return d
}

// verifierMigration : ce que doctor et install disent de l'ancien dossier des réglages ; faux s'il n'y
// a rien à dire (pas d'ancien dossier, ou dossier imposé par AGENTLINE_CONFIG).
func verifierMigration() (installation.Verification, bool) {
	dossierConfig()
	a, d := migration.ancien, migration.nouveau
	switch {
	case a == "":
		return installation.Verification{}, false
	case migration.err != nil:
		return installation.Verification{Niveau: installation.Avertissement, Sujet: i18n.T("Réglages"),
			Detail:     i18n.T("copie de %s vers %s impossible : %s", a, d, migration.err.Error()),
			Correction: i18n.T("copie le dossier à la main, puis : agentline doctor")}, true
	case migration.copie:
		return installation.Verification{Niveau: installation.Bon, Sujet: i18n.T("Réglages"),
			Detail: i18n.T("copiés de %s vers %s (l'ancien dossier est gardé tel quel)", a, d)}, true
	}
	if fi, err := os.Stat(a); err == nil && fi.IsDir() {
		return installation.Verification{Niveau: installation.Bon, Sujet: i18n.T("Réglages"),
			Detail: i18n.T("ancien dossier %s gardé, plus lu (les réglages sont dans %s)", a, d)}, true
	}
	return installation.Verification{}, false
}
