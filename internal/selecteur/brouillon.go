package selecteur

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Le brouillon : ce qui est choisi dans les deux onglets (un thème et sa portée, un réglage à retirer, le
// style de la statusline), écrit ensemble par s, sans fermer le sélecteur. Tant qu'il reste quelque chose à
// enregistrer, une barre le dit au-dessus du pied ; q le demande avant de quitter.

const dureeFlash = 3000 // ms : le « ✓ Enregistré » reste affiché

// porteeInitiale : là où le thème en vigueur est réglé (projet, branche), sinon partout.
func (m *Modele) porteeInitiale() string {
	switch a := theme.Active(m.chaine); {
	case a.Source == theme.SourceProjet, a.Source == theme.SourceBranche && m.branche != "":
		return a.Source
	}
	return theme.SourceDefaut
}

// libellePortee : où le thème sera enregistré, en peu de mots.
func (m *Modele) libellePortee() string {
	switch m.portee {
	case theme.SourceProjet:
		return i18n.T("ce projet (%s)", m.depot())
	case theme.SourceBranche:
		return i18n.T("la branche %s", m.branche)
	}
	return i18n.T("partout (défaut)")
}

// changerPortee : p, b ou d ; le choix du thème reste.
func (m *Modele) changerPortee(source string) tea.Cmd {
	m.portee = source
	return nil
}

// choisir : Entrée sur la liste : le thème affiché devient le choix à enregistrer.
func (m *Modele) choisir() tea.Cmd {
	m.choix, m.retrait = m.noms[m.i], false
	m.reagir("content")
	return nil
}

// marquerRetrait : x : le réglage d'ici (règle de branche, sinon thème du projet) sera retiré.
func (m *Modele) marquerRetrait() tea.Cmd {
	m.retrait, m.choix = true, ""
	return nil
}

// themeEnAttente : un thème choisi qui n'est pas déjà celui de sa portée.
func (m *Modele) themeEnAttente() bool {
	if m.choix == "" {
		return false
	}
	for _, e := range m.chaine {
		if e.Source == m.portee && e.Nom != "" {
			return e.Nom != m.choix
		}
	}
	return true
}

// reglagesStyle : combien d'options du style diffèrent de celles du fichier.
func (m *Modele) reglagesStyle() int {
	n := 0
	for _, c := range style.Cles() {
		if m.os.st.Valeur(c.Nom) != m.os.garde.Valeur(c.Nom) {
			n++
		}
	}
	return max(n, 1) // une disposition changée sans option : un réglage
}

// enAttente : ce qui reste à enregistrer, en peu de mots (vide : rien).
func (m *Modele) enAttente() []string {
	var l []string
	if m.themeEnAttente() {
		l = append(l, i18n.T("thème %s pour %s", m.titreDe(m.choix), m.libellePortee()))
	}
	if m.retrait {
		if _, ok := m.aRetirer(); ok {
			l = append(l, m.libelleRetirer())
		}
	}
	if m.os.modifie() {
		n := m.reglagesStyle()
		l = append(l, i18n.N(n, "style de la statusline (%d réglage)", "style de la statusline (%d réglages)", n))
	}
	return l
}

func (m *Modele) aEnregistrer() bool { return len(m.enAttente()) > 0 }

// masque : l'avertissement quand le thème choisi restera caché ici par une source plus forte.
func (m *Modele) masque() string {
	if !m.themeEnAttente() {
		return ""
	}
	if e, ok := theme.MasquePar(m.chaine, m.portee); ok {
		return i18n.T("⚠ Ici, %s reste prioritaire.", m.qui(e))
	}
	return ""
}

// enregistrer : s : tout le brouillon écrit d'un coup ; le sélecteur reste ouvert et le dit quelques secondes.
func (m *Modele) enregistrer() tea.Cmd {
	faits, avert := m.ecrire()
	switch {
	case m.Resultat != "": // une écriture impossible : le sélecteur se ferme sur l'erreur
		return tea.Quit
	case len(faits) == 0:
		m.flash = i18n.T("Rien à enregistrer.")
	default:
		m.flash = i18n.T("✓ Enregistré : %s", strings.Join(faits, " · "))
		m.gardes = append(m.gardes, faits...)
		m.avertis = append(m.avertis, avert...)
		m.reagir("danse")
	}
	m.flashT = m.t
	return nil
}

// ecrire : écrit le brouillon ; ce qui a été écrit, et les avertissements à rappeler en sortant.
func (m *Modele) ecrire() (faits, avert []string) {
	if a := m.masque(); a != "" {
		avert = append(avert, a)
	}
	if m.themeEnAttente() {
		nom, titre := m.choix, m.titreDe(m.choix)
		var err error
		switch m.portee {
		case theme.SourceProjet:
			err = m.sources.Associer(m.projet, nom)
		case theme.SourceBranche:
			err = m.sources.RegleBranche(m.branche, nom)
		default:
			err = m.sources.EcrireDefaut(nom)
		}
		if err != nil {
			m.Resultat = erreur(err)
			return nil, nil
		}
		faits = append(faits, i18n.T("thème %s pour %s", titre, m.libellePortee()))
	}
	if e, ok := m.aRetirer(); m.retrait && ok {
		var err error
		quoi := i18n.T("thème du projet retiré (%s)", m.titreDe(e.Nom))
		if e.Source == theme.SourceBranche {
			err, quoi = m.sources.RegleBranche(e.Motif, ""), i18n.T("règle %s retirée (%s)", e.Motif, m.titreDe(e.Nom))
		} else {
			err = m.sources.Associer(m.projet, "")
		}
		if err != nil {
			m.Resultat = erreur(err)
			return nil, nil
		}
		faits = append(faits, quoi)
	}
	if m.os.modifie() {
		n := m.reglagesStyle()
		if err := style.Ecrire(m.sources.Config, m.os.st); err != nil {
			m.Resultat = erreur(err)
			return nil, nil
		}
		m.os.garde = m.os.st.Copie()
		faits = append(faits, i18n.N(n, "style de la statusline (%d réglage)", "style de la statusline (%d réglages)", n))
		if m.os.ignore {
			avert = append(avert, i18n.T("⚠ AGENTLINE_STYLE=default : il ne se verra pas tant qu'elle est définie."))
		}
	}
	m.choix, m.retrait = "", false
	m.chaine = m.sources.Resoudre(m.dossier, m.branche, m.getenv)
	m.avant = theme.Active(m.chaine).Nom
	return faits, avert
}

// annuler : u : le brouillon oublié, tout revient à ce qui est enregistré.
func (m *Modele) annuler() tea.Cmd {
	m.choix, m.retrait = "", false
	if m.os.modifie() {
		m.os.st = m.os.garde.Copie()
		m.os.changer()
		m.os.borner()
	}
	m.reagir("clin")
	return nil
}

// quitter : q ; avec un brouillon, la question est posée d'abord (s, n, Échap).
func (m *Modele) quitter() tea.Cmd {
	if m.aEnregistrer() {
		m.questionQuitter = true
		m.reagir("surpris")
		return nil
	}
	return m.sortir()
}

// enregistrerEtQuitter : s dans la question de sortie.
func (m *Modele) enregistrerEtQuitter() tea.Cmd {
	if cmd := m.enregistrer(); cmd != nil {
		return cmd
	}
	return m.sortir()
}

// sortir : le message de sortie est fixé, le sélecteur se ferme.
func (m *Modele) sortir() tea.Cmd {
	m.Resultat = m.resume()
	return tea.Quit
}

// resume : ce qui a été enregistré pendant la session et ce qui ne l'a pas été.
func (m *Modele) resume() string {
	var l []string
	if len(m.gardes) > 0 {
		l = append(l, i18n.T("✓ Enregistré : %s ; %s", strings.Join(m.gardes, " · "), visibleBientot()))
		l = append(l, m.avertis...)
	} else {
		l = append(l, i18n.T("Rien n'a changé : %s reste en vigueur (%s).", m.titreDe(m.avant), pourquoi(theme.Active(m.chaine))))
	}
	if a := m.enAttente(); len(a) > 0 {
		l = append(l, i18n.T("Non enregistré : %s.", strings.Join(a, " · ")))
	}
	return strings.Join(l, "\n")
}

// lignesBrouillon : au-dessus du pied, le « ✓ Enregistré » de s pendant quelques secondes, sinon ce qui
// reste à enregistrer (et s'il restera caché ici).
func (m *Modele) lignesBrouillon(w int) []string {
	var l []string
	ajouter := func(t string) {
		for _, x := range replier(strings.Fields(t), " ", w) {
			l = append(l, gras+x+z)
		}
	}
	if m.flash != "" && m.t-m.flashT < dureeFlash {
		ajouter(m.flash)
		return l
	}
	if a := m.enAttente(); len(a) > 0 {
		ajouter("● " + i18n.T("Non enregistré : %s", strings.Join(a, " · ")) + "  ·  " + i18n.T("s enregistrer · u annuler"))
		if x := m.masque(); x != "" {
			ajouter(x)
		}
	}
	return l
}
