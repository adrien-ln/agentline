package selecteur

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
)

// tailles : des compactes aux larges, autour des seuils (compacteLargeur, compacteHauteur).
var tailles = [][2]int{{largeurMin, hauteurMin}, {compacteLargeur, compacteHauteur - 2}, {compacteLargeur, compacteHauteur},
	{80, 24}, {90, 30}, {110, 30}, {160, 45}, {220, 60}}

// TestResponsive : à chaque taille et dans chaque mode, l'écran tient dans le terminal et l'occupe ; le
// titre est centré quand il est dessiné ; la mini-fenêtre a des bordures alignées.
func TestResponsive(t *testing.T) {
	for _, tl := range tailles {
		w, h := tl[0], tl[1]
		m, _ := nouveauDepot(t, "git@gitlab.com:adrien-ln/statusline.git", "feat/interface")
		m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		for _, touches := range [][]string{nil, {"entrée"}, {"échap", "?"}, {"échap", "/", "r", "o"}} {
			appuyer(m, touches...)
			for _, instant := range []int{0, 9300, 30000} {
				m.t, m.derniereTouche = instant, instant
				brut := m.View()
				e := strings.Split(couleurs.ReplaceAllString(brut, ""), "\n")
				if len(e) > h {
					t.Errorf("%dx%d %v : %d lignes pour %d de haut", w, h, touches, len(e), h)
				}
				if len(e) < h-1 {
					t.Errorf("%dx%d %v : %d lignes seulement, l'écran n'est pas occupé", w, h, touches, len(e))
				}
				for k, l := range strings.Split(brut, "\n") {
					if n := largeur(l); n > w {
						t.Errorf("%dx%d %v : ligne %d de %d colonnes", w, h, touches, k, n)
					}
				}
				if fl := lignesFenetre(strings.Join(e, "\n")); m.d.fenetre && !m.aide && len(fl) != 1 {
					t.Errorf("%dx%d %v : bordures de la fenêtre décalées %v", w, h, touches, fl)
				}
			}
		}
		appuyer(m, "échap") // efface la recherche
		// le titre est centré, ses trois lignes alignées sur la même colonne (largeur paire ou impaire)
		for _, lw := range []int{w, w + 1} {
			m.Update(tea.WindowSizeMsg{Width: lw, Height: h})
			if !m.d.logo {
				continue
			}
			e := strings.Split(couleurs.ReplaceAllString(m.View(), ""), "\n")
			for k := range logo {
				gauche := len(e[k]) - len(strings.TrimLeft(e[k], " "))
				propre := len(logo[k]) - len(strings.TrimLeft(logo[k], " ")) // la 1re ligne commence par des espaces
				if attendu := (lw-len([]rune(logo[1])))/2 + propre; gauche != attendu {
					t.Errorf("%dx%d : ligne %d du titre décalée de %d colonnes, attendu %d", lw, h, k, gauche, attendu)
				}
			}
		}
		m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		d := m.d
		if d.liste < listeMin && d.agencement != compacte {
			t.Errorf("%dx%d : %d thèmes visibles seulement, la liste passe avant la fenêtre", w, h, d.liste)
		}
		if d.agencement == large && d.liste != d.corps-cadreListe {
			t.Errorf("%dx%d : %d thèmes visibles, la liste doit occuper toute la hauteur", w, h, d.liste)
		}
		if d.agencement == large && d.lw+2+d.ww != w-1 {
			t.Errorf("%dx%d : liste (%d) + fenêtre (%d) n'occupent pas la largeur", w, h, d.lw, d.ww)
		}
	}
}

// TestDispositions : les trois agencements, de part et d'autre des seuils nommés.
func TestDispositions(t *testing.T) {
	cas := []struct {
		w, h        int
		attendu     agencement
		avecFenetre bool
	}{
		{compacteLargeur - 1, 40, compacte, false},
		{120, compacteHauteur - 1, compacte, false},
		{largeurMin, hauteurMin, compacte, false},
		{compacteLargeur, 40, empilee, true},
		{110, 30, large, true},
	}
	for _, c := range cas {
		m, _ := nouveau(t)
		m.Update(tea.WindowSizeMsg{Width: c.w, Height: c.h})
		e := ecran(m)
		if m.d.agencement != c.attendu || m.d.fenetre != c.avecFenetre {
			t.Errorf("%dx%d : %v (fenêtre %v), attendu %v (fenêtre %v)", c.w, c.h, m.d.agencement, m.d.fenetre, c.attendu, c.avecFenetre)
		}
		if strings.Contains(e, "Claude Code · mon-projet") != c.avecFenetre {
			t.Errorf("%dx%d : mini-fenêtre présente : %v, attendu %v", c.w, c.h, !c.avecFenetre, c.avecFenetre)
		}
		if !strings.Contains(e, "ctx") {
			t.Errorf("%dx%d : la statusline du thème doit toujours être visible :\n%s", c.w, c.h, e)
		}
		if c.attendu == compacte {
			if m.d.logo || strings.Contains(e, logo[0]) {
				t.Errorf("%dx%d : pas de logo en compact", c.w, c.h)
			}
			if m.lignesListe < listeMin {
				t.Errorf("%dx%d : %d thèmes visibles, attendu au moins %d", c.w, c.h, m.lignesListe, listeMin)
			}
		}
	}
	// le logo ne prend sa place que s'il en reste : on retrouve les mêmes thèmes visibles sans lui
	m, _ := nouveau(t)
	for h := compacteHauteur; h < compacteHauteur+20; h++ {
		m.Update(tea.WindowSizeMsg{Width: compacteLargeur, Height: h})
		if m.d.logo && m.d.liste < listeMin {
			t.Errorf("%d lignes : logo dessiné aux dépens de la liste (%d thèmes)", h, m.d.liste)
		}
	}
}

// TestEntetePriorite : à 80 colonnes, le thème en vigueur et pourquoi passent avant le chemin du dépôt.
func TestEntetePriorite(t *testing.T) {
	m, _ := nouveauDepot(t, "git@gitlab.com:adrien-ln/un-depot-au-nom-assez-long.git", "main")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	appuyer(m, "↓")
	l := strings.Split(ecran(m), "\n")[lignesLogo]
	for _, x := range []string{"un-depot-au-nom-assez-long", "en vigueur : Tokyo Night (intégré)"} {
		if !strings.Contains(l, x) {
			t.Errorf("en-tête à 80 colonnes : %q absent de %q", x, l)
		}
	}
	if strings.Contains(l, "gitlab.com") && !strings.Contains(l, "survolé") {
		t.Errorf("le chemin ne doit venir qu'après le thème survolé : %q", l)
	}
	m.Update(tea.WindowSizeMsg{Width: 220, Height: 40})
	if l := strings.Split(ecran(m), "\n")[lignesLogo]; !strings.Contains(l, "survolé : ") || !strings.Contains(l, "gitlab.com/adrien-ln") {
		t.Errorf("en-tête large : tout doit tenir : %q", l)
	}
}

func TestEnteteSansAuteurDepotEnAvant(t *testing.T) {
	m, _ := nouveau(t)
	v := m.View()
	if strings.Contains(v, "Adrien") || strings.Contains(v, "par ") {
		t.Error("le nom d'auteur ne doit plus apparaître")
	}
	depot := m.projet[strings.LastIndex(m.projet, "/")+1:]
	p := m.palette(m.noms[m.i])
	if !strings.Contains(v, ansi.Bg(p.Modele)+ansi.Fg(visageSombre)+gras+" "+depot+" ") {
		t.Errorf("le nom du dépôt (%s) doit être mis en avant, sur la couleur du thème", depot)
	}
}

func TestEnvelopper(t *testing.T) {
	l := envelopper("  "+cAjout+"+ une très longue ligne de diff qui doit passer à la ligne en gardant son fond"+z, 30)
	if len(l) < 3 {
		t.Fatalf("%d lignes", len(l))
	}
	for k, x := range l {
		if largeur(x) > 30 {
			t.Errorf("ligne %d trop longue : %d", k, largeur(x))
		}
		if k > 0 && (!strings.HasPrefix(x, "     ") || !strings.Contains(x, cAjout)) {
			t.Errorf("suite %d : décalée sous ⎿ et en gardant la couleur attendus, obtenu %q", k, x)
		}
	}
	if r := envelopper(cTexte+"⏺"+z+" Corrigé : la boucle sautait le premier thème, d'où un thème en moins.", 30); !strings.HasPrefix(r[1], "  ") || strings.HasPrefix(r[1], "     ") {
		t.Errorf("suite d'une réponse : décalée de 2 colonnes attendu, obtenu %q", r[1])
	}
}
