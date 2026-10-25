package selecteur

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/theme"
)

// ouvrirEditeur : le sélecteur sur l'onglet Créer, avec une copie du thème survolé (e).
func ouvrirEditeur(t *testing.T) (*Modele, string) {
	t.Helper()
	m, config := nouveau(t)
	appuyer(m, "e")
	if m.onglet != 2 || !m.ed.actif {
		t.Fatal("e doit ouvrir l'onglet Créer")
	}
	return m, config
}

// TestEditeurCopie : e part d'une copie du thème survolé, à un nom libre ; rien n'attend tant que rien ne change.
func TestEditeurCopie(t *testing.T) {
	m, _ := ouvrirEditeur(t)
	base := m.noms[m.i]
	if !m.ed.nouveau || m.ed.nom != base+"-perso" || !strings.HasSuffix(m.ed.titre, "(perso)") {
		t.Errorf("copie : %q %q nouveau=%v", m.ed.nom, m.ed.titre, m.ed.nouveau)
	}
	if len(m.enAttente()) != 0 {
		t.Errorf("rien de changé, rien en attente : %v", m.enAttente())
	}
	if e := ecran(m); !strings.Contains(e, m.ed.nom+".theme") || !strings.Contains(e, "TEXT") || !strings.Contains(e, "INTENSITY 5") {
		t.Errorf("l'onglet doit montrer le fichier et les rôles :\n%s", e)
	}
}

// TestEditeurReglerEtEnregistrer : ←→ règle le canal choisi, Entrée prend un code, n renomme ; s écrit le thème,
// qui rejoint la liste et devient le thème survolé ; il se rouvre ensuite tel quel, sans copie.
func TestEditeurReglerEtEnregistrer(t *testing.T) {
	m, config := ouvrirEditeur(t)
	avant := m.ed.p.Texte
	appuyer(m, "→")
	if m.ed.p.Texte == avant || !strings.Contains(aplati(ecran(m)), "Non enregistré : thème personnel") {
		t.Fatalf("→ doit éclaircir TEXT et le mettre en attente : %s -> %s", avant, m.ed.p.Texte)
	}
	appuyer(m, "c")
	if m.ed.canal != 1 {
		t.Error("c : canal suivant")
	}
	appuyer(m, "↓", "entrée", "⌫", "⌫", "⌫", "⌫", "⌫", "⌫", "#FF8800", "entrée")
	if m.ed.p.Discret != "ff8800" || m.ed.saisie {
		t.Errorf("code tapé pour MUTED : %q (saisie %v)", m.ed.p.Discret, m.ed.saisie)
	}
	appuyer(m, "entrée", "zz", "entrée")
	if !m.ed.saisie || !strings.Contains(ecran(m), "code de couleur") {
		t.Error("un code invalide est refusé, la saisie reste ouverte")
	}
	appuyer(m, "échap", "n")
	for range []rune(m.ed.titre) {
		appuyer(m, "⌫")
	}
	appuyer(m, "Nord", "entrée")
	if !strings.Contains(ecran(m), "s'appelle déjà nord") {
		t.Error("un nom déjà pris est refusé")
	}
	appuyer(m, "⌫", "⌫", "⌫", "⌫", "Ma Nuit d'été", "entrée")
	if m.ed.nom != "ma-nuit-d-ete" || m.ed.titre != "Ma Nuit d'été" {
		t.Fatalf("renommer : %q %q", m.ed.nom, m.ed.titre)
	}
	if appuyer(m, "s") {
		t.Fatal("s enregistre sans fermer")
	}
	b, err := os.ReadFile(filepath.Join(config, "themes", "ma-nuit-d-ete.theme"))
	if err != nil || !strings.HasPrefix(string(b), "# Ma Nuit d'été\n") || !strings.Contains(string(b), "MUTED=ff8800") {
		t.Fatalf("fichier : %q %v", b, err)
	}
	if p, _ := m.sources.Charger("ma-nuit-d-ete"); !reflect.DeepEqual(p, m.ed.p) || m.ed.modifie() || m.ed.nouveau {
		t.Error("relu, le thème doit être celui de l'éditeur, et plus rien n'attend")
	}
	if m.noms[m.i] != "ma-nuit-d-ete" {
		t.Errorf("le thème enregistré doit être survolé : %s", m.noms[m.i])
	}
	appuyer(m, "tab", "e") // l'onglet Thèmes, puis e sur le thème de l'utilisateur : lui-même, pas une copie
	if m.ed.nom != "ma-nuit-d-ete" || m.ed.nouveau {
		t.Errorf("rouvrir son thème : %q nouveau=%v", m.ed.nom, m.ed.nouveau)
	}
}

// TestEditeurAnnulerEtContraste : u revient à ce qui est enregistré ; un texte illisible sur son fond est signalé.
func TestEditeurAnnulerEtContraste(t *testing.T) {
	m, _ := ouvrirEditeur(t)
	garde := m.ed.p
	appuyer(m, "entrée", "⌫", "⌫", "⌫", "⌫", "⌫", "⌫", "1a1b26", "entrée")
	if !strings.Contains(m.ligneRole(0, 80), "⚠") {
		t.Errorf("TEXT de la couleur du fond : ⚠ attendu : %q", visible(m.ligneRole(0, 80)))
	}
	if appuyer(m, "u") || !reflect.DeepEqual(m.ed.p, garde) || m.ed.modifie() {
		t.Error("u : le thème revient à la copie de départ")
	}
	sombre := m.View()
	appuyer(m, "f")
	if m.View() == sombre || !strings.Contains(ecran(m), "fond clair") {
		t.Error("f : l'aperçu passe sur l'autre fond")
	}
	appuyer(m, "v")
	if !strings.Contains(ecran(m), "256 couleurs") {
		t.Error("v : l'aperçu en 256 couleurs")
	}
}

// TestNomDeFichier : un nom affiché devient un nom de fichier de thème.
func TestNomDeFichier(t *testing.T) {
	for in, out := range map[string]string{"Ma Nuit d'été": "ma-nuit-d-ete", "  Tokyo  2 ": "tokyo-2", "Œuvre!": "oeuvre"} {
		if n := theme.NomDeTitre(in); n != out {
			t.Errorf("%q -> %q, attendu %q", in, n, out)
		}
	}
}

// TestResponsiveEditeur : à chaque taille, l'onglet Créer tient dans le terminal.
func TestResponsiveEditeur(t *testing.T) {
	for _, tl := range tailles {
		w, h := tl[0], tl[1]
		for _, etat := range [][]string{nil, {"entrée"}, {"?"}, {"f", "v"}, {"→", "q"}} {
			m, _ := ouvrirEditeur(t)
			m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			appuyer(m, etat...)
			l := strings.Split(m.View(), "\n")
			if len(l) > max(h, hauteurMin) {
				t.Errorf("%dx%d %v : %d lignes", w, h, etat, len(l))
			}
			for k, x := range l {
				if n := largeur(x); n > max(w, largeurMin) {
					t.Errorf("%dx%d %v : ligne %d de %d colonnes", w, h, etat, k, n)
				}
			}
		}
	}
}

// TestOuvrirEditeurNomme : agentline theme new <nom> : la copie prend ce nom et un titre tiré de lui, et elle
// est à créer tout de suite (s l'écrit même sans autre changement).
func TestOuvrirEditeurNomme(t *testing.T) {
	m, config := nouveau(t)
	m.OuvrirEditeur("nord", "ma-nuit")
	if m.onglet != 2 || m.ed.nom != "ma-nuit" || m.ed.titre != "Ma nuit" || !m.ed.modifie() {
		t.Fatalf("éditeur : onglet %d, %q %q modifié=%v", m.onglet, m.ed.nom, m.ed.titre, m.ed.modifie())
	}
	appuyer(m, "s")
	if _, err := os.Stat(filepath.Join(config, "themes", "ma-nuit.theme")); err != nil {
		t.Errorf("s doit créer le thème : %v", err)
	}
	p, _ := m.sources.Charger("ma-nuit")
	if nord, _ := m.sources.Charger("nord"); p != nord {
		t.Errorf("le thème créé doit reprendre les couleurs de nord :\n%+v\n%+v", p, nord)
	}
}

// TestEditeurOutils : g accorde les seuils à la couleur choisie, G refait le thème autour d'elle, a fait défiler
// les visions (et avertit quand des seuils se confondent) ; les étiquettes écrites sont calculées.
func TestEditeurOutils(t *testing.T) {
	m, config := ouvrirEditeur(t)
	appuyer(m, "↓", "↓", "↓", "↓") // MODEL
	avant := m.ed.p.D0
	appuyer(m, "g")
	if m.ed.p.D0 == avant || theme.EtiquetteDe(m.ed.p.D0) != "vert" || theme.EtiquetteDe(m.ed.p.D100) != "rouge" {
		t.Errorf("g : D0 %s -> %s, D100 %s", avant, m.ed.p.D0, m.ed.p.D100)
	}
	appuyer(m, "G")
	if m.ed.p.Modele != m.ed.couleur(4) || m.ed.p.Texte == "" {
		t.Errorf("G : %+v", m.ed.p)
	}
	appuyer(m, "a", "a")
	if m.ed.vision != 2 || !strings.Contains(ecran(m), "deutéranopie") {
		t.Errorf("a, a : deutéranopie attendue (%d)", m.ed.vision)
	}
	m.ed.p.D75, m.ed.p.D100 = "e08040", "d06030" // proches, et plus encore en deutéranopie
	if c := m.ed.confusions(); len(c) == 0 || !strings.Contains(ecran(m), "se confondent") {
		t.Errorf("seuils proches en deutéranopie : %v", c)
	}
	appuyer(m, "s")
	b, _ := os.ReadFile(filepath.Join(config, "themes", m.ed.nom+".theme"))
	if l := strings.Split(string(b), "\n"); len(l) < 2 || !strings.HasPrefix(l[1], "# tags: ") || strings.Contains(l[1], "popular") {
		t.Errorf("étiquettes calculées, jamais « popular » :\n%s", b)
	}
}

// TestOuvrirImport : un thème importé s'ouvre dans l'éditeur, à créer sous un nom tiré de son titre.
func TestOuvrirImport(t *testing.T) {
	m, _ := nouveau(t)
	m.OuvrirImport(theme.Harmoniser("3fa7d6", false), "Nuit Bleue")
	if m.onglet != 2 || m.ed.nom != "nuit-bleue" || !m.ed.nouveau || !m.ed.modifie() {
		t.Errorf("import : onglet %d, %q, nouveau=%v, à créer=%v", m.onglet, m.ed.nom, m.ed.nouveau, m.ed.modifie())
	}
	appuyer(m, "u")
	if !m.ed.modifie() {
		t.Error("u garde le thème importé à créer")
	}
}
