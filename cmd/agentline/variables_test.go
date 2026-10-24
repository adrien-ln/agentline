package main

import (
	"os"
	"testing"
)

func TestVariableAncienNom(t *testing.T) {
	env := map[string]string{"SL_THEME": "nord", "SL_CONFIG": "/ancien", "AGENTLINE_CONFIG": "/nouveau",
		"SL_STYLE": "defaut", "AGENTLINE_STYLE": "", "SL_AUTRE": "x"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	for cle, attendu := range map[string]string{
		"AGENTLINE_THEME":  "nord",     // seul l'ancien nom est défini : lu en secours
		"AGENTLINE_CONFIG": "/nouveau", // les deux : le nouveau l'emporte
		"AGENTLINE_STYLE":  "",         // le nouveau, défini vide, l'emporte aussi
		"AGENTLINE_AUTRE":  "",         // pas une variable d'agentline : pas de secours
		"AGENTLINE_NOW":    "",
		"SL_THEME":         "nord",
		"HOME":             "",
	} {
		if v := variable(lookup, cle); v != attendu {
			t.Errorf("%s : %q, attendu %q", cle, v, attendu)
		}
	}
	for _, v := range variables {
		if a := anciensNoms("AGENTLINE_" + v); len(a) == 0 || a[len(a)-1] != "SL_"+v && suffixesFrancais[v] == "" {
			t.Errorf("anciens noms de AGENTLINE_%s : %q", v, a)
		}
	}
}

// TestVariablesRenommees : AGENTLINE_SSH_RICH et AGENTLINE_PS_PROFILE ; leurs noms français
// (AGENTLINE_SSH_RICHE, AGENTLINE_PROFIL_PS), puis SL_*, sont lus en secours, dans cet ordre.
func TestVariablesRenommees(t *testing.T) {
	for _, c := range []struct {
		env     map[string]string
		attendu string
	}{
		{map[string]string{"AGENTLINE_SSH_RICH": "1", "AGENTLINE_SSH_RICHE": "2", "SL_SSH_RICHE": "3"}, "1"},
		{map[string]string{"AGENTLINE_SSH_RICHE": "2", "SL_SSH_RICHE": "3"}, "2"},
		{map[string]string{"SL_SSH_RICHE": "3"}, "3"},
		{map[string]string{"AGENTLINE_SSH_RICH": "", "AGENTLINE_SSH_RICHE": "2"}, ""},
	} {
		lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
		if v := variable(lookup, "AGENTLINE_SSH_RICH"); v != c.attendu {
			t.Errorf("%v : %q, attendu %q", c.env, v, c.attendu)
		}
	}
	lookup := func(k string) (string, bool) { return "/p.ps1", k == "AGENTLINE_PROFIL_PS" }
	if v := variable(lookup, "AGENTLINE_PS_PROFILE"); v != "/p.ps1" {
		t.Errorf("AGENTLINE_PROFIL_PS : %q", v)
	}
}

// TestAncienNomDansLaCommande : SL_THEME (ancien nom) est encore lu par « agentline theme current ».
func TestAncienNomDansLaCommande(t *testing.T) {
	configVide(t) // AGENTLINE_THEME et SL_THEME vides, rendus tels qu'avant à la fin du test
	d := t.TempDir()
	t.Setenv("SL_THEME", "nord")
	if o, _, _ := lancerTheme(t, "current", d); o != "tokyo (intégré)\n" {
		t.Errorf("AGENTLINE_THEME défini (vide) doit l'emporter sur SL_THEME : %q", o)
	}
	os.Unsetenv("AGENTLINE_THEME")
	if o, _, _ := lancerTheme(t, "current", d); o != "nord (forcé)\n" {
		t.Errorf("SL_THEME seul doit être lu en secours : %q", o)
	}
	t.Setenv("AGENTLINE_THEME", "dracula")
	if o, _, _ := lancerTheme(t, "current", d); o != "dracula (forcé)\n" {
		t.Errorf("AGENTLINE_THEME l'emporte sur SL_THEME : %q", o)
	}
}
