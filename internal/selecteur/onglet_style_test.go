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
	touches["espace"] = tea.KeySpace
}

// ongletStyle : le sélecteur ouvert sur l'onglet Statusline (comme « agentline style »).
func ouvrirStyle(t *testing.T) (*Modele, string) {
	t.Helper()
	m, config := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "main")
	m.OuvrirStyle()
	return m, config
}

// aller : la rangée de l'item dans le groupe.
func aller(t *testing.T, m *Modele, g int, item string) {
	t.Helper()
	m.os.viser(g, item)
	if r := m.os.courante(); r.groupe != g || r.item != item {
		t.Fatalf("rangée %d/%s introuvable", g, item)
	}
}

func TestOngletStyleBascule(t *testing.T) {
	m, _ := nouveau(t)
	appuyer(m, "tab")
	if m.onglet != 1 || m.mode() != enStyle {
		t.Fatal("Tab doit ouvrir l'onglet Statusline")
	}
	e := ecran(m)
	for _, x := range []string{"Ligne 1", "Ligne 2", "Ligne SSH", "Aperçu · complète", "☑ model", "+ effort", "SSH "} {
		if !strings.Contains(e, x) {
			t.Errorf("%q absent :\n%s", x, e)
		}
	}
	appuyer(m, "tab")
	if m.onglet != 0 {
		t.Error("Tab revient à l'onglet Thèmes")
	}
}

func TestOngletStyleItems(t *testing.T) {
	m, _ := ouvrirStyle(t)
	aller(t, m, gLigne1, "session")
	appuyer(m, "espace")
	if !m.os.st.Masques()["session"] || !strings.Contains(ecran(m), "☐ session") {
		t.Error("Espace doit masquer l'item")
	}
	appuyer(m, "espace")
	if m.os.st.Masques()["session"] {
		t.Error("Espace de nouveau l'affiche")
	}
	// déplacer : dans la ligne, puis d'une ligne à l'autre
	aller(t, m, gLigne1, "effort")
	appuyer(m, "K")
	if v := m.os.st.Valeur("line1"); !strings.HasPrefix(v, "effort model fast |") {
		t.Errorf("K : %q", v)
	}
	aller(t, m, gLigne2, "ctx")
	appuyer(m, "maj+↑")
	if a, b := m.os.st.Valeur("line1"), m.os.st.Valeur("line2"); !strings.HasSuffix(a, "| session | ctx") || b != "5h | 7d | cost" {
		t.Errorf("ctx vers la ligne 1 : %q / %q", a, b)
	}
	if r := m.os.courante(); r.groupe != gLigne1 || r.item != "ctx" {
		t.Errorf("le curseur suit l'item : %+v", r)
	}
	appuyer(m, "J")
	if b := m.os.st.Valeur("line2"); b != "ctx | 5h | 7d | cost" {
		t.Errorf("ctx revient : %q", b)
	}
	// coller, détacher
	aller(t, m, gLigne2, "5h")
	appuyer(m, "g")
	if b := m.os.st.Valeur("line2"); b != "ctx 5h | 7d | cost" {
		t.Errorf("g : %q", b)
	}
	appuyer(m, "g")
	if m.os.st.Regle("line2") {
		t.Errorf("g de nouveau : %q", m.os.st.Valeur("line2"))
	}
	// ligne SSH : retirer, puis l'item absent revient à la fin
	aller(t, m, gCourt, "ctx")
	appuyer(m, "espace")
	if c := m.os.st.Valeur("short"); c != "model effort | folder git | cost | 5h" || !strings.Contains(ecran(m), "☐ ctx absent") {
		t.Errorf("retirer de la ligne SSH : %q", c)
	}
	aller(t, m, gCourt, "ctx")
	appuyer(m, "espace")
	if c := m.os.st.Valeur("short"); c != "model effort | folder git | cost | 5h | ctx" {
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
	appuyer(m, "→")
	if m.mode() != enStyleOptions || !strings.Contains(ecran(m), "Options · ctx") {
		t.Fatalf("→ ouvre les options :\n%s", ecran(m))
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
	appuyer(m, "↓", "↓", "entrée")
	if m.mode() != enStyleSaisie {
		t.Fatal("Entrée sur un libellé : saisie")
	}
	appuyer(m, "⌫", "⌫", "⌫", "c", "o", "n", "t", "entrée")
	if v := m.os.st.Valeur("ctx.label"); v != "cont" || m.mode() != enStyleOptions {
		t.Errorf("libellé : %q", v)
	}
	appuyer(m, "entrée", "🚀", "entrée")
	if v := m.os.st.Valeur("ctx.label"); v != "cont" || m.mode() != enStyleSaisie || !strings.Contains(ecran(m), "2 colonnes") {
		t.Errorf("glyphe large : %q, mode %d\n%s", v, m.mode(), ecran(m))
	}
	appuyer(m, "échap", "échap")
	if m.mode() != enStyle {
		t.Error("Échap ferme la saisie, puis les options")
	}
	// général : les icônes
	aller(t, m, gGeneral, "")
	appuyer(m, "→", "→")
	if v := m.os.st.Valeur("icons"); v != "text" {
		t.Errorf("icons : %q", v)
	}
	// un item sans option le dit
	appuyer(m, "échap")
	aller(t, m, gLigne1, "agent")
	appuyer(m, "→")
	if m.mode() != enStyle || !strings.Contains(ecran(m), "pas d'option") {
		t.Error("agent n'a pas d'option")
	}
}

func TestOngletStyleGarder(t *testing.T) {
	m, config := ouvrirStyle(t)
	aller(t, m, gLigne1, "pr")
	appuyer(m, "espace", "entrée")
	if m.mode() != enStyleQuestion || m.humeur() != "danse" {
		t.Fatal("Entrée : la question, et Clawd danse")
	}
	e := ecran(m)
	if !strings.Contains(e, "[d] partout") || !strings.Contains(e, "[p] ce projet (bientôt)") {
		t.Errorf("panneau :\n%s", e)
	}
	if appuyer(m, "p") {
		t.Fatal("p n'est pas encore disponible")
	}
	if !appuyer(m, "d") {
		t.Fatal("d garde et quitte")
	}
	b, err := os.ReadFile(filepath.Join(config, style.Fichier))
	if err != nil || !strings.Contains(string(b), "hide=pr\n") {
		t.Errorf("fichier : %q %v", b, err)
	}
	if !strings.HasPrefix(m.Message(), "✓ Style de la statusline gardé (1 réglage") {
		t.Errorf("message : %q", m.Message())
	}
}

func TestOngletStyleQuitterSansGarder(t *testing.T) {
	m, config := ouvrirStyle(t)
	if !appuyer(m, "q") {
		t.Fatal("sans modification, q quitte")
	}
	m, config = ouvrirStyle(t)
	appuyer(m, "espace")
	if appuyer(m, "q") {
		t.Fatal("modifications non gardées : q avertit d'abord")
	}
	if !strings.Contains(ecran(m), "non gardées") {
		t.Error("l'avertissement doit s'afficher")
	}
	appuyer(m, "↓") // une autre touche : l'avertissement tombe
	if appuyer(m, "échap") {
		t.Fatal("l'avertissement vaut pour la touche suivante seulement")
	}
	if !appuyer(m, "échap") {
		t.Fatal("la seconde fois, Échap quitte")
	}
	if !strings.Contains(m.Message(), "pas été gardées") {
		t.Errorf("message : %q", m.Message())
	}
	if _, err := os.Stat(filepath.Join(config, style.Fichier)); !os.IsNotExist(err) {
		t.Error("rien ne doit être écrit")
	}
	// depuis l'onglet Thèmes aussi
	m, _ = ouvrirStyle(t)
	appuyer(m, "espace", "tab")
	if appuyer(m, "q") || !appuyer(m, "q") {
		t.Error("onglet Thèmes : avertir, puis quitter")
	}
}

// TestOngletStyleApercu : v fait défiler les aperçus ; le thème est celui survolé dans l'onglet Thèmes.
func TestOngletStyleApercu(t *testing.T) {
	m, _ := ouvrirStyle(t)
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
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
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

// TestResponsiveStyle : à chaque taille et dans chaque état, l'onglet tient dans le terminal.
func TestResponsiveStyle(t *testing.T) {
	for _, tl := range tailles {
		w, h := tl[0], tl[1]
		for _, etat := range [][]string{nil, {"→"}, {"entrée"}, {"?"}, {"fin"}, {"espace", "q"}} {
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
				if r := m.os.courante(); !m.aide && !r.titre && !strings.Contains(ecran(m), "›") {
					t.Errorf("%dx%d %v : la rangée choisie n'est pas visible", w, h, etat)
				}
			}
		}
	}
}
