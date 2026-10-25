package selecteur

import (
	"math"
	"reflect"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/theme"
)

// L'onglet Créer : l'éditeur de thèmes. Il part d'une copie du thème survolé (ou du thème de l'utilisateur
// lui-même), règle chaque rôle en clarté, chroma ou teinte CIELAB (ou par son code), et montre la statusline
// en direct, sur l'un ou l'autre fond, à chaque niveau de couleurs. Ses changements font partie du brouillon :
// s les écrit dans $AGENTLINE_CONFIG/themes/<nom>.theme (theme.Format).

// roleTheme : une couleur du thème, telle que l'éditeur la montre.
type roleTheme struct {
	cle   string                        // le nom documenté (TEXT, MODEL…)
	aide  string                        // ce qu'il colore, en français (traduit à l'affichage)
	champ func(*theme.Couleurs) *string // où la couleur est rangée
	seuil float64                       // contraste minimal face au fond (0 : décoratif, pas de contrôle)
}

var rolesTheme = []roleTheme{
	{"TEXT", i18n.M("texte"), func(c *theme.Couleurs) *string { return &c.Texte }, 4.5},
	{"MUTED", i18n.M("libellés, heures"), func(c *theme.Couleurs) *string { return &c.Discret }, 3},
	{"SEP", i18n.M("séparateurs"), func(c *theme.Couleurs) *string { return &c.Sep }, 0},
	{"DIM", i18n.M("cases éteintes des barres"), func(c *theme.Couleurs) *string { return &c.Eteint }, 0},
	{"MODEL", i18n.M("nom du modèle (début du dégradé)"), func(c *theme.Couleurs) *string { return &c.Modele }, 3},
	{"MODEL_END", i18n.M("fin du dégradé (vide : BRANCH)"), func(c *theme.Couleurs) *string { return &c.ModeleFin }, 3},
	{"FOLDER", i18n.M("dossier"), func(c *theme.Couleurs) *string { return &c.Dossier }, 3},
	{"BRANCH", i18n.M("branche, PR"), func(c *theme.Couleurs) *string { return &c.Lieu }, 3},
	{"D0", i18n.M("jauges : bas"), func(c *theme.Couleurs) *string { return &c.D0 }, 3},
	{"D50", i18n.M("jauges : moitié"), func(c *theme.Couleurs) *string { return &c.D50 }, 3},
	{"D75", i18n.M("jauges : haut"), func(c *theme.Couleurs) *string { return &c.D75 }, 3},
	{"D100", i18n.M("jauges : plein, alertes"), func(c *theme.Couleurs) *string { return &c.D100 }, 3},
	{"INTENSITY 1", i18n.M("effort : bas"), func(c *theme.Couleurs) *string { return &c.Intensite[0] }, 3},
	{"INTENSITY 2", i18n.M("effort : moyen"), func(c *theme.Couleurs) *string { return &c.Intensite[1] }, 3},
	{"INTENSITY 3", i18n.M("effort : haut"), func(c *theme.Couleurs) *string { return &c.Intensite[2] }, 3},
	{"INTENSITY 4", i18n.M("effort : très haut"), func(c *theme.Couleurs) *string { return &c.Intensite[3] }, 3},
	{"INTENSITY 5", i18n.M("effort : max"), func(c *theme.Couleurs) *string { return &c.Intensite[4] }, 3},
}

// canaux : ce que ←→ règle, dans l'ordre de c ; pas : de combien à chaque appui.
var (
	canaux = []string{i18n.M("clarté"), i18n.M("saturation"), i18n.M("teinte")}
	pas    = [3]float64{2, 3, 5}
)

// niveauxApercu : les niveaux de couleurs que v fait défiler.
var niveauxApercu = []ansi.Niveau{ansi.Truecolor, ansi.Couleurs256, ansi.Couleurs16}

// editeur : l'état de l'onglet Créer.
type editeur struct {
	actif             bool
	nom, titre        string        // le fichier (themes/<nom>.theme) et le nom affiché
	nomGarde, titreG  string        // ce qui est enregistré (ou prévu, pour un nouveau thème)
	aCreer            bool          // le thème est à créer même sans autre changement (theme new, import)
	p, garde          theme.Palette // le brouillon, et ce qui est enregistré (la copie, pour un nouveau thème)
	nouveau           bool          // le fichier n'existe pas encore : s le crée
	role, canal       int
	saisie, saisieNom bool // on tape un code (Entrée) ou un nom (n)
	texte             string
	autreFond         bool        // f : l'aperçu sur l'autre fond
	niveau            int         // v : index dans niveauxApercu
	vision            ansi.Vision // a : l'aperçu tel que le voit une personne daltonienne
	message           string
}

// cleEditeur : la clé du brouillon dans le cache des palettes (Pip le dessine) ; jamais un nom de thème.
const cleEditeur = " éditeur"

func (e *editeur) modifie() bool {
	return e.actif && (e.aCreer || !reflect.DeepEqual(e.p, e.garde) || e.titre != e.titreG || e.nom != e.nomGarde)
}

// couleur : la couleur du rôle choisi (MODEL_END vide : celle de BRANCH, qu'elle reprend).
func (e *editeur) couleur(k int) string {
	c := *rolesTheme[k].champ(&e.p.Couleurs)
	if c == "" && rolesTheme[k].cle == "MODEL_END" {
		return e.p.Lieu
	}
	return c
}

// fondPropre : le fond pour lequel le thème est fait (celui des contrôles de contraste).
func (e *editeur) fondPropre() string {
	if e.p.Clair {
		return ansi.FondClair
	}
	return ansi.FondSombre
}

// editer : e (onglet Thèmes) ou l'onglet Créer : le thème de l'utilisateur lui-même, sinon une copie à un nom
// libre ; un brouillon en cours n'est pas remplacé.
func (m *Modele) editer(nom string) {
	m.onglet = 2
	if m.ed.modifie() {
		m.ed.message = i18n.T("un thème est en cours : s l'enregistre, u l'annule")
		return
	}
	p, ok := m.sources.Charger(nom)
	if !ok {
		p = theme.Secours()
	}
	titre, _ := m.sources.Entete(nom)
	e := editeur{actif: true, nom: nom, titre: titre, p: p, garde: p}
	if !m.sources.DeLUtilisateur(nom) {
		e.nom, e.titre, e.nouveau = m.nomLibre(nom+i18n.T("-perso")), i18n.T("%s (perso)", titre), true
	}
	e.nomGarde, e.titreG = e.nom, e.titre
	m.ed = e
	m.majEditeur()
}

// OuvrirEditeur : le sélecteur s'ouvre sur l'onglet Créer (agentline theme new, edit) ; nom : celui du
// nouveau thème (vide : un nom libre).
func (m *Modele) OuvrirEditeur(base, nom string) {
	m.editer(base)
	if nom != "" && m.ed.nouveau {
		t := []rune(strings.ReplaceAll(nom, "-", " "))
		m.ed.nom, m.ed.titre, m.ed.aCreer = nom, strings.ToUpper(string(t[:1]))+string(t[1:]), true
	}
	m.ajuster()
}

// OuvrirImport : le sélecteur s'ouvre sur l'onglet Créer avec un thème importé (agentline theme import), à
// retoucher puis à enregistrer.
func (m *Modele) OuvrirImport(p theme.Palette, titre string) {
	m.onglet = 2
	nom := theme.NomDeTitre(titre)
	if !theme.NomValide(nom) {
		nom = i18n.T("importe")
	}
	nom = m.nomLibre(nom)
	m.ed = editeur{actif: true, nom: nom, titre: titre, nomGarde: nom, titreG: titre, p: p, garde: p, nouveau: true, aCreer: true}
	m.majEditeur()
	m.ajuster()
}

// nomLibre : n, ou n-2, n-3… si un thème porte déjà ce nom.
func (m *Modele) nomLibre(n string) string {
	if _, pris := m.sources.Fichier(n); !pris {
		return n
	}
	for k := 2; ; k++ {
		x := n + "-" + strconv.Itoa(k)
		if _, pris := m.sources.Fichier(x); !pris {
			return x
		}
	}
}

// majEditeur : après un changement, Pip et les aperçus prennent le brouillon.
func (m *Modele) majEditeur() {
	m.palettes[cleEditeur] = simulee(m.ed.p, m.ed.vision).PourFond(m.clair())
}

// simulee : la palette telle que la voit une personne de cette vision.
func simulee(p theme.Palette, v ansi.Vision) theme.Palette {
	if v == ansi.VisionNormale {
		return p
	}
	for _, c := range []*theme.Couleurs{&p.Couleurs, &p.Autre} {
		for _, r := range rolesTheme {
			x := r.champ(c)
			*x = ansi.Simuler(*x, v)
		}
	}
	return p
}

// nomsVisions : les visions de a, en peu de mots.
var nomsVisions = []string{i18n.M("vision normale"), i18n.M("protanopie"), i18n.M("deutéranopie"), i18n.M("tritanopie")}

// generer : g, les seuils et l'intensité accordés à la couleur choisie ; G, tout le thème autour d'elle.
func (m *Modele) generer(tout bool) tea.Cmd {
	e := &m.ed
	c := e.couleur(e.role)
	if tout {
		e.p.Couleurs = theme.Harmoniser(c, e.p.Clair).Couleurs
	} else {
		e.p = theme.Seuils(e.p, c)
	}
	m.majEditeur()
	return nil
}

// confusions : en vision simulée, les seuils voisins qui ne se distinguent plus (pour l'avertir).
func (e *editeur) confusions() []string {
	if e.vision == ansi.VisionNormale {
		return nil
	}
	s := simulee(e.p, e.vision)
	seuils := [][2]string{{"D0", s.D0}, {"D50", s.D50}, {"D75", s.D75}, {"D100", s.D100}}
	var l []string
	for k := 1; k < len(seuils); k++ {
		if ansi.Ecart(seuils[k-1][1], seuils[k][1]) < 10 {
			l = append(l, seuils[k-1][0]+"/"+seuils[k][0])
		}
	}
	return l
}

// reglerCouleur : ←→ (Maj : 5 fois plus) sur le canal choisi, en CIELAB.
func (m *Modele) reglerCouleur(d int) tea.Cmd {
	e := &m.ed
	l, c, t := ansi.LCh(e.couleur(e.role))
	v := float64(d) * pas[e.canal]
	switch e.canal {
	case 0:
		l = math.Max(0, math.Min(100, l+v))
	case 1:
		c = math.Max(0, math.Min(150, c+v))
	default:
		t = math.Mod(t+v+360, 360)
	}
	*rolesTheme[e.role].champ(&e.p.Couleurs) = ansi.DepuisLCh(l, c, t)
	m.majEditeur()
	return nil
}

// validerSaisie : Entrée pendant la saisie : un code (rrggbb, # permis ; vide pour MODEL_END) ou un nom.
func (m *Modele) validerSaisie() tea.Cmd {
	e := &m.ed
	t := strings.TrimSpace(e.texte)
	if e.saisieNom {
		nom := theme.NomDeTitre(t)
		switch {
		case t == "" || !theme.NomValide(nom):
			e.message = i18n.T("⚠ nom vide : des lettres ou des chiffres")
			return nil
		case nom != e.nomGarde && m.existe(nom):
			e.message = i18n.T("⚠ un thème s'appelle déjà %s", nom)
			return nil
		}
		e.nom, e.titre, e.saisie, e.saisieNom = nom, t, false, false
		return nil
	}
	h := strings.ToLower(strings.TrimPrefix(t, "#"))
	vide := h == "" && rolesTheme[e.role].cle == "MODEL_END"
	if !vide && !codeCouleur(h) {
		e.message = i18n.T("⚠ code de couleur : 6 chiffres hexadécimaux (bb9af7)")
		return nil
	}
	*rolesTheme[e.role].champ(&e.p.Couleurs) = h
	e.saisie = false
	m.majEditeur()
	return nil
}

func (m *Modele) existe(nom string) bool { _, ok := m.sources.Fichier(nom); return ok }

func codeCouleur(h string) bool {
	if len(h) != 6 {
		return false
	}
	_, err := strconv.ParseUint(h, 16, 32)
	return err == nil
}

// --- Dessin ----------------------------------------------------------------------------------------

// apercuEditeur : les statuslines d'exemple (complète, tendue, API) au brouillon, sur le fond et au niveau
// de couleurs choisis, peintes sur ce fond.
func (m *Modele) apercuEditeur(w int) []string {
	e := &m.ed
	clair := e.p.Clair != e.autreFond
	env := m.apercu
	p := simulee(e.p, e.vision)
	env.Palette, env.Style, env.Columns = &p, m.os.st, w+4
	env.Couleurs.Fond, env.Couleurs.Niveau = ansi.Sombre, niveauxApercu[e.niveau]
	fond := ansi.FondSombre
	if clair {
		env.Couleurs.Fond, fond = ansi.Clair, ansi.FondClair
	}
	var l []string
	for k := 0; k < 3; k++ {
		s, g := sessionApercu(env.Now, k)
		env.GitFictif = g
		for _, x := range strings.Split(strings.TrimRight(render.RenderSession(s, env), "\n"), "\n") {
			x = tronquer(x+z, w)
			f := ansi.Bg(fond)
			l = append(l, f+strings.ReplaceAll(x, z, z+f)+espaces(w-largeur(x))+z)
		}
	}
	return l
}

// lignesRoles : les rôles, chacun avec sa pastille, son code, son contraste (⚠ s'il est trop faible) et ce
// qu'il colore, sur une ou deux colonnes de largeur cw.
func (m *Modele) ligneRole(k, cw int) string {
	e := &m.ed
	r := rolesTheme[k]
	c := e.couleur(k)
	code := *r.champ(&e.p.Couleurs)
	if code == "" {
		code = "------"
	}
	contraste := "     "
	if r.seuil > 0 {
		v := ansi.Contraste(c, e.fondPropre())
		contraste = strconv.FormatFloat(v, 'f', 1, 64)
		if v < r.seuil {
			contraste = "⚠" + contraste
		}
		contraste = espaces(5-largeur(contraste)) + contraste
	}
	debut := "  "
	if k == e.role {
		debut = gras + "› " + z
	}
	l := debut + ansi.Fg(c) + "██" + z + " " + remplir(r.cle, 11) + " " + code + " " + estompe + contraste + "  " + i18n.T(r.aide) + z
	if k == e.role {
		l = debut + ansi.Fg(c) + "██" + z + " " + gras + remplir(r.cle, 11) + z + " " + code + " " + contraste + "  " + i18n.T(r.aide)
	}
	return tronquer(l, cw)
}

// ligneCanaux : la clarté, la saturation et la teinte du rôle choisi, le canal de ←→ en évidence.
func (m *Modele) ligneCanaux() string {
	e := &m.ed
	l, c, t := ansi.LCh(e.couleur(e.role))
	vals := []float64{l, c, t}
	var x []string
	for k, n := range canaux {
		v := i18n.T(n) + " " + strconv.Itoa(int(math.Round(vals[k])))
		if k == e.canal {
			v = inverse + " " + v + " " + z
		} else {
			v = estompe + v + z
		}
		x = append(x, v)
	}
	return strings.Join(x, "  ")
}

// corpsEditeur : Pip et l'aperçu dans son cadre, puis les rôles, puis les canaux du rôle choisi.
func (m *Modele) corpsEditeur() []string {
	e := &m.ed
	w := max(m.largeur, largeurMin) - 1
	bw := min(w, 96)
	marge := espaces((w - bw) / 2)
	var corps []string
	pattes := ""
	if m.d.mascotte && m.d.corps >= 26 {
		var cl [hauteurMascotte]string
		cl, pattes, _ = m.Mascotte(cleEditeur)
		for _, l := range cl {
			corps = append(corps, marge+espaces(bw-largeurMascotte-3)+l)
		}
	}
	fond := i18n.T("fond sombre")
	if e.p.Clair != e.autreFond {
		fond = i18n.T("fond clair")
	}
	etat := ""
	if e.nouveau {
		etat = " · " + i18n.T("nouveau")
	}
	if e.vision != ansi.VisionNormale {
		fond += " · " + i18n.T(nomsVisions[e.vision])
	}
	titre := "╭─ " + e.titre + " · " + e.nom + ".theme" + etat + " · " + fond + " · " + nomsNiveaux(niveauxApercu[e.niveau]) + " "
	if largeur(titre)+2 > bw {
		titre = "╭─"
	}
	cadre := []rune(titre + strings.Repeat("─", max(bw-largeur(titre)-1, 0)) + "╮")
	haut := string(cadre)
	if x := bw - largeurMascotte - 3; pattes != "" && x > largeur(titre) {
		haut = string(cadre[:x]) + strings.ReplaceAll(pattes, " ", "─") + z + string(cadre[x+largeurMascotte:])
	}
	corps = append(corps, marge+haut)
	for _, l := range m.apercuEditeur(bw - 4) {
		corps = append(corps, marge+"│ "+l+" │")
	}
	corps = append(corps, marge+"╰"+strings.Repeat("─", max(bw-2, 0))+"╯", "")
	// les rôles : deux colonnes si la place le permet ; faute de hauteur, une colonne qui défile autour du
	// rôle choisi (les canaux et le message restent visibles dessous)
	n := len(rolesTheme)
	place := m.d.corps - len(corps) - 2 - b2i(e.message != "") - b2i(len(e.confusions()) > 0)
	cols, cw := 1, bw
	if bw >= 92 && (n+1)/2 <= place {
		cols, cw = 2, (bw-2)/2
	}
	parCol := (n + cols - 1) / cols
	debut, fin := 0, parCol
	if cols == 1 && n > place {
		fin = max(place, 1)
		debut = min(max(e.role-fin/2, 0), n-fin)
		fin += debut
	}
	for i := debut; i < fin; i++ {
		l := ""
		for c := 0; c < cols; c++ {
			if k := c*parCol + i; k < n {
				l += remplir(m.ligneRole(k, cw), cw) + "  "
			}
		}
		corps = append(corps, marge+strings.TrimRight(l, " "))
	}
	corps = append(corps, "", marge+"  "+m.ligneCanaux())
	if c := e.confusions(); len(c) > 0 {
		corps = append(corps, marge+"  "+gras+i18n.T("⚠ en %s, ces seuils se confondent : %s", i18n.T(nomsVisions[e.vision]), strings.Join(c, ", "))+z)
	}
	if e.message != "" {
		corps = append(corps, marge+"  "+gras+e.message+z)
	}
	return corps
}

// nomsNiveaux : le niveau de couleurs de l'aperçu, en peu de mots.
func nomsNiveaux(n ansi.Niveau) string {
	switch n {
	case ansi.Couleurs256:
		return i18n.T("256 couleurs")
	case ansi.Couleurs16:
		return i18n.T("16 couleurs")
	}
	return i18n.T("24 bits")
}
