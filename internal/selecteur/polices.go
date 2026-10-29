package selecteur

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/style"
	"github.com/adrien-ln/agentline/internal/theme"
)

// L'écran de « agentline font preview » : la police du terminal change pour de vrai à chaque déplacement
// (après un court délai, pour ne pas réécrire ses réglages à chaque touche) ; on juge sur les glyphes
// d'agentline, Pip et la statusline d'exemple. Entrée garde la police, Échap ou q remet celle d'avant.

const delaiEssai = 400 * time.Millisecond

// ChoixPolice : une police de la liste.
type ChoixPolice struct {
	Nom, Titre, Famille string
	Installee           bool
}

// Polices : l'état de l'écran.
type Polices struct {
	env              render.Env
	palette          theme.Palette
	liste            []ChoixPolice
	i                int
	terminal, avant  string                     // le terminal en cours, sa police d'origine
	essayer          func(famille string) error // pose la police dans les réglages du terminal
	demarre          bool                       // l'avertissement est passé (Entrée)
	jeton            int                        // le déplacement en attente d'essai
	essai            string                     // la famille posée en dernier
	erreur           string
	largeur, hauteur int
	Garder           bool        // Entrée : la police essayée est gardée
	Choix            ChoixPolice // celle de la ligne choisie en sortant
}

type essaiPolice int

// NouvellesPolices : l'écran, pour la liste donnée ; essayer pose une police dans le terminal.
func NouvellesPolices(env render.Env, liste []ChoixPolice, terminal, avant string, essayer func(string) error) *Polices {
	p, ok := env.Themes.Charger(env.Themes.Choisir("", env.Getenv).Nom)
	if !ok {
		p = theme.Secours()
	}
	return &Polices{env: env, palette: p.PourFond(env.Couleurs.Fond == ansi.Clair), liste: liste, terminal: terminal, avant: avant,
		essayer: essayer, largeur: 100, hauteur: 30}
}

func (m *Polices) Init() tea.Cmd { return nil }

func (m *Polices) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.largeur, m.hauteur = msg.Width, msg.Height
	case essaiPolice:
		if int(msg) == m.jeton {
			m.poser()
		}
	case tea.KeyMsg:
		return m, m.touche(msg.String())
	}
	return m, nil
}

// touche : avant le départ, Entrée commence et Échap ou q renonce ; ensuite ↑↓ change de police, Entrée la
// garde, Échap, q ou Ctrl+C remettent celle d'avant.
func (m *Polices) touche(k string) tea.Cmd {
	switch k {
	case "ctrl+c", "esc", "q":
		m.Garder = false
		return tea.Quit
	case "enter":
		if !m.demarre {
			m.demarre = true
			return m.programmer()
		}
		if !m.liste[m.i].Installee {
			m.erreur = i18n.T("cette police n'est pas installée : agentline font set %s", m.liste[m.i].Nom)
			return nil
		}
		m.poser() // le délai n'est peut-être pas écoulé : la police gardée est celle de la ligne
		m.Garder, m.Choix = m.erreur == "", m.liste[m.i]
		return tea.Quit
	case "up", "k", "down", "j":
		if !m.demarre {
			return nil
		}
		d := 1
		if k == "up" || k == "k" {
			d = -1
		}
		m.i = min(max(m.i+d, 0), len(m.liste)-1)
		m.erreur = ""
		return m.programmer()
	}
	return nil
}

// programmer : l'essai de la ligne choisie, dans delaiEssai, s'il n'y a pas eu d'autre déplacement d'ici là.
func (m *Polices) programmer() tea.Cmd {
	m.jeton++
	j := m.jeton
	return tea.Tick(delaiEssai, func(time.Time) tea.Msg { return essaiPolice(j) })
}

// poser : la police de la ligne choisie dans le terminal (si elle est installée et pas déjà posée).
func (m *Polices) poser() {
	c := m.liste[m.i]
	if !c.Installee || c.Famille == m.essai {
		return
	}
	if err := m.essayer(c.Famille); err != nil {
		m.erreur = err.Error()
		return
	}
	m.essai = c.Famille
}

// --- Dessin ----------------------------------------------------------------------------------------

func (m *Polices) touchePied(nom, effet string) string {
	return ansi.Fg(couleurTouche(m.env.Couleurs.Fond == ansi.Clair)) + gras + nom + z + " " + estompe + effet + z
}

func (m *Polices) View() string {
	w := max(m.largeur, 40)
	var l []string
	l = append(l, "", "  "+gras+i18n.T("Aperçu des Nerd Fonts dans %s", m.terminal)+z, "")
	if !m.demarre {
		for _, x := range replier(strings.Fields(i18n.T("L'aperçu change pour de vrai la police de %s le temps de choisir (une copie .bak de ses réglages est faite) ; Échap remet la tienne (%s).", m.terminal, montre(m.avant))), " ", w-4) {
			l = append(l, "  "+x)
		}
		l = append(l, "", "  "+m.touchePied(i18n.T("Entrée"), i18n.T("commencer"))+"   "+m.touchePied(i18n.T("Échap"), i18n.T("renoncer")))
		return m.env.Couleurs.Adapter(strings.Join(l, "\n"))
	}
	for k, c := range m.liste {
		marque, etat := "  ", ""
		switch {
		case !c.Installee:
			etat = estompe + i18n.T("non installée") + z
		case c.Famille == m.essai:
			etat = i18n.T("essayée")
		case c.Famille == m.avant:
			etat = estompe + i18n.T("actuelle") + z
		}
		ligne := c.Titre
		if !c.Installee {
			ligne = estompe + ligne + z
		}
		if k == m.i {
			marque, ligne = gras+"› "+z, gras+ligne+z
		}
		l = append(l, "  "+marque+remplir(ligne, 30)+" "+etat)
	}
	l = append(l, "", "  "+gras+i18n.T("Avec cette police :")+z)
	var icones []string
	for _, c := range style.Cles() {
		if n, ok := strings.CutPrefix(c.Nom, "icon."); ok {
			icones = append(icones, style.GlypheDuJeu("nerd", n))
		}
	}
	for _, g := range GlyphesTouches() {
		icones = append(icones, g[1])
	}
	l = append(l, "    "+strings.Join(icones, " ")+"    ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█ ━╸─ ╭─╮")
	pip := colorer(poser(composer("levés", "ouverts", "deux"), 0), m.palette.Modele, couleurBec(m.palette.D75, m.palette.D100, m.palette.Modele))
	for _, x := range pip[1:] {
		l = append(l, "    "+x)
	}
	env := m.env
	if env.Columns == 0 || env.Columns > w {
		env.Columns = w
	}
	for _, x := range strings.Split(strings.TrimRight(render.RenderSession(etat.Exemple(env.Now, filepath.ToSlash("/apercu/"+i18n.T("mon-projet")), 42), env), "\n"), "\n") {
		l = append(l, "    "+tronquer(x+z, w-5))
	}
	if m.erreur != "" {
		l = append(l, "", "  "+gras+"⚠ "+m.erreur+z)
	}
	l = append(l, "", "  "+m.touchePied("↑↓", i18n.T("police"))+"   "+m.touchePied(i18n.T("Entrée"), i18n.T("garder (dans tes terminaux)"))+"   "+m.touchePied(i18n.T("Échap"), i18n.T("remettre l'ancienne")))
	return m.env.Couleurs.Adapter(strings.Join(l, "\n"))
}
