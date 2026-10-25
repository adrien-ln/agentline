package selecteur

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/style"
)

func init() {
	touches["tab"] = tea.KeyTab
	touches["maj+↑"] = tea.KeyShiftUp
	touches["maj+↓"] = tea.KeyShiftDown
	touches["maj+←"] = tea.KeyShiftLeft
	touches["maj+→"] = tea.KeyShiftRight
	touches["suppr"] = tea.KeyDelete
}

// ongletStyle : le sélecteur ouvert sur l'onglet Statusline (comme « agentline style »).
func ouvrirStyle(t *testing.T) (*Modele, string) {
	t.Helper()
	m, config := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "main")
	m.OuvrirStyle()
	return m, config
}

// aller : l'item dans cette ligne.
func aller(t *testing.T, m *Modele, g int, item string) {
	t.Helper()
	m.os.viser(g, item)
	if x, ok := m.os.courant(); !ok || m.os.g != g || x.nom != item {
		t.Fatalf("item %d/%s introuvable", g, item)
	}
}

// surligne : le texte visible surligné (souligné) sur l'aperçu, hors barre des onglets.
func surligne(m *Modele) string {
	for _, l := range strings.Split(m.View(), "\n") {
		if !strings.Contains(l, "│") { // les lignes du cadre de l'aperçu
			continue
		}
		if i := strings.Index(l, souligne); i >= 0 {
			reste := l[i:]
			if j := strings.Index(reste, finSurligne); j >= 0 {
				reste = reste[:j]
			}
			return couleurs.ReplaceAllString(reste, "")
		}
	}
	return ""
}

// rangee : place le curseur du menu sur la rangée de ce nom (une option, ou coller, retirer).
func rangee(t *testing.T, m *Modele, nom string) {
	t.Helper()
	for k, r := range m.os.rangeesMenu() {
		if r.cle.Nom == nom || r.action == nom {
			m.os.opt = k
			return
		}
	}
	t.Fatalf("rangée %s absente du menu", nom)
}

func TestOngletStyleBascule(t *testing.T) {
	m, _ := nouveau(t)
	appuyer(m, "tab")
	if m.onglet != 1 || m.mode() != enStyle {
		t.Fatal("Tab doit ouvrir l'onglet Statusline")
	}
	e := ecran(m)
	for _, x := range []string{"Aperçu · complète", "▲ model", "Ligne 1", "#12 en revue"} {
		if !strings.Contains(e, x) {
			t.Errorf("%q absent :\n%s", x, e)
		}
	}
	if strings.Contains(e, "SSH Opus") {
		t.Errorf("sans ssh.short, pas de ligne SSH :\n%s", e)
	}
	m.os.st.Definir("ssh.short", "yes")
	m.os.changer()
	appuyer(m, "↑") // de la ligne 1, ↑ passe à la dernière : la ligne SSH
	if e := ecran(m); m.os.g != gCourt || !strings.Contains(e, "SSH Opus") || !strings.Contains(e, "Ligne SSH") {
		t.Errorf("ssh.short=yes : la ligne SSH, qu'on peut choisir :\n%s", e)
	}
	appuyer(m, "tab")
	if m.onglet != 0 {
		t.Error("Tab revient à l'onglet Thèmes")
	}
}

// TestOngletStyleNavigation : ←→ d'item en item (et de ligne en ligne en bout de ligne), ↑↓ de ligne en
// ligne ; l'item choisi est surligné sur l'aperçu même.
func TestOngletStyleNavigation(t *testing.T) {
	m, _ := ouvrirStyle(t)
	if x, _ := m.os.courant(); x.nom != "model" || !strings.HasPrefix(surligne(m), "Opus") {
		t.Fatalf("au départ, model surligné : %q %q", x.nom, surligne(m))
	}
	appuyer(m, "→", "→", "→")
	if x, _ := m.os.courant(); x.nom != "folder" || !strings.Contains(surligne(m), "mon-projet") {
		t.Errorf("→ : folder surligné : %q %q", x.nom, surligne(m))
	}
	appuyer(m, "←")
	if x, _ := m.os.courant(); x.nom != "fast" {
		t.Errorf("← : %q", x.nom)
	}
	aller(t, m, gLigne1, "session")
	appuyer(m, "→")
	if x, _ := m.os.courant(); m.os.g != gLigne2 || x.nom != "ctx" || !strings.HasPrefix(surligne(m), "ctx") {
		t.Errorf("→ en bout de ligne 1 : ctx, ligne 2 : %d %q %q", m.os.g, x.nom, surligne(m))
	}
	appuyer(m, "↑")
	if m.os.g != gLigne1 {
		t.Errorf("↑ : ligne %d", m.os.g)
	}
	// un item vide dans l'aperçu le dit (pas de coût hors API)
	aller(t, m, gLigne2, "cost")
	if e := ecran(m); !strings.Contains(e, "vide dans cet aperçu") || surligne(m) != "" {
		t.Errorf("cost vide :\n%s", e)
	}
}

func TestOngletStyleItems(t *testing.T) {
	m, _ := ouvrirStyle(t)
	// déplacer : dans la ligne, puis d'une ligne à l'autre
	aller(t, m, gLigne1, "effort")
	appuyer(m, "H")
	if v := m.os.st.Valeur("line1"); !strings.HasPrefix(v, "effort model fast |") {
		t.Errorf("H : %q", v)
	}
	appuyer(m, "maj+→")
	if m.os.st.Regle("line1") {
		t.Errorf("Maj+→ le remet : %q", m.os.st.Valeur("line1"))
	}
	aller(t, m, gLigne2, "ctx")
	appuyer(m, "maj+↑")
	if a, b := m.os.st.Valeur("line1"), m.os.st.Valeur("line2"); !strings.HasSuffix(a, "| session | ctx") || b != "5h | 7d | cost" {
		t.Errorf("ctx vers la ligne 1 : %q / %q", a, b)
	}
	if x, _ := m.os.courant(); m.os.g != gLigne1 || x.nom != "ctx" {
		t.Errorf("le choix suit l'item : %d %q", m.os.g, x.nom)
	}
	appuyer(m, "J")
	if b := m.os.st.Valeur("line2"); b != "ctx | 5h | 7d | cost" {
		t.Errorf("ctx revient : %q", b)
	}
	// retirer, puis ajouter
	aller(t, m, gLigne1, "session")
	appuyer(m, "x")
	if a := m.os.st.Valeur("line1"); strings.Contains(a, "session") {
		t.Errorf("x : %q", a)
	}
	aller(t, m, gLigne1, "pr")
	appuyer(m, "+")
	if m.mode() != enStyleAjout || !strings.Contains(ecran(m), "Ajouter à la ligne 1") || !strings.Contains(ecran(m), "session") {
		t.Fatalf("+ : la liste des items à ajouter :\n%s", ecran(m))
	}
	rangee(t, m, "session")
	appuyer(m, "entrée")
	if a := m.os.st.Valeur("line1"); !strings.Contains(a, "| pr | session |") || m.mode() != enStyle {
		t.Errorf("ajouter après pr : %q", a)
	}
	// coller, détacher, par le menu
	aller(t, m, gLigne2, "5h")
	appuyer(m, "entrée")
	rangee(t, m, actionColler)
	appuyer(m, "→")
	if b := m.os.st.Valeur("line2"); b != "ctx 5h | 7d | cost" || !strings.Contains(ecran(m), "collé au précédent") {
		t.Errorf("coller : %q", b)
	}
	appuyer(m, "entrée")
	if b := m.os.st.Valeur("line2"); b != "ctx | 5h | 7d | cost" {
		t.Errorf("détacher : %q", b)
	}
	// retirer par le menu
	rangee(t, m, actionRetirer)
	appuyer(m, "entrée")
	if b := m.os.st.Valeur("line2"); b != "ctx | 7d | cost" || m.mode() != enStyle {
		t.Errorf("retirer par le menu : %q", b)
	}
	// la ligne SSH (avec ssh.short=yes) : ses items à elle
	m.os.st.Definir("ssh.short", "yes")
	aller(t, m, gCourt, "ctx")
	appuyer(m, "suppr")
	if c := m.os.st.Valeur("short"); c != "model effort | folder git | cost | 5h" {
		t.Errorf("retirer de la ligne SSH : %q", c)
	}
	appuyer(m, "+")
	rangee(t, m, "ctx")
	appuyer(m, "entrée")
	if c := m.os.st.Valeur("short"); !strings.Contains(c, "ctx") {
		t.Errorf("remettre dans la ligne SSH : %q", c)
	}
	appuyer(m, "R")
	if !m.os.st.EstDefaut() {
		t.Errorf("R : %q", m.os.st.Texte())
	}
}

func TestOngletStyleOptions(t *testing.T) {
	m, _ := ouvrirStyle(t)
	aller(t, m, gLigne2, "ctx")
	appuyer(m, "entrée")
	if m.mode() != enStyleMenu || !strings.Contains(ecran(m), "Options · ctx") {
		t.Fatalf("Entrée ouvre le menu :\n%s", ecran(m))
	}
	avant := ecran(m)
	appuyer(m, "→") // ctx.bar : lines -> blocks
	if v := m.os.st.Valeur("ctx.bar"); v != "blocks" {
		t.Errorf("→ : %q", v)
	}
	if e := ecran(m); e == avant || !strings.Contains(e, "█") {
		t.Error("l'aperçu doit suivre en direct")
	}
	appuyer(m, "↓", "→", "→") // ctx.width 8 -> 10
	if v := m.os.st.Valeur("ctx.width"); v != "10" {
		t.Errorf("ctx.width : %q", v)
	}
	appuyer(m, "r")
	if m.os.st.Regle("ctx.width") {
		t.Error("r remet l'option à son défaut")
	}
	// saisie : un libellé, puis un glyphe trop large refusé
	rangee(t, m, "ctx.label")
	appuyer(m, "entrée")
	if m.mode() != enStyleSaisie {
		t.Fatal("Entrée sur un libellé : saisie")
	}
	appuyer(m, "⌫", "⌫", "⌫", "c", "o", "n", "t", "entrée")
	if v := m.os.st.Valeur("ctx.label"); v != "cont" || m.mode() != enStyleMenu {
		t.Errorf("libellé : %q", v)
	}
	appuyer(m, "entrée", "🚀", "entrée")
	if v := m.os.st.Valeur("ctx.label"); v != "cont" || m.mode() != enStyleSaisie || !strings.Contains(ecran(m), "2 colonnes") {
		t.Errorf("glyphe large : %q, mode %d\n%s", v, m.mode(), ecran(m))
	}
	appuyer(m, "échap", "échap")
	if m.mode() != enStyle {
		t.Error("Échap ferme la saisie, puis le menu")
	}
	// général : les icônes
	appuyer(m, "g")
	if !strings.Contains(ecran(m), "Options · générales") {
		t.Fatalf("g : les options générales :\n%s", ecran(m))
	}
	rangee(t, m, "icons")
	appuyer(m, "→")
	if v := m.os.st.Valeur("icons"); v != "text" {
		t.Errorf("icons : %q", v)
	}
	// un item sans option : le menu n'a que coller et retirer
	appuyer(m, "échap")
	aller(t, m, gLigne1, "agent")
	appuyer(m, "entrée")
	if l := m.os.rangeesMenu(); len(l) != 2 || l[0].action != actionColler || l[1].action != actionRetirer {
		t.Errorf("agent : %+v", l)
	}
}

func TestOngletStyleGarder(t *testing.T) {
	m, config := ouvrirStyle(t)
	aller(t, m, gLigne1, "pr")
	appuyer(m, "x")
	if e := aplati(ecran(m)); !strings.Contains(e, "Non enregistré : style de la statusline (1 réglage)") {
		t.Errorf("x : la barre doit le dire :\n%s", ecran(m))
	}
	if appuyer(m, "s") || m.humeur() != "danse" {
		t.Fatal("s enregistre sans fermer, et Pip danse")
	}
	b, err := os.ReadFile(filepath.Join(config, style.Fichier))
	if err != nil || !strings.Contains(string(b), "line1=model effort fast | folder git worktree | agent | style | session\n") {
		t.Errorf("fichier : %q %v", b, err)
	}
	if m.os.modifie() || !strings.Contains(aplati(ecran(m)), "✓ Enregistré : style de la statusline (1 réglage)") {
		t.Errorf("après s :\n%s", ecran(m))
	}
	if !appuyer(m, "q") || !strings.HasPrefix(m.Message(), "✓ Enregistré : style de la statusline (1 réglage)") {
		t.Errorf("q : %q", m.Message())
	}
}

func TestOngletStyleQuitterSansGarder(t *testing.T) {
	m, _ := ouvrirStyle(t)
	if !appuyer(m, "q") {
		t.Fatal("sans modification, q quitte")
	}
	m, config := ouvrirStyle(t)
	appuyer(m, "x")
	if appuyer(m, "q") || m.mode() != enQuitter {
		t.Fatal("modifications non enregistrées : q demande d'abord")
	}
	if appuyer(m, "échap") || m.mode() != enStyle {
		t.Fatal("Échap : retour aux items")
	}
	if appuyer(m, "u") || m.os.modifie() {
		t.Fatal("u : le style revient à celui du fichier")
	}
	appuyer(m, "x")
	if !appuyer(m, "q", "n") || !strings.Contains(m.Message(), "Non enregistré : style de la statusline") {
		t.Errorf("q puis n : %q", m.Message())
	}
	if _, err := os.Stat(filepath.Join(config, style.Fichier)); !os.IsNotExist(err) {
		t.Error("rien ne doit être écrit")
	}
}

// TestEnregistrerLesDeuxOnglets : un thème choisi dans un onglet et le style réglé dans l'autre partent
// ensemble avec s, depuis n'importe quel onglet ; s dans la question de sortie enregistre puis quitte.
func TestEnregistrerLesDeuxOnglets(t *testing.T) {
	m, config := ouvrirStyle(t)
	appuyer(m, "x", "tab", "↓", "entrée", "d", "tab")
	nom := m.noms[m.i]
	if a := m.enAttente(); len(a) != 2 {
		t.Fatalf("un thème et le style en attente : %v", a)
	}
	if !appuyer(m, "q", "s") {
		t.Fatal("q puis s : enregistrer et quitter")
	}
	if b, _ := os.ReadFile(filepath.Join(config, "defaut")); string(b) != nom+"\n" {
		t.Errorf("defaut = %q, attendu %s", b, nom)
	}
	if _, err := os.Stat(filepath.Join(config, style.Fichier)); err != nil {
		t.Errorf("le style doit être écrit : %v", err)
	}
	if msg := m.Message(); !strings.Contains(msg, "thème ") || !strings.Contains(msg, "style de la statusline") || strings.Contains(msg, "Non enregistré") {
		t.Errorf("message : %q", msg)
	}
}

// TestOngletStyleApercu : v fait défiler les aperçus ; le thème est celui survolé dans l'onglet Thèmes.
func TestOngletStyleApercu(t *testing.T) {
	m, _ := ouvrirStyle(t)
	m.os.st.Definir("ssh.short", "yes") // l'aperçu SSH n'existe qu'avec la ligne courte
	for _, n := range nomsApercus[1:] {
		appuyer(m, "v")
		if !strings.Contains(ecran(m), "Aperçu · "+n) {
			t.Errorf("aperçu %s absent", n)
		}
	}
	appuyer(m, "v")
	if e := ecran(m); !strings.Contains(e, "Aperçu · complète") || !strings.Contains(e, "#12 en revue") {
		t.Error("retour à l'aperçu complet")
	}
	avant := m.View()
	appuyer(m, "tab", "↓", "tab")
	if m.View() == avant {
		t.Error("l'aperçu doit prendre le thème survolé")
	}
}

func TestAideStyle(t *testing.T) {
	m, _ := ouvrirStyle(t)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 60})
	appuyer(m, "?")
	e := ecran(m)
	for _, r := range raccourcis {
		if r.mode >= enStyle && !strings.Contains(e, r.effet) && !strings.Contains(e, strings.Fields(r.effet)[0]) {
			t.Errorf("%s (%s) absent de l'aide", r.nom, r.effet)
		}
	}
	for _, md := range modesAide[1] {
		if !strings.Contains(e, nomsModes[md]) {
			t.Errorf("section %s absente", nomsModes[md])
		}
	}
}

// TestResponsiveStyle : à chaque taille et dans chaque état, l'onglet tient dans le terminal, et l'item
// choisi est nommé sous l'aperçu.
func TestResponsiveStyle(t *testing.T) {
	for _, tl := range tailles {
		w, h := tl[0], tl[1]
		for _, etat := range [][]string{nil, {"entrée"}, {"g"}, {"+"}, {"?"}, {"→", "→", "→"}, {"x", "q"}} {
			m, _ := ouvrirStyle(t)
			m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			appuyer(m, etat...)
			for _, instant := range []int{0, 9300} {
				m.t, m.derniereTouche = instant, instant
				brut := m.View()
				l := strings.Split(brut, "\n")
				if len(l) > max(h, hauteurMin) {
					t.Errorf("%dx%d %v : %d lignes", w, h, etat, len(l))
				}
				for k, x := range l {
					if n := largeur(x); n > max(w, largeurMin) {
						t.Errorf("%dx%d %v : ligne %d de %d colonnes", w, h, etat, k, n)
					}
				}
				if x, _ := m.os.courant(); !m.aide && !strings.Contains(ecran(m), "▲ "+x.nom) {
					t.Errorf("%dx%d %v : l'item choisi n'est pas nommé\n%s", w, h, etat, ecran(m))
				}
			}
		}
	}
}

// TestOptionsAlignees : dans le menu, le nom de l'option choisie, ceux des autres et l'aide commencent sur
// la même colonne, le curseur et le titre deux colonnes avant, dans chaque disposition.
func TestOptionsAlignees(t *testing.T) {
	for _, c := range []struct{ w, h int }{{150, 45}, {100, 40}, {58, 30}} {
		m, _ := ouvrirStyle(t)
		m.Update(tea.WindowSizeMsg{Width: c.w, Height: c.h})
		aller(t, m, gLigne2, "ctx")
		appuyer(m, "entrée")
		colonne := func(quoi string) int {
			t.Helper()
			for _, l := range strings.Split(ecran(m), "\n") {
				if i := strings.Index(l, quoi); i >= 0 {
					return largeur(l[:i])
				}
			}
			t.Fatalf("%dx%d : %q absent\n%s", c.w, c.h, quoi, ecran(m))
			return 0
		}
		titre, curseur, choisie, autre := colonne("Options · ctx"), colonne("› ctx.bar"), colonne("ctx.bar"), colonne("ctx.width")
		if curseur != titre || choisie != curseur+2 || autre != choisie {
			t.Errorf("%dx%d : titre %d, curseur %d, option choisie %d, autre option %d", c.w, c.h, titre, curseur, choisie, autre)
		}
	}
}

// TestSurligner : la marque couvre les caractères visibles demandés, remis après chaque séquence de la
// portion, et les séquences (CSI, OSC 8) ne comptent pas comme des colonnes.
func TestSurligner(t *testing.T) {
	marquer := func(s string) string {
		s = strings.ReplaceAll(s, souligne, "⟦")
		s = strings.ReplaceAll(s, finSurligne, "⟧")
		return couleurs.ReplaceAllString(s, "")
	}
	for _, c := range []struct {
		s          string
		debut, fin int
		attendu    string
	}{
		{"abcdef", 1, 3, "a⟦bc⟧def"},
		{"ab\x1b[31mcd\x1b[0mef", 1, 4, "a⟦b⟦cd⟧ef"},
		{"abc", 1, 3, "a⟦bc⟧"},
		{"a\x1b]8;;https://x\x1b\\#12\x1b]8;;\x1b\\b", 1, 4, "a⟦#12⟧b"},
	} {
		obtenu := marquer(surligner(c.s, c.debut, c.fin, souligne))
		obtenu = strings.NewReplacer("\x1b]8;;https://x\x1b\\", "", "\x1b]8;;\x1b\\", "").Replace(obtenu)
		if obtenu != c.attendu {
			t.Errorf("%q [%d:%d] : %q, attendu %q", c.s, c.debut, c.fin, obtenu, c.attendu)
		}
	}
}

// TestStyleCentre : sur un grand écran, l'aperçu descend vers le milieu, et ne bouge pas quand un menu
// s'ouvre.
func TestStyleCentre(t *testing.T) {
	ligneCadre := func(m *Modele) int {
		for k, l := range strings.Split(ecran(m), "\n") {
			if strings.Contains(l, "╭─ Aperçu") {
				return k
			}
		}
		t.Fatalf("pas d'aperçu :\n%s", ecran(m))
		return 0
	}
	m, _ := ouvrirStyle(t)
	m.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	petit := ligneCadre(m)
	m.Update(tea.WindowSizeMsg{Width: 150, Height: 60})
	grand := ligneCadre(m)
	if grand <= petit {
		t.Errorf("le bloc doit descendre sur un grand écran : %d puis %d", petit, grand)
	}
	for _, menu := range [][]string{{"entrée"}, {"g"}, {"x", "+"}} {
		appuyer(m, menu...)
		if k := ligneCadre(m); k != grand {
			t.Errorf("%v : l'aperçu passe de la ligne %d à %d", menu, grand, k)
		}
		appuyer(m, "échap")
	}
}
