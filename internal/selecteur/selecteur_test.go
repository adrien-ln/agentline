package selecteur

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/theme"
)

// ouvrir : le sélecteur sur le dossier, avec l'environnement donné et des réglages préparés avant
// l'ouverture (regler peut être nil) ; renvoie aussi le dossier des réglages.
func ouvrir(t *testing.T, dossier string, env map[string]string, regler func(s theme.Sources)) (*Modele, string) {
	t.Helper()
	config := t.TempDir()
	s := theme.Sources{Config: config, Integre: agentline.Themes, Windows: runtime.GOOS == "windows"}
	if regler != nil {
		regler(s)
	}
	getenv := func(k string) string { return env[k] }
	e := render.Env{Now: 1800000000, Loc: time.UTC, TempDir: t.TempDir(), Themes: s, Getenv: getenv}
	m := Nouveau(s, e, filepath.ToSlash(dossier), getenv)
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	return m, config
}

func nouveau(t *testing.T) (*Modele, string) { return ouvrir(t, t.TempDir(), nil, nil) }

// depotFictif : un dossier vu comme un dépôt Git (remote et branche courante), sans lancer git.
func depotFictif(t *testing.T, url, branche string) string {
	t.Helper()
	d := t.TempDir()
	os.MkdirAll(filepath.Join(d, ".git"), 0o755)
	os.WriteFile(filepath.Join(d, ".git", "HEAD"), []byte("ref: refs/heads/"+branche+"\n"), 0o644)
	os.WriteFile(filepath.Join(d, ".git", "config"), []byte("[remote \"origin\"]\n\turl = "+url+"\n"), 0o644)
	return d
}

func nouveauDepot(t *testing.T, url, branche string) (*Modele, string) {
	return ouvrir(t, depotFictif(t, url, branche), nil, nil)
}

var touches = map[string]tea.KeyType{
	"↑": tea.KeyUp, "↓": tea.KeyDown, "début": tea.KeyHome, "fin": tea.KeyEnd, "pgsuiv": tea.KeyPgDown,
	"←": tea.KeyLeft, "→": tea.KeyRight, "entrée": tea.KeyEnter, "échap": tea.KeyEsc, "⌫": tea.KeyBackspace,
}

// appuyer : des touches nommées (voir touches) ou du texte tapé ; renvoie vrai si le sélecteur s'est fermé.
func appuyer(m *Modele, ks ...string) bool {
	for _, k := range ks {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		if t, ok := touches[k]; ok {
			msg = tea.KeyMsg{Type: t}
		}
		if _, cmd := m.Update(msg); cmd != nil {
			if _, fin := cmd().(tea.QuitMsg); fin {
				return true
			}
		}
	}
	return false
}

func ecran(m *Modele) string { return couleurs.ReplaceAllString(m.View(), "") }

// lignesFenetre : les largeurs distinctes des lignes de la mini-fenêtre.
func lignesFenetre(e string) map[int]bool {
	l := map[int]bool{}
	cadre := regexp.MustCompile(`(?:^|  )([╭│╰].*[╮│╯])$`)
	for _, x := range strings.Split(e, "\n") {
		if c := cadre.FindStringSubmatch(x); c != nil {
			l[len([]rune(c[1]))] = true
		}
	}
	return l
}

func populaires(m *Modele) []int {
	var p []int
	for k := range m.noms {
		if m.aLeTag(k, "populaire") {
			p = append(p, k)
		}
	}
	return p
}

func TestDepartListeCompacte(t *testing.T) {
	m, _ := nouveau(t)
	if m.noms[m.i] != "tokyo" {
		t.Fatalf("thème de départ : %s, attendu tokyo", m.noms[m.i])
	}
	e := ecran(m)
	n := 0
	for _, l := range strings.Split(e, "\n") {
		if regexp.MustCompile(`^( › |  ●|   [^\s↑↓])`).MatchString(l) {
			n++
		}
	}
	if n != m.lignesListe {
		t.Errorf("%d thèmes visibles, attendu %d", n, m.lignesListe)
	}
	if regexp.MustCompile(`(?m)^ *[↑↓]`).FindAllString(e, -1)[0] != "   ↑" {
		t.Errorf("indicateurs de défilement non alignés sur les noms des thèmes :\n%s", e)
	}
	reste := len(m.vue) - m.haut - m.lignesListe
	if !strings.Contains(e, "↑ "+strconv.Itoa(m.haut)+" de plus") || (reste > 0) != strings.Contains(e, "↓ "+strconv.Itoa(reste)+" de plus") {
		t.Errorf("indicateurs de défilement faux (%d au-dessus, %d au-dessous) :\n%s", m.haut, reste, e)
	}
	if m.p < m.haut || m.p >= m.haut+m.lignesListe {
		t.Errorf("le thème en vigueur doit être visible : position %d, liste de %d à %d", m.p, m.haut, m.haut+m.lignesListe)
	}
	if !strings.Contains(e, "Claude Code · mon-projet") || !strings.Contains(e, "main ?1") || !strings.Contains(e, "ctx") {
		t.Errorf("mini-fenêtre ou statusline fictive absente :\n%s", e)
	}
	appuyer(m, "début")
	if e := ecran(m); !strings.Contains(e, "↓ "+strconv.Itoa(len(m.noms)-m.lignesListe)+" de plus") || regexp.MustCompile(`↑ \d+ de plus`).MatchString(e) {
		t.Errorf("Début : indicateurs faux :\n%s", e)
	}
	appuyer(m, "pgsuiv")
	if m.p != m.lignesListe {
		t.Errorf("Page suivante : position %d, attendu %d", m.p, m.lignesListe)
	}
}

func TestBorduresAlignees(t *testing.T) {
	for _, l := range []int{110, 70} { // à côté de la liste, puis dessous
		m, _ := nouveau(t)
		m.Update(tea.WindowSizeMsg{Width: l, Height: 40})
		for _, k := range []string{"début", "↓", "↓", "fin"} {
			appuyer(m, k)
			if w := lignesFenetre(ecran(m)); len(w) != 1 {
				t.Errorf("largeur %d, après %s : lignes de largeurs %v", l, k, w)
			}
		}
	}
}

func TestFiltres(t *testing.T) {
	m, _ := nouveau(t)
	appuyer(m, "→")
	if m.filtres[m.f] != "populaire" || len(m.vue) != len(populaires(m)) {
		t.Fatalf("→ : filtre %s, %d thèmes, attendu populaire et %d", m.filtres[m.f], len(m.vue), len(populaires(m)))
	}
	e := ecran(m)
	if !strings.Contains(e, "Tokyo Night") || strings.Contains(e, "Néon") {
		t.Errorf("filtre populaire : contenu faux :\n%s", e)
	}
	m, _ = nouveau(t)
	appuyer(m, "←")
	if dernier := m.filtres[len(m.filtres)-1]; m.f != len(m.filtres)-1 || dernier != filtresOrdre[len(filtresOrdre)-1] {
		t.Errorf("← depuis « tous » : %s, attendu le dernier filtre (%s)", m.filtres[m.f], dernier)
	}
}

func TestRechercheEtDefaut(t *testing.T) {
	m, config := nouveau(t)
	if appuyer(m, "/", "d", "r", "a", "x", "⌫", "entrée") {
		t.Fatal("la recherche ne doit pas fermer le sélecteur")
	}
	if len(m.vue) != 1 || m.noms[m.vue[0]] != "dracula" {
		t.Fatalf("/dra : %d thèmes", len(m.vue))
	}
	if _, err := os.Stat(filepath.Join(config, "defaut")); err == nil {
		t.Fatal("rien ne doit être écrit avant la validation")
	}
	if !appuyer(m, "entrée", "d") {
		t.Fatal("Entrée puis d doivent fermer le sélecteur")
	}
	if b, _ := os.ReadFile(filepath.Join(config, "defaut")); string(b) != "dracula\n" {
		t.Errorf("defaut = %q, attendu dracula", b)
	}
}

func TestRechercheSansResultat(t *testing.T) {
	m, config := nouveau(t)
	appuyer(m, "/", "z", "z", "z", "entrée")
	if e := ecran(m); !strings.Contains(e, "aucun thème") {
		t.Errorf("« aucun thème » absent :\n%s", e)
	}
	if w := lignesFenetre(ecran(m)); len(w) != 1 {
		t.Errorf("bordures décalées : %v", w)
	}
	appuyer(m, "entrée")
	if m.confirmer {
		t.Error("Entrée sans thème ne doit rien proposer de garder")
	}
	if appuyer(m, "échap") || m.recherche != "" || len(m.vue) != len(m.noms) {
		t.Error("Échap doit d'abord effacer la recherche, sans quitter")
	}
	if !appuyer(m, "échap") || m.Message() != "Annulé : Tokyo Night reste en vigueur (intégré)." {
		t.Errorf("Échap : %q", m.Message())
	}
	if e, _ := os.ReadDir(config); len(e) != 0 {
		t.Errorf("des réglages ont été écrits : %v", e)
	}
}

// Mesures : une image pour un thème jamais affiché (rendu de la statusline compris), puis en cache.
func BenchmarkImageNouveauTheme(b *testing.B) {
	t := &testing.T{}
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		m, _ := nouveau(t)
		appuyer(m, "début")
		b.StartTimer()
		appuyer(m, "↓")
		m.View()
	}
}

func BenchmarkImageEnCache(b *testing.B) {
	m, _ := nouveau(&testing.T{})
	appuyer(m, "↑")
	m.View()
	appuyer(m, "↓")
	m.View()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if n%2 == 0 {
			appuyer(m, "↑")
		} else {
			appuyer(m, "↓")
		}
		m.View()
	}
}

func TestThemeDuProjet(t *testing.T) {
	m, config := nouveau(t)
	appuyer(m, "↑", "entrée")
	if !m.confirmer {
		t.Fatal("Entrée doit demander quoi garder")
	}
	appuyer(m, "échap")
	if m.confirmer {
		t.Fatal("Échap doit revenir à la liste")
	}
	nom := m.noms[m.i]
	if !appuyer(m, "entrée", "p") {
		t.Fatal("Entrée puis p doivent fermer le sélecteur")
	}
	b, _ := os.ReadFile(filepath.Join(config, "projets"))
	if !strings.HasSuffix(string(b), "\t"+nom+"\n") || !strings.Contains(m.Message(), "pour le projet") {
		t.Errorf("projets = %q, message = %q", b, m.Message())
	}
	// le message de sortie : le titre, quand ce sera visible, et rien de masqué ici
	if msg := m.Message(); !strings.HasPrefix(msg, "✓ "+m.titres[m.i]+" ") || !strings.Contains(msg, "prochain rafraîchissement") || strings.Contains(msg, "⚠") {
		t.Errorf("message de sortie : %q", msg)
	}
}

// TestValidation80Colonnes : à 80×24, avec un vrai remote, le panneau de validation montre toutes ses
// touches (replié, jamais tronqué) et remplace le pied de page.
func TestValidation80Colonnes(t *testing.T) {
	m, _ := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "feat/interface")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	appuyer(m, "entrée")
	e := ecran(m)
	for _, x := range []string{"[p] ce projet (statusline)", "[b] la branche feat/interface", "[d] partout (défaut)", "[Échap] retour"} {
		if !strings.Contains(e, x) {
			t.Errorf("%q absent du panneau :\n%s", x, e)
		}
	}
	if strings.Contains(e, "[u]") {
		t.Error("rien à retirer ici : pas de [u]")
	}
	if strings.Contains(e, "↑↓ parcourir") {
		t.Error("le panneau remplace le pied de page")
	}
	for k, l := range strings.Split(m.View(), "\n") {
		if largeur(l) > 80 {
			t.Errorf("ligne %d de %d colonnes", k, largeur(l))
		}
	}
	if len(m.d.pied) < 2 {
		t.Errorf("à 80 colonnes, le panneau doit se replier : %d ligne(s)", len(m.d.pied))
	}
}

// TestEchapParCrans : Échap remonte d'un cran (question, recherche, filtre, puis quitter) ; q quitte toujours.
func TestEchapParCrans(t *testing.T) {
	m, _ := nouveau(t)
	appuyer(m, "→", "/", "z", "z", "z", "entrée", "entrée")
	if m.confirmer || m.recherche != "zzz" || m.f == 0 {
		t.Fatalf("préparation : question %v, recherche %q, filtre %d", m.confirmer, m.recherche, m.f)
	}
	appuyer(m, "↓") // retrouver un thème pour pouvoir poser la question
	appuyer(m, "échap")
	if m.recherche != "" || m.f == 0 {
		t.Errorf("1er Échap : la recherche seule doit être effacée (recherche %q, filtre %d)", m.recherche, m.f)
	}
	appuyer(m, "entrée")
	if !m.confirmer {
		t.Fatal("Entrée doit poser la question")
	}
	if appuyer(m, "échap") || m.confirmer || m.f == 0 {
		t.Error("Échap dans la question : retour à la liste, filtre gardé")
	}
	if appuyer(m, "échap") || m.f != 0 {
		t.Errorf("2e Échap : le filtre doit revenir à « tous » (%s)", m.filtres[m.f])
	}
	if !appuyer(m, "échap") {
		t.Error("3e Échap : quitter")
	}
	for _, avant := range [][]string{{"entrée"}, {"?"}, {"→"}} {
		m, _ := nouveau(t)
		appuyer(m, avant...)
		if !appuyer(m, "q") {
			t.Errorf("q après %v doit quitter", avant)
		}
	}
	m, _ = nouveau(t)
	if appuyer(m, "/", "q") || m.recherche != "q" {
		t.Error("en recherche, q est une lettre comme les autres")
	}
}

// TestAvertissementMasque : la question signale le choix qui restera masqué ici, et le message de sortie aussi.
func TestAvertissementMasque(t *testing.T) {
	d := depotFictif(t, "git@gitlab.com:a/b.git", "main")
	m, config := ouvrir(t, d, nil, func(s theme.Sources) { s.RegleBranche("main", "nord") })
	appuyer(m, "↓", "entrée")
	e := ecran(m)
	if !strings.Contains(e, "⚠ Ici, la branche main (Nord) l'emporte sur [p] [d] ; [u] le retire") {
		t.Errorf("avertissement absent ou faux :\n%s", e)
	}
	if !strings.Contains(e, "[u] retirer la règle main (Nord)") {
		t.Errorf("[u] doit dire ce qu'il retire :\n%s", e)
	}
	appuyer(m, "p")
	if !strings.Contains(m.Message(), "⚠ Ici, la branche main (Nord) reste prioritaire") {
		t.Errorf("message de sortie sans avertissement : %q", m.Message())
	}

	m, _ = ouvrir(t, d, map[string]string{"AGENTLINE_THEME": "dracula"}, nil)
	appuyer(m, "entrée")
	if e := ecran(m); !strings.Contains(e, "AGENTLINE_THEME=dracula (Dracula) l'emporte sur [p] [b] [d]") {
		t.Errorf("AGENTLINE_THEME doit masquer tous les choix :\n%s", e)
	}
	m, _ = ouvrir(t, d, nil, nil)
	appuyer(m, "entrée")
	if strings.Contains(ecran(m), "⚠") {
		t.Error("rien de réglé : aucun avertissement")
	}
	_ = config
}

// TestBrancheEtRetirer : b garde le thème pour la branche courante, u retire ce qui s'applique ici.
func TestBrancheEtRetirer(t *testing.T) {
	d := depotFictif(t, "git@gitlab.com:a/b.git", "feat/x")
	m, config := ouvrir(t, d, nil, nil)
	appuyer(m, "↓")
	nom := m.noms[m.i]
	if !appuyer(m, "entrée", "b") {
		t.Fatal("Entrée puis b doivent fermer le sélecteur")
	}
	if b, _ := os.ReadFile(filepath.Join(config, "branches")); string(b) != "feat/x\t"+nom+"\n" {
		t.Errorf("branches = %q", b)
	}
	if !strings.Contains(m.Message(), "pour la branche feat/x") {
		t.Errorf("message : %q", m.Message())
	}
	s := theme.Sources{Config: config, Integre: agentline.Themes, Windows: runtime.GOOS == "windows"}
	m, _ = ouvrir(t, d, nil, func(x theme.Sources) {
		x.RegleBranche("feat/x", nom)
		x.Associer(x.ProjetID(filepath.ToSlash(d)), "nord")
	})
	if !appuyer(m, "entrée", "u") || !strings.Contains(m.Message(), "Règle de branche feat/x retiré") || !strings.Contains(m.Message(), "Nord (projet)") {
		t.Errorf("u doit retirer la règle de la branche d'abord : %q", m.Message())
	}
	_ = s
	m, _ = nouveau(t) // hors Git : pas de branche à proposer
	appuyer(m, "entrée")
	if e := ecran(m); strings.Contains(e, "[b]") || strings.Contains(e, "[u]") {
		t.Errorf("hors Git et sans réglage : ni [b] ni [u] :\n%s", e)
	}
	if appuyer(m, "b") || !m.confirmer {
		t.Error("b indisponible : la touche ne fait rien")
	}
}

// TestRechercheTags : la recherche porte aussi sur les couleurs (tags).
func TestRechercheTags(t *testing.T) {
	m, _ := nouveau(t)
	attendu := 0
	for k := range m.noms {
		if m.aLeTag(k, "rose") || strings.Contains(m.noms[k], "rose") || strings.Contains(strings.ToLower(m.titres[k]), "rose") {
			attendu++
		}
	}
	appuyer(m, "/", "r", "o", "s", "e", "entrée")
	if len(m.vue) != attendu || attendu < 2 {
		t.Errorf("/rose : %d thèmes, attendu %d (tags compris)", len(m.vue), attendu)
	}
}

// TestRienNeCorrespond : « aucun thème » dit comment en retrouver.
func TestRienNeCorrespond(t *testing.T) {
	m, _ := nouveau(t)
	m.Update(tea.WindowSizeMsg{Width: compacteLargeur - 10, Height: compacteHauteur - 4}) // la liste a toute la largeur
	appuyer(m, "→", "/", "m", "i", "a", "m", "i", "entrée")
	if len(m.vue) != 0 {
		t.Skip("Miami est populaire : choisir un autre exemple")
	}
	e := ecran(m)
	for _, x := range []string{"aucun thème", "Échap : effacer /miami", "←→ : un autre filtre (1 dans « tous »)"} {
		if !strings.Contains(e, x) {
			t.Errorf("%q absent :\n%s", x, e)
		}
	}
}

// TestAideComplete : ? montre toutes les touches de la table, Échap la referme.
func TestAideComplete(t *testing.T) {
	m, _ := nouveau(t)
	if !strings.Contains(ecran(m), "? aide") {
		t.Error("le pied doit rappeler ? aide")
	}
	appuyer(m, "?")
	e := ecran(m)
	for _, r := range raccourcis {
		if r.mode >= enStyle { // l'aide de l'onglet Statusline : TestAideStyle
			continue
		}
		if !strings.Contains(e, r.effet) && !strings.Contains(e, strings.Fields(r.effet)[0]) {
			t.Errorf("%s (%s) absent de l'aide", r.nom, r.effet)
		}
	}
	for _, md := range []mode{enListe, enRecherche, enQuestion} {
		if !strings.Contains(e, nomsModes[md]) {
			t.Errorf("section %s absente de l'aide", nomsModes[md])
		}
	}
	if appuyer(m, "échap") || m.aide {
		t.Error("Échap doit refermer l'aide sans quitter")
	}
}
