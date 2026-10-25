package main

import (
	"strings"
	"testing"
)

// TestOutilDemande : --from claude et --from=claude ; rien ou un --from sans valeur : vide (détection).
func TestOutilDemande(t *testing.T) {
	for args, attendu := range map[string]string{"--from antigravity": "antigravity", "--from=copilot": "copilot", "": "", "--from": "", "x --from claude": "claude"} {
		if o := outilDemande(strings.Fields(args)); o != attendu {
			t.Errorf("%q : %q, attendu %q", args, o, attendu)
		}
	}
}
