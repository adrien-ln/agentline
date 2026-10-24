package main

import (
	"os"
	"strings"
)

// variables : les variables d'environnement d'agentline, sans le préfixe AGENTLINE_. Chacune a un
// ancien nom, SL_<même suffixe> (obsolète), lu en secours quand le nouveau n'est pas défini.
// COLORS et BACKGROUND sont venues après le renommage : leur ancien nom n'a jamais servi.
var variables = []string{"CONFIG", "THEME", "STYLE", "PALETTE", "SSH_RICH", "COLORS", "BACKGROUND", "NOW", "GIT_TIMEOUT", "PS_PROFILE"}

// suffixesFrancais : les variables renommées en anglais -> leur nom d'avant (AGENTLINE_SSH_RICHE,
// SL_SSH_RICHE…), lus en secours.
var suffixesFrancais = map[string]string{"SSH_RICH": "SSH_RICHE", "PS_PROFILE": "PROFIL_PS"}

// anciensNoms : les anciens noms de AGENTLINE_X, du plus récent au plus ancien (AGENTLINE_<nom français>,
// puis SL_…), si c'est une variable d'agentline ; aucun sinon.
func anciensNoms(cle string) []string {
	suffixe, ok := strings.CutPrefix(cle, "AGENTLINE_")
	if !ok {
		return nil
	}
	for _, v := range variables {
		if v == suffixe {
			if f, ok := suffixesFrancais[v]; ok {
				return []string{"AGENTLINE_" + f, "SL_" + f}
			}
			return []string{"SL_" + v}
		}
	}
	return nil
}

// variable : comme os.Getenv, avec les anciens noms en secours ; le nouveau, défini (même vide), l'emporte.
func variable(lookup func(string) (string, bool), cle string) string {
	for _, n := range append([]string{cle}, anciensNoms(cle)...) {
		if v, ok := lookup(n); ok {
			return v
		}
	}
	return ""
}

// getenv : l'environnement tel que le lit agentline (à passer partout à la place d'os.Getenv).
func getenv(cle string) string { return variable(os.LookupEnv, cle) }
