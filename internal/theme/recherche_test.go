package theme

import (
	"testing"

	"github.com/adrien-ln/agentline"
)

func TestTrouverEtProche(t *testing.T) {
	s := Sources{Config: t.TempDir(), Integre: agentline.Themes}
	for saisie, nom := range map[string]string{"tokyo": "tokyo", "Tokyo Night": "tokyo", "tokyo night": "tokyo", "DRACULA": "dracula", " nord ": "nord"} {
		if n, ok := s.Trouver(saisie); !ok || n != nom {
			t.Errorf("Trouver(%q) = %q, %v ; attendu %q", saisie, n, ok, nom)
		}
	}
	if n, ok := s.Trouver("tokio"); ok {
		t.Errorf("Trouver(tokio) ne doit rien trouver : %q", n)
	}
	for saisie, nom := range map[string]string{"tokio": "tokyo", "drakula": "dracula", "Tokyo Nite": "tokyo", "solar": "solarized"} {
		if p := s.Proche(saisie); p != nom {
			t.Errorf("Proche(%q) = %q, attendu %q", saisie, p, nom)
		}
	}
	for _, saisie := range []string{"", "zzzzzzzz", "xq"} {
		if p := s.Proche(saisie); p != "" {
			t.Errorf("Proche(%q) = %q, attendu aucune suggestion", saisie, p)
		}
	}
}
