package selecteur

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
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
	enQuitter               // q avec quelque chose à enregistrer : enregistrer, quitter sans, ou revenir ?
	enAide                  // l'aide complète

	enStyle       // onglet Statusline : on choisit un item sur l'aperçu
	enStyleMenu   // le menu de l'item (Entrée) ou les options générales (g)
	enStyleAjout  // les items à ajouter à la ligne (+)
	enStyleSaisie // on tape une valeur (Entrée sur un libellé, un glyphe)

	enEditeur       // onglet Créer : on règle les couleurs du thème
	enEditeurSaisie // on tape un code de couleur (Entrée) ou un nom (n)
)

// nomsModes : les titres des sections de l'aide (en français, traduits à l'affichage).
var nomsModes = map[mode]string{enListe: i18n.M("Thèmes : liste"), enRecherche: i18n.M("Thèmes : recherche (/)"), enQuitter: i18n.M("Quitter avec des changements (q)"), enAide: i18n.M("Aide (?)"),
	enStyle: i18n.M("Statusline : items"), enStyleMenu: i18n.M("Statusline : menu (Entrée, g)"), enStyleAjout: i18n.M("Statusline : ajouter (+)"), enStyleSaisie: i18n.M("Statusline : saisie (Entrée)"),
	enEditeur: i18n.M("Créer : couleurs"), enEditeurSaisie: i18n.M("Créer : saisie (Entrée, n)")}

// modesAide : les sections de l'aide complète de chaque onglet, dans l'ordre.
var modesAide = [3][]mode{{enListe, enRecherche, enQuitter}, {enStyle, enStyleMenu, enStyleAjout, enStyleSaisie, enQuitter},
	{enEditeur, enEditeurSaisie, enQuitter}}

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

// glyphesTouches : les touches qui ont leur glyphe dans les Nerd Fonts (md-keyboard_*), sous leurs noms français
// et anglais ; prefixesTouches : les modificateurs (Maj+…).
var (
	glyphesTouches = map[string]string{
		"Entrée": "\U000f0311", "Enter": "\U000f0311", "Échap": "\U000f12b7", "Esc": "\U000f12b7",
		"Tab": "\U000f0312", "⌫": "\U000f030d",
	}
	prefixesTouches = []string{"Maj+", "Shift+"}
)

const glypheMaj = "\U000f0636" // md-apple_keyboard_shift

// capsule : une touche (son nom déjà traduit) telle que le sélecteur la dessine, partout où il la rappelle :
// chaque touche du nom sur un fond discret (SEP du thème) ; avec une Nerd Font (icons=nerd), Entrée, Tab,
// Échap, ⌫ et Maj prennent leur glyphe ; en gras, pour rester distincte de l'effet estompé même sans couleur.
func (m *Modele) capsule(nom string) string {
	if nom == i18n.T("lettres") { // pas une touche : ce qu'on tape
		return nom
	}
	nerd := m.os.st.Valeur("icons") == "nerd"
	p := m.palette(m.noms[m.i])
	var l []string
	for _, t := range strings.Fields(nom) {
		if g, ok := glyphesTouches[t]; ok && nerd {
			t = g
		}
		for _, pr := range prefixesTouches {
			if r, ok := strings.CutPrefix(t, pr); ok && nerd {
				t = glypheMaj + r
			}
		}
		l = append(l, ansi.Bg(p.Sep)+ansi.Fg(p.Texte)+gras+" "+t+" "+z)
	}
	return strings.Join(l, " ")
}

// GlyphesTouches : les touches dessinées avec un glyphe Nerd Font, nom traduit et glyphe, pour « agentline font
// test ».
func GlyphesTouches() [][2]string {
	return [][2]string{{i18n.T("Entrée"), glyphesTouches["Entrée"]}, {"Tab", glyphesTouches["Tab"]},
		{i18n.T("Échap"), glyphesTouches["Échap"]}, {"⌫", glyphesTouches["⌫"]}, {i18n.T("Maj"), glypheMaj}}
}

// rappel : une touche et ce qu'elle fait, l'effet estompé.
func (m *Modele) rappel(nom, effet string) string { return m.capsule(nom) + " " + estompe + effet + z }

func (r raccourci) disponible(m *Modele) bool { return r.dispo == nil || r.dispo(m) }

var raccourcis []raccourci

func init() {
	quitter := (*Modele).quitter
	avecBranche := func(m *Modele) bool { return m.branche != "" }
	enregistrer := raccourci{touches: []string{"s", "ctrl+s"}, nom: "s", effet: i18n.M("enregistrer le thème choisi et le style, sans fermer"),
		libelle: func(*Modele) string { return i18n.T("enregistrer") }, rang: 1, faire: (*Modele).enregistrer}
	annuler := raccourci{touches: []string{"u"}, nom: "u", effet: i18n.M("oublier ce qui n'est pas enregistré"), libelle: func(*Modele) string { return i18n.T("annuler") }, rang: 2,
		dispo: (*Modele).aEnregistrer, faire: (*Modele).annuler}
	dans := func(md mode, r raccourci) raccourci { r.mode = md; return r }
	raccourcis = []raccourci{
		{mode: enListe, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("parcourir"), rang: 2},
		{mode: enListe, touches: []string{"pgup", "pgdown", "home", "end"}, nom: i18n.M("PgPréc PgSuiv Début Fin"), effet: i18n.M("sauter")},
		{mode: enListe, touches: []string{"left", "right"}, nom: "←→", effet: i18n.M("filtre (populaire, couleur)"), libelle: func(*Modele) string { return i18n.T("filtre") }, rang: 5},
		{mode: enListe, touches: []string{"/"}, nom: "/", effet: i18n.M("chercher (nom, titre ou couleur)"), libelle: func(*Modele) string { return i18n.T("chercher") }, rang: 4,
			faire: func(m *Modele) tea.Cmd { m.enRecherche = true; return nil }},
		{mode: enListe, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("choisir ce thème (s l'enregistre)"), libelle: func(*Modele) string { return i18n.T("choisir") }, rang: 1,
			dispo: func(m *Modele) bool { return len(m.vue) > 0 }, faire: (*Modele).choisir},
		dans(enListe, enregistrer),
		dans(enListe, annuler),
		{mode: enListe, nom: "p b d", libelle: func(*Modele) string { return i18n.T("portée") }, rang: 3},
		{mode: enListe, touches: []string{"p"}, nom: "p", effet: i18n.M("portée du thème : ce projet (reconnu par son remote)"),
			faire: func(m *Modele) tea.Cmd { return m.changerPortee(theme.SourceProjet) }},
		{mode: enListe, touches: []string{"b"}, nom: "b", effet: i18n.M("portée du thème : la branche courante, dans tous les projets"), dispo: avecBranche,
			faire: func(m *Modele) tea.Cmd { return m.changerPortee(theme.SourceBranche) }},
		{mode: enListe, touches: []string{"d"}, nom: "d", effet: i18n.M("portée du thème : partout (le thème par défaut)"),
			faire: func(m *Modele) tea.Cmd { return m.changerPortee(theme.SourceDefaut) }},
		{mode: enListe, touches: []string{"x"}, nom: "x", effet: i18n.M("retirer le réglage d'ici (règle de branche, sinon thème du projet) ; s l'enregistre"), rang: 6,
			dispo: func(m *Modele) bool { _, ok := m.aRetirer(); return ok }, libelle: func(*Modele) string { return i18n.T("retirer le réglage d'ici") }, faire: (*Modele).marquerRetrait},
		{mode: enListe, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("un cran en arrière : recherche, puis filtre"),
			libelle: (*Modele).retourListe, rang: 3, faire: (*Modele).remonter},
		{mode: enListe, touches: []string{"?"}, nom: "?", effet: i18n.M("cette aide"), libelle: func(*Modele) string { return i18n.T("aide") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = true; return nil }},
		{mode: enListe, touches: []string{"m"}, nom: "m", effet: i18n.M("mascotte : unie, aucune"),
			libelle: func(m *Modele) string { return i18n.T("mascotte (%s)", i18n.T(modesMascotte[m.mascotte])) }, rang: 6,
			faire: func(m *Modele) tea.Cmd { m.mascotte = (m.mascotte + 1) % len(modesMascotte); return nil }},
		{mode: enListe, touches: []string{"tab", "shift+tab"}, nom: "Tab", effet: i18n.M("onglet suivant : Statusline, puis Créer (Maj+Tab : l'inverse)"), libelle: func(*Modele) string { return i18n.T("statusline") }, rang: 7,
			faire: (*Modele).basculerOnglet},
		{mode: enListe, touches: []string{"e"}, nom: "e", effet: i18n.M("créer un thème à partir de celui-ci (onglet Créer)"), libelle: func(*Modele) string { return i18n.T("créer") }, rang: 7,
			dispo: func(m *Modele) bool { return len(m.vue) > 0 }, faire: func(m *Modele) tea.Cmd { m.editer(m.noms[m.i]); return nil }},
		{mode: enListe, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter (demande d'abord s'il reste à enregistrer)"), libelle: func(*Modele) string { return i18n.T("quitter") }, rang: 8, faire: quitter},

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

		{mode: enQuitter, touches: []string{"s", "ctrl+s"}, nom: "s", effet: i18n.M("enregistrer, puis quitter"), libelle: func(*Modele) string { return i18n.T("enregistrer et quitter") }, rang: 1,
			faire: (*Modele).enregistrerEtQuitter},
		{mode: enQuitter, touches: []string{"n"}, nom: "n", effet: i18n.M("quitter sans enregistrer"), rang: 1, faire: (*Modele).sortir},
		{mode: enQuitter, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("revenir sans quitter"), libelle: func(*Modele) string { return i18n.T("revenir") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.questionQuitter = false; m.reagir("clin"); return nil }},

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
		{mode: enStyle, touches: []string{"tab", "shift+tab"}, nom: "Tab", effet: i18n.M("onglet suivant : Créer (Maj+Tab : Thèmes ; l'aperçu prend le thème survolé)"), libelle: func(*Modele) string { return i18n.T("créer") }, rang: 8,
			faire: (*Modele).basculerOnglet},
		dans(enStyle, enregistrer),
		dans(enStyle, annuler),
		{mode: enStyle, touches: []string{"?"}, nom: "?", effet: i18n.M("cette aide"), libelle: func(*Modele) string { return i18n.T("aide") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = true; return nil }},
		{mode: enStyle, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter (demande d'abord s'il reste à enregistrer)"), libelle: func(*Modele) string { return i18n.T("quitter") }, rang: 4, faire: quitter},

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

		{mode: enEditeur, touches: []string{"up", "k", "down", "j"}, nom: "↑↓", effet: i18n.M("choisir la couleur à régler"), libelle: func(*Modele) string { return i18n.T("couleur") }, rang: 2,
			faire: func(m *Modele) tea.Cmd {
				m.ed.role = min(max(m.ed.role+sens(m.derniere), 0), len(rolesTheme)-1)
				return nil
			}},
		{mode: enEditeur, touches: []string{"left", "h", "right", "l"}, nom: "←→", effet: i18n.M("régler la couleur sur le canal choisi"), libelle: func(*Modele) string { return i18n.T("régler") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { return m.reglerCouleur(sensH(m.derniere)) }},
		{mode: enEditeur, touches: []string{"shift+left", "H", "shift+right", "L"}, nom: i18n.M("Maj+←→ H L"), effet: i18n.M("régler 5 fois plus vite"),
			faire: func(m *Modele) tea.Cmd { return m.reglerCouleur(5 * sensH(m.derniere)) }},
		{mode: enEditeur, touches: []string{"c"}, nom: "c", effet: i18n.M("canal suivant : clarté, saturation, teinte"), rang: 3,
			libelle: func(m *Modele) string { return i18n.T(canaux[m.ed.canal]) },
			faire:   func(m *Modele) tea.Cmd { m.ed.canal = (m.ed.canal + 1) % len(canaux); return nil }},
		{mode: enEditeur, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("taper le code de la couleur (rrggbb)"), libelle: func(*Modele) string { return i18n.T("code") }, rang: 4,
			faire: func(m *Modele) tea.Cmd {
				m.ed.saisie, m.ed.texte = true, *rolesTheme[m.ed.role].champ(&m.ed.p.Couleurs)
				return nil
			}},
		{mode: enEditeur, touches: []string{"n"}, nom: "n", effet: i18n.M("renommer le thème (le nom du fichier en découle)"), libelle: func(*Modele) string { return i18n.T("nom") }, rang: 5,
			faire: func(m *Modele) tea.Cmd { m.ed.saisie, m.ed.saisieNom, m.ed.texte = true, true, m.ed.titre; return nil }},
		{mode: enEditeur, touches: []string{"f"}, nom: "f", effet: i18n.M("aperçu sur l'autre fond"), libelle: func(*Modele) string { return i18n.T("fond") }, rang: 6,
			faire: func(m *Modele) tea.Cmd { m.ed.autreFond = !m.ed.autreFond; return nil }},
		{mode: enEditeur, touches: []string{"v"}, nom: "v", effet: i18n.M("aperçu en 24 bits, 256 ou 16 couleurs"), libelle: func(*Modele) string { return i18n.T("couleurs") }, rang: 6,
			faire: func(m *Modele) tea.Cmd { m.ed.niveau = (m.ed.niveau + 1) % len(niveauxApercu); return nil }},
		{mode: enEditeur, touches: []string{"r"}, nom: "r", effet: i18n.M("la couleur enregistrée de ce rôle"), rang: 9,
			faire: func(m *Modele) tea.Cmd {
				*rolesTheme[m.ed.role].champ(&m.ed.p.Couleurs) = *rolesTheme[m.ed.role].champ(&m.ed.garde.Couleurs)
				m.majEditeur()
				return nil
			}},
		dans(enEditeur, enregistrer),
		dans(enEditeur, annuler),
		{mode: enEditeur, touches: []string{"?"}, nom: "?", effet: i18n.M("cette aide"), libelle: func(*Modele) string { return i18n.T("aide") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = true; return nil }},
		{mode: enEditeur, touches: []string{"tab", "shift+tab"}, nom: "Tab", effet: i18n.M("onglet suivant : Thèmes (Maj+Tab : Statusline)"), libelle: func(*Modele) string { return i18n.T("thèmes") }, rang: 7,
			faire: (*Modele).basculerOnglet},
		{mode: enEditeur, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter (demande d'abord s'il reste à enregistrer)"), libelle: func(*Modele) string { return i18n.T("quitter") }, rang: 8, faire: quitter},

		{mode: enEditeurSaisie, nom: i18n.M("lettres"), effet: i18n.M("le code (rrggbb, # permis ; vide pour MODEL_END), ou le nom du thème")},
		{mode: enEditeurSaisie, touches: []string{"enter"}, nom: i18n.M("Entrée"), effet: i18n.M("valider"), libelle: func(*Modele) string { return i18n.T("valider") }, rang: 1,
			faire: (*Modele).validerSaisie},
		{mode: enEditeurSaisie, touches: []string{"esc"}, nom: i18n.M("Échap"), effet: i18n.M("annuler"), libelle: func(*Modele) string { return i18n.T("annuler") }, rang: 2,
			faire: func(m *Modele) tea.Cmd { m.ed.saisie, m.ed.saisieNom = false, false; return nil }},
		{mode: enEditeurSaisie, touches: []string{"backspace"}, nom: "⌫", effet: i18n.M("corriger"), rang: 3, faire: func(m *Modele) tea.Cmd {
			if r := []rune(m.ed.texte); len(r) > 0 {
				m.ed.texte = string(r[:len(r)-1])
			}
			return nil
		}},

		{mode: enAide, touches: []string{"esc", "?"}, nom: i18n.M("Échap ?"), effet: i18n.M("fermer l'aide"), libelle: func(*Modele) string { return i18n.T("fermer") }, rang: 1,
			faire: func(m *Modele) tea.Cmd { m.aide = false; return nil }},
		{mode: enAide, touches: []string{"q"}, nom: "q", effet: i18n.M("quitter"), rang: 2, faire: quitter},
	}
}

func (m *Modele) mode() mode {
	switch {
	case m.aide:
		return enAide
	case m.questionQuitter:
		return enQuitter
	case m.onglet == 2 && m.ed.saisie && !m.aide:
		return enEditeurSaisie
	case m.onglet == 2 && !m.aide:
		return enEditeur
	case m.onglet == 1 && m.os.saisie:
		return enStyleSaisie
	case m.onglet == 1 && m.os.menu == menuAjout:
		return enStyleAjout
	case m.onglet == 1 && m.os.menu != sansMenu:
		return enStyleMenu
	case m.onglet == 1:
		return enStyle
	case m.enRecherche:
		return enRecherche
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
	m.derniereTouche = m.t
	m.os.message, m.ed.message = "", ""
	cmd := m.toucheBrute(k)
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
	if m.mode() == enEditeurSaisie && (k.Type == tea.KeyRunes || k.Type == tea.KeySpace) {
		m.ed.texte += string(k.Runes)
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
	return ""
}

func (m *Modele) remonter() tea.Cmd {
	switch {
	case m.recherche != "":
		m.recherche = ""
	case m.f != 0:
		m.f = 0
	default:
		return nil
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

// aRetirer : ce que x retire ici : la règle de branche qui s'applique, sinon le thème du projet.
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

// aideCourte : « touche effet  touche effet… » sur une ligne de largeur w ; faute de place, les raccourcis de plus
// grand rang sont retirés d'abord (jamais une touche à moitié coupée).
func (m *Modele) aideCourte(md mode, w int) string {
	l := m.raccourcisDuPied(md)
	for {
		var t []string
		for _, r := range l {
			t = append(t, m.rappel(r.touche(), r.maintenant(m)))
		}
		s := strings.Join(t, "  ")
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
func (m *Modele) lignesAide(w int, modes []mode) []string {
	var l []string
	for _, md := range modes {
		if len(l) > 0 {
			l = append(l, "")
		}
		l = append(l, gras+i18n.T(nomsModes[md])+z)
		for _, r := range raccourcis {
			if r.mode != md || r.effet == "" {
				continue
			}
			touche := m.capsule(r.touche())
			nom := " " + touche + espaces(colonneTouche-largeur(touche)) + " "
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
