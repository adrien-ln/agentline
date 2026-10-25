package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
)

func fichierImport(t *testing.T, nom string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "tests", "import", nom))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestLireJeux : chaque format est reconnu à son contenu et donne un jeu complet.
func TestLireJeux(t *testing.T) {
	for _, c := range []struct{ fichier, nom, fond, rouge, titre string }{
		{"campbell.json", "", "0c0c0c", "c50f1f", "Campbell"},
		{"settings.json", "nuit", "1a1b26", "f7768e", "Nuit"},
		{"settings.json", "Solarized Light", "fdf6e3", "dc322f", "Solarized Light"},
		{"dracula.itermcolors", "", "282a36", "ff5555", ""},
		{"base16-ocean.yaml", "", "2b303b", "bf616a", "Ocean"},
	} {
		j, err := LireJeu(fichierImport(t, c.fichier), c.nom)
		if err != nil || j.Fond != c.fond || j.ANSI[1] != c.rouge || j.Nom != c.titre {
			t.Errorf("%s %q : %+v %v", c.fichier, c.nom, j, err)
		}
	}
	if _, err := LireJeu(fichierImport(t, "settings.json"), "absent"); err == nil || !strings.Contains(err.Error(), "Nuit, Solarized Light") {
		t.Errorf("jeu absent : %v", err)
	}
	if j, _ := LireJeu(fichierImport(t, "base16-ocean.yaml"), ""); j.Orange != "d08770" {
		t.Errorf("base16 : base09 est l'orange (%q)", j.Orange)
	}
	if _, err := LireJeu([]byte("pas un jeu"), ""); err == nil {
		t.Error("un contenu inconnu doit être refusé")
	}
}

// TestDepuisJeu : le thème d'un jeu est complet, lisible sur son fond, et reconnaît un fond clair ; sans orange
// dans le jeu, D75 en prend un entre jaune et rouge.
func TestDepuisJeu(t *testing.T) {
	for _, c := range []struct {
		fichier, nom string
		clair        bool
	}{{"campbell.json", "", false}, {"settings.json", "Solarized Light", true}, {"dracula.itermcolors", "", false}, {"base16-ocean.yaml", "", false}} {
		j, _ := LireJeu(fichierImport(t, c.fichier), c.nom)
		p := DepuisJeu(j)
		if p.Clair != c.clair || p.Intensite[4] == "" || p.Sep == "" {
			t.Errorf("%s : %+v", c.fichier, p)
		}
		for _, x := range []string{p.Modele, p.Dossier, p.Lieu, p.D0, p.D50, p.D75, p.D100} {
			if ansi.Contraste(x, j.Fond) < 2.9 {
				t.Errorf("%s : %s illisible sur %s", c.fichier, x, j.Fond)
			}
		}
		if ansi.Contraste(p.Texte, j.Fond) < 4.4 {
			t.Errorf("%s : texte %s illisible", c.fichier, p.Texte)
		}
	}
	p := DepuisJeu(Campbell)
	if e := EtiquetteDe(p.D75); e != "orange" && e != "rouge" && e != "jaune" {
		t.Errorf("orange déduit de Campbell : %s (%s)", p.D75, e)
	}
}

// TestSansCommentaires : commentaires et virgules finales partent, les chaînes restent.
func TestSansCommentaires(t *testing.T) {
	in := "{ // c\n \"a\": \"x // pas un commentaire\", /* b */ \"c\": [1, 2,], }"
	if s := sansCommentaires(in); strings.Contains(s, "/* b */") || strings.Contains(s, ",]") || !strings.Contains(s, "x // pas un commentaire") {
		t.Errorf("%q", s)
	}
}
