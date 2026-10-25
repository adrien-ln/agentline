package entree

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/etat"
)

// cas : un fichier de tests/cas (ou de tests/cas/outils), @DIR@ remplacé par /d et @T+n@ par « 18000n ».
func cas(t *testing.T, nom string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "tests", "cas", nom+".json"))
	if err != nil {
		t.Fatal(err)
	}
	j := strings.ReplaceAll(string(b), "@DIR@", "/d")
	return []byte(regexp.MustCompile(`@T\+([0-9]+)@`).ReplaceAllString(j, "18000${1}"))
}

const now = 18000000

func TestAntigravity(t *testing.T) {
	attendu := etat.Session{
		Harness: "antigravity", Modele: "Gemini 3 Pro", Effort: "high", Session: "refonte de la statusline",
		Agent: "relecteur", Dossier: "/d/projet-propre", Projet: "/d/projet-propre", Contexte: "42", Cout: "1.25",
		Quotas: etat.Quotas{
			H5: etat.Quota{Utilise: "61", Reset: "18003600"},    // 3600 s après now
			J7: etat.Quota{Utilise: "23", Reset: "18000300000"}, // reset_time en secondes Unix
		},
		Colonnes: 120,
	}
	if s := Lire(cas(t, "outils/antigravity"), "", now); !reflect.DeepEqual(s, attendu) {
		t.Errorf("obtenu  %+v\nattendu %+v", s, attendu)
	}
}

func TestCopilot(t *testing.T) {
	attendu := etat.Session{
		Harness: "copilot", Modele: "Claude Sonnet 4.6", Session: "refonte de la statusline",
		Dossier: "/d/projet-propre", Contexte: "30", PR: etat.PR{Num: "42"},
	}
	if s := Lire(cas(t, "outils/copilot"), "", now); !reflect.DeepEqual(s, attendu) {
		t.Errorf("obtenu  %+v\nattendu %+v", s, attendu)
	}
}

// TestDetecter : chaque outil est reconnu à son JSON ; les cas de Claude Code restent à Claude Code.
func TestDetecter(t *testing.T) {
	for nom, outil := range map[string]string{"outils/antigravity": "antigravity", "outils/copilot": "copilot",
		"complet": "claude", "minimal": "claude", "api": "claude", "tendu": "claude", "plafond": "claude", "hors-git": "claude"} {
		if o := Detecter(cas(t, nom)); o != outil {
			t.Errorf("%s : %s, attendu %s", nom, o, outil)
		}
	}
}

// TestOutilImpose : --from passe avant la détection ; un nom inconnu revient à la détection.
func TestOutilImpose(t *testing.T) {
	j := cas(t, "outils/copilot")
	if s := Lire(j, "claude", now); s.Harness != "claude" {
		t.Errorf("--from claude : %s", s.Harness)
	}
	if s := Lire(j, "inconnu", now); s.Harness != "copilot" {
		t.Errorf("--from inconnu : %s", s.Harness)
	}
	if s := Lire([]byte("pas du json"), "antigravity", now); s.Harness != "antigravity" || s.Modele != "" {
		t.Errorf("JSON illisible : %+v", s)
	}
}
