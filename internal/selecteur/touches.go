package selecteur

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Les touches du sélecteur : une seule table, qui sert à réagir aux touches, à écrire le pied de page, le
// panneau de validation et l'aide complète (?).

type mode int

const (
	enListe     mode = iota // on parcourt la liste
	enRecherche             // on tape le texte cherché
	enQuestion              // Entrée : où garder le thème ?
	enAide                  // l'aide complète

	enStyle         // onglet Statusline : on choisit un item sur l'aperçu
	enStyleMenu     // le menu de l'item (Entrée) ou les options générales (g)
	enStyleAjout    // les items à ajouter à la ligne (+)
	enStyleSaisie   // on tape une valeur (Entrée sur un libellé, un glyphe)
	enStyleQuestion // Entrée : où garder ce style ?
)

// nomsModes : les titres des sections de l'aide (en français, traduits à l'affichage).
var nomsModes = map[mode]string{enListe: i18n.M("Thèmes : liste"), enRecherche: i18n.M("Thèmes : recherche (/)"), enQuestion: i18n.M("Thèmes : validation (Entrée)"), enAide: i18n.M("Aide (?)"),
	enStyle: i18n.M("Statusline : items"), enStyleMenu: i18n.M("Statusline : menu (Entrée, g)"), enStyleAjout: i18n.M("Statusline : ajouter (+)"), enStyleSaisie: i18n.M("Statusline : saisie (Entrée)"), enStyleQuestion: i18n.M("Statusline : validation (s)")}

// modesAide : les sections de l'aide complète de chaque onglet, dans l'ordre.
var modesAide = [2][]mode{{enListe, enRecherche, enQuestion}, {enStyle, enStyleMenu, enStyleAjout, enStyleSaisie, enStyleQuestion}}

// raccourci : une touche (ou plusieurs équivalentes) dans un mode.
type raccourci struct {
	mode    mode
	touches []string               // noms Bubble Tea (« up », « k ») ; aucun : ligne d'aide seulement
	nom     string                 // tel qu'affiché (« ↑↓ »), en français, traduit par touche
	effet   string                 // ce qu'elle fait, pour l'aide complète (en français, traduit à l'affichage)
	libelle func(m *Modele) string // ce qu'elle fait maintenant (pied, panneau), déjà traduit ; nil : effet
	dispo   func(m *Modele) bool   // nil : toujours disponible ; sinon ignorée et cachée quand faux
	rang    int                    // place dans le pied (1 : la dernière retirée faute de place) ; 0 : absente
	faire   func(m *Modele) tea.Cmd
}

func (r raccourci) maintenant(m *Modele) string {
	if r.libelle != nil {
		return r.libelle(m)
	}
	return i18n.T(r.effet)
}

// touche : le nom de la touche, tel qu'affiché dans la langue des textes (« Entrée », « Enter »).
func (r raccourci) touche() string { return i18n.T(r.nom) }

func (r raccourci) disponible(m *Modele) bool { return r.dispo == nil || r.dispo(m) }

var raccourcis []raccourci

func init() {
	quitter := (*Modele).quitterOuAvertir
	avecBranche := func(m *Modele) bool { return m.branche != "" }
	raccourcis = []raccourci{
		{mode: enListe, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("parcourir"), rang: 2},
		{mode: enListe, touches: []string{"pgup", "pgdown", "home", "end"}, nom: i18n.M("PgPréc PgSuiv Début Fin"), effet: i18n.M("sauter")},
		{mode: enListe, touches: []string{"left", "right"}, nom: "←→", effet: i18n.M("filtre (populaire, couleur)"), libelle: func(*Modele) string { return i18n.T("filtre") }, rang: 5},
		{mode: enListe, touches: []string{"/"}, nom: "/", effet: i18n.M("chercher (nom, titre ou couleur)"), libelle: func(*Modele) string { return i18n.T("chercher") }, rang: 4,
			faire: func(m *Modele) tea.Cmd { m.enRecherche = true; return nil }},
		{mode: enListe, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("garder le thème (ce projet, la branche, partout)"), libelle: func(*Modele) string { return i18n.T("garder") }, rang: 1,
			dispo: func(m *Modele) bool { return len(m.vue) > 0 },
			faire: func(m *Modele) tea.Cmd {
				m.confirmer = true
				m.reagir("danse") // il danse tant que la question est posée
				return nil
			}},
		{mode: enListe, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("un cran en arrière : recherche, filtre, puis quitter"),
			libelle: (*Modele).retourListe, rang: 3, faire: (*Modele).remonter},
		{mode: enListe, touches: []string{"?"}, nom: "?", effet: i18n.M("cette aide"), libelle: func(*Modele) string { return i18n.T("aide") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = true; return nil }},
		{mode: enListe, touches: []string{"m"}, nom: "m", effet: i18n.M("mascotte : unie, aucune"),
			libelle: func(m *Modele) string { return i18n.T("mascotte (%s)", i18n.T(modesMascotte[m.mascotte])) }, rang: 6,
			faire: func(m *Modele) tea.Cmd { m.mascotte = (m.mascotte + 1) % len(modesMascotte); return nil }},
		{mode: enListe, touches: []string{"tab", "shift+tab"}, nom: "Tab", effet: i18n.M("onglet Statusline : les items de la statusline"), libelle: func(*Modele) string { return i18n.T("statusline") }, rang: 7,
			faire: (*Modele).basculerOnglet},
		{mode: enListe, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter sans rien changer (sauf en recherche)"), libelle: func(*Modele) string { return i18n.T("quitter") }, rang: 8, faire: quitter},

		{mode: enRecherche, nom: i18n.M("lettres"), effet: i18n.M("nom, titre ou couleur (bleu, rose…)")},
		{mode: enRecherche, touches: []string{"up", "down"}, nom: "↑↓", effet: i18n.M("parcourir les résultats")},
		{mode: enRecherche, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("valider la recherche"), libelle: func(*Modele) string { return i18n.T("valider") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.enRecherche = false; return nil }},
		{mode: enRecherche, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("effacer la recherche"), libelle: func(*Modele) string { return i18n.T("effacer") }, rang: 2,
			faire: func(m *Modele) tea.Cmd {
				m.enRecherche, m.recherche = false, ""
				m.calculerVue()
				return nil
			}},
		{mode: enRecherche, touches: []string{"backspace"}, nom: "⌫", effet: i18n.M("corriger"), rang: 3, faire: func(m *Modele) tea.Cmd {
			if r := []rune(m.recherche); len(r) > 0 {
				m.recherche = string(r[:len(r)-1])
				m.calculerVue()
			}
			return nil
		}},

		{mode: enQuestion, touches: []string{"p"}, nom: "p", effet: i18n.M("pour ce projet (reconnu par son remote)"), rang: 1,
			libelle: func(m *Modele) string { return i18n.T("ce projet (%s)", m.depot()) },
			faire:   func(m *Modele) tea.Cmd { return m.garder(theme.SourceProjet) }},
		{mode: enQuestion, touches: []string{"b"}, nom: "b", effet: i18n.M("pour la branche courante, dans tous les projets"), rang: 1, dispo: avecBranche,
			libelle: func(m *Modele) string { return i18n.T("la branche %s", m.branche) },
			faire:   func(m *Modele) tea.Cmd { return m.garder(theme.SourceBranche) }},
		{mode: enQuestion, touches: []string{"d"}, nom: "d", effet: i18n.M("partout : le thème par défaut"), libelle: func(*Modele) string { return i18n.T("partout (défaut)") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { return m.garder(theme.SourceDefaut) }},
		{mode: enQuestion, touches: []string{"u"}, nom: "u", effet: i18n.M("retirer le réglage d'ici (règle de branche, sinon thème du projet)"), rang: 1,
			dispo: func(m *Modele) bool { _, ok := m.aRetirer(); return ok }, libelle: (*Modele).libelleRetirer, faire: (*Modele).retirer},
		{mode: enQuestion, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("revenir à la liste"), libelle: func(*Modele) string { return i18n.T("retour") }, rang: 1,
			faire: func(m *Modele) tea.Cmd {
				m.confirmer = false
				m.reagir("clin")
				return nil
			}},
		{mode: enQuestion, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter sans rien changer"), faire: quitter},

		{mode: enStyle, touches: []string{"left", "h", "right", "l"}, nom: "←→", effet: i18n.M("choisir l'item (surligné sur l'aperçu)"), libelle: func(*Modele) string { return i18n.T("item") }, rang: 2,
			faire: func(m *Modele) tea.Cmd { m.os.choisir(sensH(m.derniere)); return nil }},
		{mode: enStyle, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("changer de ligne"), libelle: func(*Modele) string { return i18n.T("ligne") }, rang: 5,
			faire: func(m *Modele) tea.Cmd { m.os.changerLigne(sens(m.derniere)); return nil }},
		{mode: enStyle, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("le menu de l'item : ses options, coller, retirer"), libelle: func(*Modele) string { return i18n.T("menu") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.ouvrir(menuItem); return nil }},
		{mode: enStyle, touches: []string{"shift+left", "H", "shift+right", "L"}, nom: i18n.M("Maj+←→ H L"), effet: i18n.M("déplacer l'item dans sa ligne"),
			libelle: func(*Modele) string { return i18n.T("déplacer") }, rang: 6, faire: func(m *Modele) tea.Cmd { m.os.deplacer(sensH(m.derniere)); return nil }},
		{mode: enStyle, touches: []string{"shift+up", "K", "shift+down", "J"}, nom: i18n.M("Maj+↑↓ K J"), effet: i18n.M("passer l'item de la ligne 1 à la ligne 2, et inversement"),
			faire: func(m *Modele) tea.Cmd { m.os.changerDeLigne(sens(m.derniere)); return nil }},
		{mode: enStyle, touches: []string{"+", "a"}, nom: "+", effet: i18n.M("ajouter un item à la ligne"), libelle: func(*Modele) string { return i18n.T("ajouter") }, rang: 4,
			faire: func(m *Modele) tea.Cmd { m.os.ouvrir(menuAjout); return nil }},
		{mode: enStyle, touches: []string{"x", "delete"}, nom: "x", effet: i18n.M("retirer l'item de la ligne"), libelle: func(*Modele) string { return i18n.T("retirer") }, rang: 6,
			faire: func(m *Modele) tea.Cmd { m.os.retirer(); return nil }},
		{mode: enStyle, touches: []string{"g"}, nom: "g", effet: i18n.M("options générales : icônes, séparateurs, SSH"), libelle: func(*Modele) string { return i18n.T("général") }, rang: 5,
			faire: func(m *Modele) tea.Cmd { m.os.ouvrir(menuGeneral); return nil }},
		{mode: enStyle, touches: []string{"v"}, nom: "v", effet: i18n.M("aperçu : complète, tendue (alertes, conflit, plafond), API (coût), SSH"),
			libelle: func(m *Modele) string { return i18n.T("aperçu (%s)", i18n.T(nomsApercus[m.os.apercuVu()])) }, rang: 7, faire: (*Modele).apercuSuivant},
		{mode: enStyle, touches: []string{"r"}, nom: "r", effet: i18n.M("l'item par défaut (ses options, affiché)"), rang: 9,
			faire: func(m *Modele) tea.Cmd { m.os.remettre(); return nil }},
		{mode: enStyle, touches: []string{"R"}, nom: "R", effet: i18n.M("tout le style par défaut"), rang: 9,
			faire: func(m *Modele) tea.Cmd { m.os.st = style.Defaut(); m.os.changer(); m.os.borner(); return nil }},
		{mode: enStyle, touches: []string{"tab", "shift+tab"}, nom: "Tab", effet: i18n.M("onglet Thèmes (l'aperçu prend le thème survolé)"), libelle: func(*Modele) string { return i18n.T("thèmes") }, rang: 8,
			faire: (*Modele).basculerOnglet},
		{mode: enStyle, touches: []string{"s"}, nom: "s", effet: i18n.M("garder ce style"), libelle: func(*Modele) string { return i18n.T("garder") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.confirmer = true; m.reagir("danse"); return nil }},
		{mode: enStyle, touches: []string{"?"}, nom: "?", effet: i18n.M("cette aide"), libelle: func(*Modele) string { return i18n.T("aide") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = true; return nil }},
		{mode: enStyle, touches: []string{"esc", "q"}, nom: i18n.M("Échap q"), effet: i18n.M("quitter (modifications non gardées : le dit une fois)"), libelle: func(*Modele) string { return i18n.T("quitter") }, rang: 4, faire: quitter},

		{mode: enStyleMenu, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("choisir la rangée"), rang: 3,
			faire: func(m *Modele) tea.Cmd {
				m.os.opt = min(max(m.os.opt+sens(m.derniere), 0), len(m.os.rangeesMenu())-1)
				return nil
			}},
		{mode: enStyleMenu, touches: []string{"left", "right", "h", "l"}, nom: "←→", effet: i18n.M("changer la valeur"), libelle: func(*Modele) string { return i18n.T("valeur") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.regler(sensH(m.derniere)); return nil }},
		{mode: enStyleMenu, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("taper la valeur (libellé, glyphe, barre), sinon la suivante ; coller, retirer"),
			libelle: func(m *Modele) string {
				switch r := m.os.rangeeMenu(); {
				case r.action == actionRetirer:
					return i18n.T("retirer")
				case r.action == actionColler:
					return i18n.T("coller")
				case saisissable(r.cle):
					return i18n.T("taper")
				}
				return i18n.T("suivante")
			}, rang: 2,
			faire: func(m *Modele) tea.Cmd { m.os.entrer(); return nil }},
		{mode: enStyleMenu, touches: []string{"r"}, nom: "r", effet: i18n.M("l'option par défaut"), rang: 5, faire: func(m *Modele) tea.Cmd { m.os.remettre(); return nil }},
		{mode: enStyleMenu, touches: []string{"v"}, nom: "v", effet: i18n.M("aperçu suivant"), rang: 6, faire: (*Modele).apercuSuivant},
		{mode: enStyleMenu, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("fermer le menu"), libelle: func(*Modele) string { return i18n.T("fermer") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.menu = sansMenu; return nil }},
		{mode: enStyleMenu, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter"), rang: 7, faire: quitter},

		{mode: enStyleAjout, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("choisir l'item"), rang: 2,
			faire: func(m *Modele) tea.Cmd {
				m.os.opt = min(max(m.os.opt+sens(m.derniere), 0), len(m.os.rangeesMenu())-1)
				return nil
			}},
		{mode: enStyleAjout, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("l'ajouter après l'item choisi"), libelle: func(*Modele) string { return i18n.T("ajouter") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.entrer(); return nil }},
		{mode: enStyleAjout, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("fermer sans rien ajouter"), libelle: func(*Modele) string { return i18n.T("fermer") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.menu = sansMenu; return nil }},
		{mode: enStyleAjout, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter"), rang: 3, faire: quitter},

		{mode: enStyleSaisie, nom: i18n.M("lettres"), effet: i18n.M("la valeur ; un caractère large (CJK, emoji) est refusé")},
		{mode: enStyleSaisie, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("valider"), libelle: func(*Modele) string { return i18n.T("valider") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.valider(); return nil }},
		{mode: enStyleSaisie, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("annuler"), libelle: func(*Modele) string { return i18n.T("annuler") }, rang: 2,
			faire: func(m *Modele) tea.Cmd { m.os.saisie = false; return nil }},
		{mode: enStyleSaisie, touches: []string{"backspace"}, nom: "⌫", effet: i18n.M("corriger"), rang: 3, faire: func(m *Modele) tea.Cmd {
			if r := []rune(m.os.texte); len(r) > 0 {
				m.os.texte = string(r[:len(r)-1])
			}
			return nil
		}},

		{mode: enStyleQuestion, touches: []string{"d"}, nom: "d", effet: i18n.M("partout : le style de toutes les statuslines"), libelle: func(*Modele) string { return i18n.T("partout (défaut)") }, rang: 1,
			faire: (*Modele).garderStyle},
		{mode: enStyleQuestion, touches: []string{"p"}, nom: "p", effet: i18n.M("pour ce projet : bientôt (le style par projet n'existe pas encore)"),
			libelle: func(*Modele) string { return estompe + i18n.T("ce projet (bientôt)") + z }, rang: 1,
			faire: func(m *Modele) tea.Cmd {
				m.os.message = i18n.T("le style par projet arrive bientôt : d le garde pour tous les projets")
				return nil
			}},
		{mode: enStyleQuestion, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("revenir aux items"), libelle: func(*Modele) string { return i18n.T("retour") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.os.confirmer = false; m.reagir("clin"); return nil }},
		{mode: enStyleQuestion, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter sans garder"), faire: quitter},

		{mode: enAide, touches: []string{"esc", "?"}, nom: i18n.M("Échap ?"), effet: i18n.M("fermer l'aide"), libelle: func(*Modele) string { return i18n.T("fermer") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = false; return nil }},
		{mode: enAide, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter"), rang: 2, faire: quitter},
	}
}

func (m *Modele) mode() mode {
	switch {
	case m.aide:
		return enAide
	case m.onglet == 1 && m.os.saisie:
		return enStyleSaisie
	case m.onglet == 1 && m.os.confirmer:
		return enStyleQuestion
	case m.onglet == 1 && m.os.menu == menuAjout:
		return enStyleAjout
	case m.onglet == 1 && m.os.menu != sansMenu:
		return enStyleMenu
	case m.onglet == 1:
		return enStyle
	case m.enRecherche:
		return enRecherche
	case m.confirmer:
		return enQuestion
	}
	return enListe
}

// raccourciDe : le raccourci de la touche dans le mode en cours.
func (m *Modele) raccourciDe(touche string) (raccourci, bool) {
	md := m.mode()
	for _, r := range raccourcis {
		if r.mode != md || !r.disponible(m) {
			continue
		}
		for _, t := range r.touches {
			if t == touche {
				return r, true
			}
		}
	}
	return raccourci{}, false
}

// touche : la touche, puis la réaction de Pip à ce qu'elle a changé, puis la disposition.
func (m *Modele) touche(k tea.KeyMsg) tea.Cmd {
	theme, filtre, dormait, ver := m.i, m.f, m.dort(), m.os.ver
	averti := m.avertiQuitter
	m.derniereTouche = m.t
	m.os.message = ""
	cmd := m.toucheBrute(k)
	if averti { // l'avertissement ne vaut que pour la touche qui suit
		m.avertiQuitter = false
	}
	switch {
	case dormait:
		m.reagir("reveil")
	case m.os.ver != ver:
		m.reagir("content")
	case m.f != filtre: // un filtre change souvent aussi le thème affiché : la surprise l'emporte
		m.reagir("surpris")
	case m.i != theme && (m.reaction == "content" || m.reaction == "suit") && m.t-m.reactionT < dureesReaction["suit"]:
		m.reagir("suit") // on parcourt la liste : un bond suffit, ensuite il la suit des yeux
	case m.i != theme:
		m.reagir("content")
	}
	m.ajuster()
	return cmd
}

func (m *Modele) toucheBrute(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	m.derniere = s
	if s == "ctrl+c" {
		return tea.Quit
	}
	if m.deplacer(s) {
		return nil
	}
	if r, ok := m.raccourciDe(s); ok && r.faire != nil {
		return r.faire(m)
	}
	if m.enRecherche && (k.Type == tea.KeyRunes || k.Type == tea.KeySpace) {
		m.recherche += string(k.Runes)
		m.calculerVue()
	}
	if m.mode() == enStyleSaisie && (k.Type == tea.KeyRunes || k.Type == tea.KeySpace) {
		m.os.texte += string(k.Runes)
	}
	return nil
}

// deplacer : les touches qui déplacent dans la liste ou changent de filtre (vrai si elle en était une).
func (m *Modele) deplacer(s string) bool {
	md := m.mode()
	if md == enRecherche && s != "up" && s != "down" || md != enListe && md != enRecherche {
		return false
	}
	n := len(m.vue)
	switch s {
	case "up", "k":
		if m.p > 0 {
			m.p--
		}
	case "down", "j":
		if m.p < n-1 {
			m.p++
		}
	case "home":
		m.p = 0
	case "end":
		m.p = max(n-1, 0)
	case "pgup":
		m.p = max(m.p-m.lignesListe, 0)
	case "pgdown":
		m.p = max(min(m.p+m.lignesListe, n-1), 0)
	case "right", "left":
		d := 1
		if s == "left" {
			d = len(m.filtres) - 1
		}
		m.f = (m.f + d) % len(m.filtres)
		m.calculerVue()
		return true
	default:
		return false
	}
	if n > 0 {
		m.i = m.vue[m.p]
	}
	return true
}

// --- Échap : un cran à la fois --------------------------------------------------------------------

func (m *Modele) retourListe() string {
	switch {
	case m.recherche != "":
		return i18n.T("effacer /%s", m.recherche)
	case m.f != 0:
		return i18n.T("filtre « tous »")
	}
	return i18n.T("quitter")
}

func (m *Modele) remonter() tea.Cmd {
	switch {
	case m.recherche != "":
		m.recherche = ""
	case m.f != 0:
		m.f = 0
	default:
		return m.quitterOuAvertir()
	}
	m.calculerVue()
	return nil
}

// --- Validation -----------------------------------------------------------------------------------

func (m *Modele) depot() string { return m.projet[strings.LastIndex(m.projet, "/")+1:] }

func visibleBientot() string {
	return i18n.T("visible au prochain rafraîchissement de la statusline.")
}

// erreur : le message de sortie d'une écriture impossible.
func erreur(err error) string { return i18n.T("✗ Erreur : %s", err.Error()) }

// garder : le thème affiché, gardé à cette source ; le message de sortie dit s'il restera masqué.
func (m *Modele) garder(source string) tea.Cmd {
	nom, titre := m.noms[m.i], m.titres[m.i]
	var err error
	var cible string
	switch source {
	case theme.SourceProjet:
		err, cible = m.sources.Associer(m.projet, nom), i18n.T("pour le projet %s", m.projet)
	case theme.SourceBranche:
		err, cible = m.sources.RegleBranche(m.branche, nom), i18n.T("pour la branche %s (dans tous les projets)", m.branche)
	default:
		err, cible = m.sources.EcrireDefaut(nom), i18n.T("par défaut")
	}
	if err != nil {
		m.Resultat = erreur(err)
		return tea.Quit
	}
	m.Resultat = i18n.T("✓ %s %s : %s", titre, cible, visibleBientot())
	if e, ok := theme.MasquePar(m.sources.Resoudre(m.dossier, m.branche, m.getenv), source); ok {
		m.Resultat += i18n.T("\n⚠ Ici, %s reste prioritaire.", m.qui(e))
	}
	if m.os.modifie() { // l'autre onglet : ses réglages ne sont pas gardés, et il faut le dire
		m.Resultat += i18n.T("\n⚠ Les modifications du style de la statusline n'ont pas été gardées (s dans l'onglet Statusline).")
	}
	return tea.Quit
}

// aRetirer : ce que u retire ici : la règle de branche qui s'applique, sinon le thème du projet.
func (m *Modele) aRetirer() (theme.Etape, bool) {
	for _, e := range m.chaine {
		if (e.Source == theme.SourceBranche || e.Source == theme.SourceProjet) && e.Nom != "" {
			return e, true
		}
	}
	return theme.Etape{}, false
}

func (m *Modele) libelleRetirer() string {
	e, _ := m.aRetirer()
	if e.Source == theme.SourceBranche {
		return i18n.T("retirer la règle %s (%s)", e.Motif, m.titreDe(e.Nom))
	}
	return i18n.T("retirer le thème du projet (%s)", m.titreDe(e.Nom))
}

func (m *Modele) retirer() tea.Cmd {
	e, _ := m.aRetirer()
	var err error
	quoi := i18n.T("Règle de branche %s", e.Motif)
	if e.Source == theme.SourceBranche {
		err = m.sources.RegleBranche(e.Motif, "")
	} else {
		err, quoi = m.sources.Associer(m.projet, ""), i18n.T("Thème du projet %s", m.projet)
	}
	if err != nil {
		m.Resultat = erreur(err)
		return tea.Quit
	}
	a := theme.Active(m.sources.Resoudre(m.dossier, m.branche, m.getenv))
	m.Resultat = i18n.T("✓ %s retiré : %s (%s) en vigueur ici, %s", quoi, m.titreDe(a.Nom), pourquoi(a), visibleBientot())
	return tea.Quit
}

// avertissement : la ligne « ⚠ » du panneau de validation, si un choix restera masqué ici.
func (m *Modele) avertissement() string {
	var masques []string
	var par theme.Etape
	for _, o := range []struct{ touche, source string }{{"p", theme.SourceProjet}, {"b", theme.SourceBranche}, {"d", theme.SourceDefaut}} {
		if o.touche == "b" && m.branche == "" {
			continue
		}
		if e, ok := theme.MasquePar(m.chaine, o.source); ok {
			masques, par = append(masques, "["+o.touche+"]"), e
		}
	}
	if len(masques) == 0 {
		return ""
	}
	s := i18n.T("⚠ Ici, %s l'emporte sur %s", m.qui(par), strings.Join(masques, " "))
	if e, ok := m.aRetirer(); ok && e.Source == par.Source {
		s += i18n.T(" ; [u] le retire")
	}
	return s
}

// --- Pied de page et aide -------------------------------------------------------------------------

// raccourcisDuPied : les raccourcis disponibles du mode rappelés dans le pied, dans l'ordre de la table.
func (m *Modele) raccourcisDuPied(md mode) []raccourci {
	var l []raccourci
	for _, r := range raccourcis {
		if r.mode == md && r.rang > 0 && r.disponible(m) && r.maintenant(m) != "" {
			l = append(l, r)
		}
	}
	return l
}

// aideCourte : « touche effet · … » sur une ligne de largeur w ; faute de place, les raccourcis de plus
// grand rang sont retirés d'abord (jamais une touche à moitié coupée).
func (m *Modele) aideCourte(md mode, w int) string {
	l := m.raccourcisDuPied(md)
	for {
		var t []string
		for _, r := range l {
			t = append(t, r.touche()+" "+r.maintenant(m))
		}
		s := strings.Join(t, " · ")
		if largeur(s) <= w || len(l) <= 1 {
			return s
		}
		pire := 0
		for k, r := range l {
			if r.rang > l[pire].rang {
				pire = k
			}
		}
		l = append(l[:pire:pire], l[pire+1:]...)
	}
}

const colonneTouche = 8 // largeur de la colonne des touches, dans l'aide complète

// lignesAide : l'aide complète tirée de la table, une section par mode, en lignes de largeur w au plus
// (l'effet passe à la ligne sous lui-même, rien n'est coupé).
func lignesAide(w int, modes []mode) []string {
	var l []string
	for _, md := range modes {
		if len(l) > 0 {
			l = append(l, "")
		}
		l = append(l, gras+i18n.T(nomsModes[md])+z)
		for _, r := range raccourcis {
			if r.mode != md {
				continue
			}
			nom := " " + r.touche() + espaces(colonneTouche-largeur(r.touche())) + " "
			retrait := espaces(colonneTouche + 2)
			if largeur(nom) > colonneTouche+2 { // nom long : l'effet commence à la ligne suivante
				l = append(l, nom)
				nom = retrait
			}
			for k, x := range replier(strings.Fields(i18n.T(r.effet)), " ", w-colonneTouche-2) {
				if k > 0 {
					nom = retrait
				}
				l = append(l, nom+estompe+x+z)
			}
		}
	}
	return l
}
