package render

import (
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/entree/claude"
	"github.com/adrien-ln/agentline/internal/style"
)

// TestPlaces : RenderPlaces écrit la même statusline que RenderSession, et chaque place couvre le texte de
// son item, sans espace autour, dans l'ordre et sans chevauchement, dans les dispositions et en SSH.
func TestPlaces(t *testing.T) {
	for _, c := range []struct {
		cas, regles string
		ssh         bool
	}{
		{"complet", "", false},
		{"tendu", "", false}, // alertes : le signal ● avant le segment
		{"api", "", false},
		{"complet", "line1=folder git | model effort\nline2=ctx 5h | cost\n", false},
		{"complet", "hide=pr effort\n", false},
		{"complet", "", true},
	} {
		e := env(t, modes["riche"], t.TempDir())
		if c.ssh {
			e = env(t, modes["ssh"], t.TempDir())
		}
		e.Style = style.Lire([]byte(c.regles))
		e.GitFictif = &GitFictif{Branche: "main", Indexe: 1, Modifie: 2, Neuf: 3, Avance: 2, Retard: 1}
		s := claude.Lire([]byte(casJSON(t, c.cas, "/x")))
		sortie, places := RenderPlaces(s, e)
		if sortie != RenderSession(s, e) {
			t.Fatalf("%s %q : RenderPlaces ne rend pas la même statusline", c.cas, c.regles)
		}
		lignes := strings.Split(ansi.Visible(sortie), "\n")
		if len(places) == 0 {
			t.Fatalf("%s : aucune place", c.cas)
		}
		vus := map[string]bool{}
		for i, p := range places {
			l := []rune(lignes[p.Ligne])
			if p.Debut < 0 || p.Fin > len(l) || p.Debut >= p.Fin {
				t.Fatalf("%s : %+v hors de la ligne %q", c.cas, p, string(l))
			}
			x := string(l[p.Debut:p.Fin])
			if strings.TrimSpace(x) != x {
				t.Errorf("%s : %s couvre %q (espaces autour)", c.cas, p.Item, x)
			}
			if i > 0 && places[i-1].Ligne == p.Ligne && places[i-1].Fin > p.Debut {
				t.Errorf("%s : %s chevauche %s", c.cas, p.Item, places[i-1].Item)
			}
			if c.regles == "" && !c.ssh && p.Item == "model" && !strings.HasPrefix(x, strings.Fields(s.Modele)[0]) {
				t.Errorf("%s : model couvre %q", c.cas, x)
			}
			if c.regles == "" && !c.ssh && p.Item == "ctx" && !strings.HasPrefix(x, "ctx") {
				t.Errorf("%s : ctx couvre %q", c.cas, x)
			}
			vus[p.Item] = true
		}
		if strings.Contains(c.regles, "hide=pr") && (vus["pr"] || vus["effort"]) {
			t.Errorf("un item masqué n'a pas de place : %v", vus)
		}
	}
}

// TestPaletteImposee : une palette imposée (aperçu de l'éditeur) rend comme le thème qu'elle copie, sur fond
// sombre comme sur fond clair ; sans palette, rien ne change.
func TestPaletteImposee(t *testing.T) {
	s := claude.Lire([]byte(casJSON(t, "complet", "/x")))
	for _, fond := range []ansi.Fond{ansi.Sombre, ansi.Clair} {
		e := env(t, modes["riche"], t.TempDir())
		e.Couleurs.Fond = fond
		e.GitFictif = &GitFictif{Branche: "main"}
		parNom := e
		parNom.Getenv = func(k string) string {
			if k == "AGENTLINE_THEME" {
				return "dracula"
			}
			return ""
		}
		p, _ := e.Themes.Charger("dracula")
		imposee := e
		imposee.Palette = &p
		if a, b := RenderSession(s, parNom), RenderSession(s, imposee); a != b {
			t.Errorf("fond %v : palette imposée différente du thème\n%q\n%q", fond, a, b)
		}
	}
}

// TestStyleDuProjet : le style d'un projet s'ajoute au style général, dans ce projet seulement ;
// AGENTLINE_STYLE=default l'ignore aussi.
func TestStyleDuProjet(t *testing.T) {
	config := t.TempDir()
	e := env(t, modes["riche"], config)
	e.GitFictif = &GitFictif{Branche: "dev"}
	s := claude.Lire([]byte(casJSON(t, "complet", "/x")))
	projet := s.Projet
	if projet == "" {
		projet = s.Dossier
	}
	sp := &style.Surcharge{}
	sp.Definir("hide", "pr")
	if err := style.EcrireProjet(config, e.Themes.ProjetID(projet), sp); err != nil {
		t.Fatal(err)
	}
	if v := ansi.Visible(RenderSession(s, e)); strings.Contains(v, "#42") {
		t.Errorf("le projet masque la PR :\n%s", v)
	}
	autre := claude.Lire([]byte(casJSON(t, "complet", "/ailleurs")))
	if v := ansi.Visible(RenderSession(autre, e)); !strings.Contains(v, "#42") {
		t.Errorf("un autre projet garde la PR :\n%s", v)
	}
	e.Getenv = func(k string) string {
		if k == "AGENTLINE_STYLE" {
			return "default"
		}
		return ""
	}
	if v := ansi.Visible(RenderSession(s, e)); !strings.Contains(v, "#42") {
		t.Errorf("AGENTLINE_STYLE=default ignore le style du projet :\n%s", v)
	}
}
