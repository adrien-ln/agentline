// Package selecteur : le sélecteur de thèmes interactif (« agentline theme »), avec Bubble Tea.
// En-tête (titre centré, thème en vigueur et pourquoi), liste filtrable, Pip perché sur la mini-fenêtre
// Claude Code (session animée, vraie statusline calculée dans le même processus), à la taille du terminal.
// Rien n'est écrit pendant la navigation : Entrée choisit un thème, le second onglet (Tab, « Statusline ») règle
// les items de la statusline (internal/style), et s enregistre ce brouillon d'un coup, sans fermer.
//
//	brouillon.go     ce qui reste à enregistrer, s, u, q, le message de sortie
//
//	selecteur.go     l'état, son initialisation, la liste filtrée, le message de sortie
//	touches.go       la table des touches (actions, pied de page, aide « ? »)
//	onglet_style.go  l'onglet Statusline : items par ligne, options, aperçu en direct
//	disposition.go   les trois dispositions (large, empilée, compacte) et leurs seuils
//	dessin.go        View et ce qu'il dessine
package selecteur

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/theme"
)

const (
	z       = ansi.Z
	estompe = "\x1b[2m"
	gras    = "\x1b[1m"
	inverse = "\x1b[7m"
	// souligne, finSurligne : le surlignage de l'item choisi sur l'aperçu (avec un fond), et sa fin seule
	souligne    = "\x1b[4m"
	finSurligne = "\x1b[24;49m"
)

// Filtres, dans l'ordre de ← → ; ceux qui n'ont aucun thème sont sautés. Ce sont les tags des thèmes, en
// français ; ils sont traduits à l'affichage, et la recherche les trouve dans les deux langues.
var filtresOrdre = []string{i18n.M("tous"), i18n.M("populaire"), i18n.M("rouge"), i18n.M("orange"), i18n.M("jaune"), i18n.M("vert"),
	i18n.M("cyan"), i18n.M("bleu"), i18n.M("violet"), i18n.M("rose"), i18n.M("gris"), filtreClairs}

// filtreClairs : les thèmes faits pour un fond clair (« # fond: clair »), filtre ajouté à leurs tags.
var filtreClairs = i18n.M("clairs")

// Modele : l'état du sélecteur.
type Modele struct {
	sources          theme.Sources
	apercu           render.Env // modèle d'environnement pour la statusline fictive
	getenv           func(string) string
	noms, titres     []string
	tags             [][]string
	filtres          []string
	f                int
	recherche        string
	enRecherche      bool
	aide             bool  // l'aide complète (?) est affichée
	vue              []int // index (dans noms) des thèmes affichés
	p, haut, i       int   // position dans vue, début de la liste, thème affiché
	avant            string
	chaine           []theme.Etape // la chaîne de priorité à l'ouverture (theme.Resoudre)
	dossier, branche string
	projet           string
	largeur, hauteur int
	rendus, logos    map[string]string
	palettes         map[string]theme.Palette
	Resultat         string // message affiché en sortant

	scenes   []scene   // sessions Claude Code fictives, jouées en boucle
	t        int       // temps écoulé depuis l'ouverture (ms)
	debut    time.Time // ouverture (horloge réelle)
	mascotte int       // 0 couleur unie, 1 aucune (touche m)

	derniereTouche int         // instant de la dernière touche (ms) : Pip s'endort après 20 s sans rien
	reaction       string      // dernière réaction de Pip, et son instant
	reactionT      int         //
	lignesListe    int         // thèmes visibles dans la liste, selon la hauteur du terminal (Page préc./suiv.)
	d              disposition // calculée par ajuster, à chaque touche et redimensionnement

	onglet          int         // 0 Thèmes, 1 Statusline (Tab)
	os              ongletStyle // l'onglet Statusline (onglet_style.go)
	choix           string      // le thème choisi (Entrée), à enregistrer ; vide : aucun
	portee          string      // où l'enregistrer : projet, branche ou défaut (p, b, d)
	retrait         bool        // x : le réglage d'ici (règle de branche, sinon thème du projet) sera retiré
	questionQuitter bool        // q avec un brouillon : enregistrer, quitter sans, ou revenir ?
	flash           string      // « ✓ Enregistré … » après s, et son instant
	flashT          int         //
	gardes, avertis []string    // ce que s a écrit pendant la session, et ses avertissements, pour la sortie
	derniere        string      // la touche en cours (nom Bubble Tea), pour les raccourcis à plusieurs touches
}

type tic time.Time

// image : ~20 images par seconde, seulement tant que le sélecteur est ouvert ; 2 par seconde quand la
// mascotte dort (personne ne touche le clavier : seuls ses « z » bougent encore). Une touche est dessinée
// tout de suite ; l'animation reprend son rythme à l'image suivante.
func image(lent bool) tea.Cmd {
	d := 50 * time.Millisecond
	if lent {
		d = 500 * time.Millisecond
	}
	return tea.Tick(d, func(t time.Time) tea.Msg { return tic(t) })
}

// Nouveau : le sélecteur pour le dossier courant (chemin tel que Claude Code l'enverra).
func Nouveau(s theme.Sources, apercu render.Env, dossier string, getenv func(string) string) *Modele {
	m := &Modele{sources: s, apercu: apercu, getenv: getenv, dossier: dossier, largeur: 110, hauteur: 24, scenes: Scenes(),
		rendus: map[string]string{}, logos: map[string]string{}, palettes: map[string]theme.Palette{}}
	m.noms = s.Noms()
	for _, n := range m.noms {
		t, g := s.Entete(n)
		if p, ok := s.Charger(n); ok && p.Clair {
			g = append(g, filtreClairs)
		}
		m.titres = append(m.titres, t)
		m.tags = append(m.tags, g)
	}
	m.projet = s.ProjetID(dossier)
	m.branche = theme.BrancheDe(dossier)
	m.chaine = s.Resoudre(dossier, m.branche, getenv)
	m.avant = theme.Active(m.chaine).Nom
	m.portee = m.porteeInitiale()
	for k, n := range m.noms {
		if n == m.avant {
			m.i = k
		}
	}
	for _, f := range filtresOrdre {
		for k := range m.noms {
			if f == "tous" || m.aLeTag(k, f) {
				m.filtres = append(m.filtres, f)
				break
			}
		}
	}
	m.calculerVue()
	m.initStyle()
	m.ajuster()
	return m
}

func (m *Modele) aLeTag(k int, f string) bool {
	for _, t := range m.tags[k] {
		if t == f {
			return true
		}
	}
	return false
}

// correspond : le thème k passe-t-il le filtre f et la recherche r (nom, titre ou couleur) ?
func (m *Modele) correspond(k int, f, r string) bool {
	if f != "tous" && !m.aLeTag(k, f) {
		return false
	}
	if r == "" || strings.Contains(m.noms[k], r) || strings.Contains(strings.ToLower(m.titres[k]), r) {
		return true
	}
	for _, t := range m.tags[k] {
		if strings.Contains(t, r) || strings.Contains(strings.ToLower(i18n.T(t)), r) {
			return true
		}
	}
	return false
}

// calculerVue : la liste filtrée ; garde le thème affiché s'il y est, sinon prend le premier.
func (m *Modele) calculerVue() {
	f, r := m.filtres[m.f], strings.ToLower(m.recherche)
	m.vue = m.vue[:0]
	for k := range m.noms {
		if m.correspond(k, f, r) {
			m.vue = append(m.vue, k)
		}
	}
	m.p = 0
	for k, v := range m.vue {
		if v == m.i {
			m.p = k
			return
		}
	}
	if len(m.vue) > 0 {
		m.i = m.vue[0]
	}
}

// ailleurs : combien de thèmes la recherche trouverait sans le filtre.
func (m *Modele) ailleurs() int {
	n := 0
	for k := range m.noms {
		if m.correspond(k, "tous", strings.ToLower(m.recherche)) {
			n++
		}
	}
	return n
}

func (m *Modele) Init() tea.Cmd {
	m.debut = time.Now()
	return image(false)
}

// Update : une touche, un redimensionnement ou une image de l'animation -> le nouvel état.
func (m *Modele) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tic:
		m.t = int(time.Time(msg).Sub(m.debut).Milliseconds())
		return m, image(m.dort())
	case tea.WindowSizeMsg:
		m.largeur, m.hauteur = msg.Width, msg.Height
		m.ajuster()
	case tea.KeyMsg:
		return m, m.touche(msg)
	}
	return m, nil
}

// titreDe : le titre affiché d'un thème (« Tokyo Night »), à défaut son nom.
func (m *Modele) titreDe(nom string) string {
	for k, n := range m.noms {
		if n == nom {
			return m.titres[k]
		}
	}
	return nom
}

// pourquoi : d'où vient le thème d'une étape, en peu de mots (« branche main », « projet », « défaut »).
func pourquoi(e theme.Etape) string {
	switch e.Source {
	case theme.SourceForce:
		return "AGENTLINE_THEME"
	case theme.SourceBranche:
		return i18n.T("branche %s", e.Cle)
	}
	return theme.NomSource(e.Source)
}

// qui : l'étape qui l'emporte, dans une phrase (« la branche main (Nord) »).
func (m *Modele) qui(e theme.Etape) string {
	t := " (" + m.titreDe(e.Nom) + ")"
	switch e.Source {
	case theme.SourceForce:
		return "AGENTLINE_THEME=" + e.Nom + t
	case theme.SourceBranche:
		if e.Motif != e.Cle {
			return i18n.T("la règle de branche %s", e.Motif) + t
		}
		return i18n.T("la branche %s", e.Cle) + t
	case theme.SourceProjet:
		return i18n.T("le thème du projet") + t
	}
	return i18n.T("le thème %s", theme.NomSource(e.Source)) + t
}

// Message : à afficher une fois le sélecteur fermé.
func (m *Modele) Message() string {
	if m.Resultat != "" {
		return m.Resultat
	}
	return m.resume() // Ctrl+C
}
