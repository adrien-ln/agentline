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
	if appuyer(m, "entrée", "d", "s") {
		t.Fatal("Entrée, d puis s enregistrent sans fermer le sélecteur")
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
	if m.choix != "" {
		t.Error("Entrée sans thème ne doit rien choisir")
	}
	if appuyer(m, "échap") || m.recherche != "" || len(m.vue) != len(m.noms) {
		t.Error("Échap doit d'abord effacer la recherche, sans quitter")
	}
	if appuyer(m, "échap") || !appuyer(m, "q") || m.Message() != "Rien n'a changé : Tokyo Night reste en vigueur (intégré)." {
		t.Errorf("Échap ne quitte pas, q si : %q", m.Message())
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
	appuyer(m, "↑", "entrée", "p")
	nom, titre := m.noms[m.i], m.titres[m.i]
	if m.choix != nom || !strings.Contains(aplati(ecran(m)), "Non enregistré : thème "+titre+" pour ce projet") {
		t.Fatalf("Entrée puis p : le choix doit attendre dans la barre :\n%s", ecran(m))
	}
	if _, err := os.Stat(filepath.Join(config, "projets")); err == nil {
		t.Fatal("rien ne doit être écrit avant s")
	}
	if appuyer(m, "s") {
		t.Fatal("s enregistre sans fermer")
	}
	b, _ := os.ReadFile(filepath.Join(config, "projets"))
	if !strings.HasSuffix(string(b), "\t"+nom+"\n") || !strings.Contains(aplati(ecran(m)), "✓ Enregistré : thème "+titre) || m.aEnregistrer() {
		t.Errorf("projets = %q, écran :\n%s", b, ecran(m))
	}
	if m.avant != nom {
		t.Errorf("le thème en vigueur doit suivre ce qui est enregistré : %s", m.avant)
	}
	if !appuyer(m, "q") {
		t.Fatal("tout est enregistré : q quitte sans question")
	}
	if msg := m.Message(); !strings.HasPrefix(msg, "✓ Enregistré : thème "+titre+" pour ce projet") || !strings.Contains(msg, "prochain rafraîchissement") || strings.Contains(msg, "⚠") {
		t.Errorf("message de sortie : %q", msg)
	}
}

// aplati : l'écran sans retours à la ligne ni espaces répétés (la barre et la question se replient).
func aplati(e string) string { return strings.Join(strings.Fields(e), " ") }

// TestBarre80Colonnes : à 80×24, avec un vrai remote, la barre du brouillon et la question de sortie se
// replient sans rien tronquer ; la question remplace le pied de page.
func TestBarre80Colonnes(t *testing.T) {
	m, _ := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "feat/interface")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	appuyer(m, "↓", "entrée", "b")
	verifier := func(quoi string, attendus ...string) {
		e := ecran(m)
		for _, x := range attendus {
			if !strings.Contains(aplati(e), x) {
				t.Errorf("%s : %q absent :\n%s", quoi, x, e)
			}
		}
		for k, l := range strings.Split(m.View(), "\n") {
			if largeur(l) > 80 {
				t.Errorf("%s : ligne %d de %d colonnes", quoi, k, largeur(l))
			}
		}
	}
	verifier("barre", "Non enregistré : thème", "pour la branche feat/interface")
	if appuyer(m, "q") {
		t.Fatal("un thème choisi : q demande d'abord")
	}
	verifier("question", "Quitter : s enregistrer et quitter n quitter sans enregistrer", "revenir")
	if strings.Contains(ecran(m), "↑↓ parcourir") {
		t.Error("la question remplace le pied de page")
	}
	if appuyer(m, "échap") || m.questionQuitter {
		t.Error("Échap : retour à la liste, sans quitter")
	}
	if !appuyer(m, "q", "n") || !strings.Contains(m.Message(), "Non enregistré : thème") {
		t.Errorf("q puis n : quitter sans enregistrer, et le dire : %q", m.Message())
	}
}

// TestEchapParCrans : Échap remonte d'un cran (recherche, puis filtre) sans jamais quitter ; q quitte.
func TestEchapParCrans(t *testing.T) {
	m, _ := nouveau(t)
	appuyer(m, "→", "/", "z", "z", "z", "entrée")
	if m.recherche != "zzz" || m.f == 0 {
		t.Fatalf("préparation : recherche %q, filtre %d", m.recherche, m.f)
	}
	appuyer(m, "échap")
	if m.recherche != "" || m.f == 0 {
		t.Errorf("1er Échap : la recherche seule doit être effacée (recherche %q, filtre %d)", m.recherche, m.f)
	}
	if appuyer(m, "échap") || m.f != 0 {
		t.Errorf("2e Échap : le filtre doit revenir à « tous » (%s)", m.filtres[m.f])
	}
	if appuyer(m, "échap") {
		t.Error("3e Échap : rien (q quitte)")
	}
	for _, avant := range [][]string{{"?"}, {"→"}, {"tab"}} {
		m, _ := nouveau(t)
		appuyer(m, avant...)
		if !appuyer(m, "q") {
			t.Errorf("q après %v doit quitter", avant)
		}
	}
	m, _ = nouveau(t)
	if appuyer(m, "tab", "échap") {
		t.Error("onglet Statusline : Échap ne quitte pas")
	}
	m, _ = nouveau(t)
	if appuyer(m, "/", "q") || m.recherche != "q" {
		t.Error("en recherche, q est une lettre comme les autres")
	}
}

// TestAvertissementMasque : la barre signale le choix qui restera masqué ici, et le message de sortie aussi.
func TestAvertissementMasque(t *testing.T) {
	d := depotFictif(t, "git@gitlab.com:a/b.git", "main")
	m, _ := ouvrir(t, d, nil, func(s theme.Sources) { s.RegleBranche("main", "nord") })
	if m.portee != theme.SourceBranche {
		t.Errorf("portée de départ : %s, attendu celle du thème en vigueur (branche)", m.portee)
	}
	appuyer(m, "↓", "entrée", "p")
	if e := aplati(ecran(m)); !strings.Contains(e, "⚠ Ici, la branche main (Nord) reste prioritaire.") {
		t.Errorf("avertissement absent ou faux :\n%s", ecran(m))
	}
	appuyer(m, "s", "q")
	if !strings.Contains(m.Message(), "⚠ Ici, la branche main (Nord) reste prioritaire") {
		t.Errorf("message de sortie sans avertissement : %q", m.Message())
	}

	m, _ = ouvrir(t, d, map[string]string{"AGENTLINE_THEME": "dracula"}, nil)
	appuyer(m, "↓", "entrée")
	if e := aplati(ecran(m)); !strings.Contains(e, "AGENTLINE_THEME=dracula (Dracula) reste prioritaire") {
		t.Errorf("AGENTLINE_THEME doit masquer le choix :\n%s", ecran(m))
	}
	m, _ = ouvrir(t, d, nil, nil)
	appuyer(m, "↓", "entrée")
	if strings.Contains(ecran(m), "⚠") {
		t.Error("rien de réglé : aucun avertissement")
	}
}

// TestBrancheEtRetirer : b enregistre le thème pour la branche courante, x retire ce qui s'applique ici.
func TestBrancheEtRetirer(t *testing.T) {
	d := depotFictif(t, "git@gitlab.com:a/b.git", "feat/x")
	m, config := ouvrir(t, d, nil, nil)
	appuyer(m, "↓")
	nom := m.noms[m.i]
	if appuyer(m, "entrée", "b", "s") {
		t.Fatal("s enregistre sans fermer")
	}
	if b, _ := os.ReadFile(filepath.Join(config, "branches")); string(b) != "feat/x\t"+nom+"\n" {
		t.Errorf("branches = %q", b)
	}
	if !strings.Contains(aplati(ecran(m)), "pour la branche feat/x") {
		t.Errorf("le « ✓ Enregistré » doit dire où :\n%s", ecran(m))
	}
	m, _ = ouvrir(t, d, nil, func(x theme.Sources) {
		x.RegleBranche("feat/x", nom)
		x.Associer(x.ProjetID(filepath.ToSlash(d)), "nord")
	})
	appuyer(m, "x")
	if e := aplati(ecran(m)); !strings.Contains(e, "Non enregistré : retirer la règle feat/x") {
		t.Errorf("x : le retrait attend dans la barre :\n%s", ecran(m))
	}
	appuyer(m, "s", "q")
	if msg := m.Message(); !strings.Contains(msg, "règle feat/x retirée") || m.avant != "nord" {
		t.Errorf("x puis s : la règle de branche part d'abord, le thème du projet reprend : %q (%s)", msg, m.avant)
	}
	m, _ = nouveau(t) // hors Git et sans réglage : ni branche ni réglage à retirer
	if appuyer(m, "b", "x") || m.portee == theme.SourceBranche || m.retrait {
		t.Error("b et x indisponibles : les touches ne font rien")
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
	if !strings.Contains(aplati(ecran(m)), "? aide") {
		t.Error("le pied doit rappeler ? aide")
	}
	appuyer(m, "?")
	e := ecran(m)
	for _, r := range raccourcis {
		if r.mode >= enStyle || r.effet == "" { // l'aide de l'onglet Statusline : TestAideStyle ; sans effet : le pied seul
			continue
		}
		if !strings.Contains(e, r.effet) && !strings.Contains(e, strings.Fields(r.effet)[0]) {
			t.Errorf("%s (%s) absent de l'aide", r.nom, r.effet)
		}
	}
	for _, md := range []mode{enListe, enRecherche, enQuitter} {
		if !strings.Contains(e, nomsModes[md]) {
			t.Errorf("section %s absente de l'aide", nomsModes[md])
		}
	}
	if appuyer(m, "échap") || m.aide {
		t.Error("Échap doit refermer l'aide sans quitter")
	}
}

// TestCapsules : les touches rappelées sont des capsules ; Entrée, Tab, Échap prennent leur glyphe Nerd Font
// avec icons=nerd (le défaut), leur nom avec icons=text ; la ligne surlignée rappelle la touche utile.
func TestCapsules(t *testing.T) {
	m, _ := nouveau(t)
	if c := visible(m.capsule("Entrée")); c != " "+glyphesTouches["Entrée"]+" " {
		t.Errorf("icons=nerd : %q", c)
	}
	if c := visible(m.capsule("Maj+←→ H L")); c != " "+glypheMaj+"←→ "+"  H   L " {
		t.Errorf("Maj+←→ H L : %q", c)
	}
	if !strings.Contains(m.ligneListe(m.i, true, 60), "choisir") {
		t.Error("la ligne surlignée doit rappeler Entrée choisir")
	}
	appuyer(m, "entrée")
	if !strings.Contains(m.ligneListe(m.i, true, 60), "s enregistrer") {
		t.Error("thème choisi : la ligne surlignée doit rappeler s enregistrer")
	}
	m.os.st.Definir("icons", "text")
	if c := visible(m.capsule("Entrée")); c != " Entrée " {
		t.Errorf("icons=text : %q", c)
	}
}
