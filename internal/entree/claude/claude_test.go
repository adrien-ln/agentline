package claude

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/etat"
)

// cas : un fichier de tests/cas, @DIR@ remplacé par /d et @T+n@ par « 18000n » (recopié tel quel).
func cas(t *testing.T, nom string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "cas", nom+".json"))
	if err != nil {
		t.Fatal(err)
	}
	j := strings.ReplaceAll(string(b), "@DIR@", "/d")
	return []byte(regexp.MustCompile(`@T\+([0-9]+)@`).ReplaceAllString(j, "18000${1}"))
}

func TestLireCas(t *testing.T) {
	attendus := map[string]etat.Session{
		"minimal": {Harness: "claude", Modele: "Opus 5.5"},
		"complet": {
			Harness: "claude", Modele: "Opus 5.5", Effort: "high", Session: `refonte de la "statusline"`,
			Dossier: "/d/projet-propre", Projet: "/d/projet-propre",
			PR:       etat.PR{Num: "42", URL: "https://gitlab.com/test/projet/-/merge_requests/42", Revue: "pending"},
			Contexte: "42",
			Quotas:   etat.Quotas{H5: etat.Quota{Utilise: "61", Reset: "180003600"}, J7: etat.Quota{Utilise: "23", Reset: "18000300000"}},
		},
		"api": {
			Harness: "claude", Modele: "Haiku 5.5", Effort: "low", Agent: "relecteur", Style: "Explanatory",
			Dossier: "/d/projet-sale", PR: etat.PR{Num: "12", Revue: "draft"}, Contexte: "12", Cout: "4.98",
		},
		"hors-git": {
			Harness: "claude", Modele: "Opus 5.5", Effort: "xhigh", Dossier: "/d/hors-git", Contexte: "100",
			Quotas: etat.Quotas{H5: etat.Quota{Utilise: "100", Reset: "18000600"}},
		},
		"plafond": {
			Harness: "claude", Modele: "Sonnet 5.5", Effort: "medium", Dossier: "/d/projet-propre",
			PR: etat.PR{Num: "3", Revue: "approved"}, Contexte: "67",
			Quotas: etat.Quotas{H5: etat.Quota{Utilise: "72", Reset: "180005400"}, J7: etat.Quota{Utilise: "58", Reset: "18000172800"}},
		},
		"tendu": {
			Harness: "claude", Modele: "Opus 5.5", Effort: "max", Rapide: true,
			Dossier: "/d/projet-sale", Projet: "/d/projet-sale", PR: etat.PR{Num: "7", Revue: "changes_requested"}, Contexte: "91",
			Quotas: etat.Quotas{H5: etat.Quota{Utilise: "95", Reset: "180001800"}, J7: etat.Quota{Utilise: "80", Reset: "1800086400"}},
		},
	}
	fichiers, _ := filepath.Glob(filepath.Join("..", "..", "..", "tests", "cas", "*.json"))
	if len(fichiers) != len(attendus) {
		t.Errorf("%d cas dans tests/cas, %d attendus ici", len(fichiers), len(attendus))
	}
	for nom, a := range attendus {
		if obtenu := Lire(cas(t, nom)); obtenu != a {
			t.Errorf("%s :\nattendu %+v\nobtenu  %+v", nom, a, obtenu)
		}
	}
}

// TestLireBrut : les valeurs restent telles qu'écrites, la conversion revient au rendu (voir etat).
func TestLireBrut(t *testing.T) {
	s := Lire([]byte(`{"workspace":{"current_dir":"C:\\Users\\a\\p","git_worktree":"wt1"},"worktree":{"name":"wt2"},` +
		`"fast_mode":false,"context_window":{"used_percentage":41.5},"five_hour":{"used_percentage":1e2,"resets_at":1.8e9}}`))
	if s.Dossier != "C:/Users/a/p" || s.Worktree != "wt2" || s.Rapide || s.Contexte != "41.5" ||
		s.Quotas.H5 != (etat.Quota{Utilise: "1e2", Reset: "1.8e9"}) {
		t.Errorf("obtenu %+v", s)
	}
	if s := Lire([]byte(`{"model":{"display_name":"Op`)); s != (etat.Session{Harness: "claude"}) {
		t.Errorf("JSON tronqué : %+v", s)
	}
}
