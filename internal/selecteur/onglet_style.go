package selecteur

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/style"
)

// L'onglet « Statusline » : les items de la statusline groupés par ligne (affichés ou masqués, dans quel
// ordre, collés ou non), leurs options, et l'aperçu en direct de la vraie statusline (au thème survolé
// dans l'onglet Thèmes) avec sa ligne SSH. Rien n'est écrit avant la validation (Entrée, puis d).

// Les groupes de la liste.
const (
	gLigne1 = iota
	gLigne2
	gCourt
	gGeneral
)

// nomsGroupes : les titres des groupes (en français, traduits à l'affichage).
var nomsGroupes = [...]string{i18n.M("Ligne 1"), i18n.M("Ligne 2"), i18n.M("Ligne SSH"), i18n.M("Général")}

// clesGroupes : l'option de disposition de chaque groupe.
var clesGroupes = [...]string{"line1", "line2", "short"}

// Les aperçus (touche v), en français, traduits à l'affichage.
var nomsApercus = []string{i18n.M("complète"), i18n.M("tendue"), "API", "SSH"}

// ongletStyle : l'état de l'onglet.
type ongletStyle struct {
	st, garde  *style.Style // le style en cours, et celui du fichier
	ver        int          // incrémenté à chaque modification (caches des aperçus)
	k, haut    int          // rangée choisie, début de la liste affichée
	options    bool         // le panneau des options de l'item est ouvert
	opt        int          // option choisie dans le panneau
	saisie     bool         // on tape une valeur (libellé, glyphe)
	texte      string       // ce qui est tapé
	confirmer  bool         // la question « où garder ce style ? » est posée
	apercu     int          // index dans nomsApercus
	message    string       // une information ou une erreur, effacée à la touche suivante
	ignore     bool         // AGENTLINE_STYLE=default : le style gardé ne se verra pas
	rendus     map[string]string
	lignesVues int // rangées visibles, selon la disposition (calculé par ajuster)
}

// place : un item dans une ligne, et s'il est collé à celui d'avant (même segment).
type place struct {
	nom   string
	colle bool
}

func lirePlaces(v string) []place {
	segs, _ := style.LireDisposition(v)
	var p []place
	for _, s := range segs {
		for k, n := range s {
			p = append(p, place{n, k > 0})
		}
	}
	return p
}

func ecrirePlaces(p []place) string {
	var segs [][]string
	for k, x := range p {
		if k == 0 || !x.colle {
			segs = append(segs, nil)
		}
		segs[len(segs)-1] = append(segs[len(segs)-1], x.nom)
	}
	return style.EcrireDisposition(segs)
}

func (o *ongletStyle) places(g int) []place { return lirePlaces(o.st.Valeur(clesGroupes[g])) }

func (o *ongletStyle) poser(g int, p []place) {
	if len(p) > 0 {
		p[0].colle = false
	}
	o.st.Definir(clesGroupes[g], ecrirePlaces(p))
}

// rangee : une ligne de la liste : l'en-tête d'un groupe, un item, ou les options générales.
type rangee struct {
	groupe int
	titre  bool
	item   string
	pos    int // place dans sa ligne (-1 : absent de la ligne)
	colle  bool
}

func contient(p []place, n string) bool {
	for _, x := range p {
		if x.nom == n {
			return true
		}
	}
	return false
}

// rangees : la liste de l'onglet. Ligne 2 finit par les items absents des deux lignes longues, Ligne SSH
// par ceux absents de la ligne courte (Espace les y ajoute).
func (o *ongletStyle) rangees() []rangee {
	var l []rangee
	l1, l2, c := o.places(gLigne1), o.places(gLigne2), o.places(gCourt)
	for g, p := range [][]place{l1, l2, c} {
		if g == gCourt && !o.court() {
			continue
		}
		l = append(l, rangee{groupe: g, titre: true})
		for k, x := range p {
			l = append(l, rangee{groupe: g, item: x.nom, pos: k, colle: x.colle})
		}
		if g == gLigne1 {
			continue
		}
		for _, n := range style.Items {
			if g == gLigne2 && !contient(l1, n) && !contient(l2, n) || g == gCourt && !contient(c, n) {
				l = append(l, rangee{groupe: g, item: n, pos: -1})
			}
		}
	}
	return append(l, rangee{groupe: gGeneral, titre: true}, rangee{groupe: gGeneral, pos: -1})
}

// court : la ligne courte de SSH est demandée (ssh.short=yes) ; sinon l'onglet ne la montre pas.
func (o *ongletStyle) court() bool { return o.st.Oui("ssh.short") }

// apercuVu : l'aperçu montré (celui de SSH seulement avec la ligne courte).
func (o *ongletStyle) apercuVu() int {
	if o.apercu == 3 && !o.court() {
		return 0
	}
	return o.apercu
}

func (o *ongletStyle) courante() rangee {
	l := o.rangees()
	return l[min(max(o.k, 0), len(l)-1)]
}

// viser : la rangée de cet item dans ce groupe (sinon la plus proche de la position actuelle).
func (o *ongletStyle) viser(g int, item string) {
	for k, r := range o.rangees() {
		if !r.titre && r.groupe == g && r.item == item {
			o.k = k
			return
		}
	}
	o.parcourir(0)
}

// parcourir : d rangées plus bas (ou plus haut), en sautant les en-têtes.
func (o *ongletStyle) parcourir(d int) {
	l := o.rangees()
	k := min(max(o.k+d, 0), len(l)-1)
	pas := 1
	if d < 0 {
		pas = -1
	}
	for k >= 0 && k < len(l) && l[k].titre {
		k += pas
	}
	if k < 0 || k >= len(l) { // en bout de liste : la première rangée valide dans l'autre sens
		k = min(max(o.k, 0), len(l)-1)
		for l[k].titre {
			k -= pas
		}
	}
	o.k = k
}

func (o *ongletStyle) changer() { o.ver++ }

// basculer : Espace. Lignes longues : masquer ou afficher (partout) ; un item absent est ajouté à la fin
// de la ligne 2. Ligne SSH : retirer de la ligne courte, ou l'y ajouter.
func (o *ongletStyle) basculer() {
	r := o.courante()
	switch {
	case r.groupe == gGeneral:
		o.message = i18n.T("→ : les options générales (icônes, séparateurs)")
		return
	case r.groupe == gCourt && r.pos >= 0:
		p := o.places(gCourt)
		o.poser(gCourt, append(p[:r.pos:r.pos], p[r.pos+1:]...))
	case r.pos < 0:
		o.poser(r.groupe, append(o.places(r.groupe), place{nom: r.item}))
		o.st.Masquer(r.item, false)
	default:
		o.st.Masquer(r.item, !o.st.Masques()[r.item])
	}
	o.changer()
	o.viser(r.groupe, r.item)
}

// deplacer : Maj+↑↓. L'item échange sa place avec son voisin (les segments gardent leur forme) ; entre
// la ligne 1 et la ligne 2, il passe d'une ligne à l'autre, seul dans son segment.
func (o *ongletStyle) deplacer(d int) {
	r := o.courante()
	if r.titre || r.pos < 0 {
		o.message = i18n.T("Espace d'abord : cet item n'est pas dans la ligne")
		return
	}
	p := o.places(r.groupe)
	j := r.pos + d
	switch {
	case j >= 0 && j < len(p):
		p[r.pos].nom, p[j].nom = p[j].nom, p[r.pos].nom
		o.poser(r.groupe, p)
		o.changer()
		o.viser(r.groupe, r.item)
		return
	case r.groupe == gLigne1 && d > 0, r.groupe == gLigne2 && d < 0:
		autre := gLigne2
		if r.groupe == gLigne2 {
			autre = gLigne1
		}
		q := o.places(autre)
		if d > 0 {
			q = append([]place{{nom: r.item}}, q...)
		} else {
			q = append(q, place{nom: r.item})
		}
		o.poser(r.groupe, append(p[:r.pos:r.pos], p[r.pos+1:]...))
		o.poser(autre, q)
		o.changer()
		o.viser(autre, r.item)
		return
	}
	o.message = i18n.T("déjà en bout de %s", strings.ToLower(i18n.T(nomsGroupes[r.groupe])))
}

// coller : g. L'item rejoint le segment de l'item précédent, ou s'en détache.
func (o *ongletStyle) coller() {
	r := o.courante()
	if r.titre || r.pos < 0 {
		return
	}
	if r.pos == 0 {
		o.message = i18n.T("premier de la ligne : rien à quoi le coller")
		return
	}
	p := o.places(r.groupe)
	p[r.pos].colle = !p[r.pos].colle
	o.poser(r.groupe, p)
	o.changer()
	o.viser(r.groupe, r.item)
}

// cles : les options du panneau (celles de l'item, ou les générales).
func (o *ongletStyle) cles() []style.Cle {
	r := o.courante()
	if r.groupe == gGeneral {
		return style.DesItems("")
	}
	return style.DesItems(r.item)
}

func (o *ongletStyle) cle() style.Cle {
	c := o.cles()
	return c[min(max(o.opt, 0), len(c)-1)]
}

// ouvrir : →, le panneau des options de l'item.
func (o *ongletStyle) ouvrir() {
	r := o.courante()
	if r.titre {
		return
	}
	if len(o.cles()) == 0 {
		o.message = i18n.T("pas d'option pour %s : Espace l'affiche ou le masque", r.item)
		return
	}
	o.options, o.opt = true, 0
}

// valeursDe : ce que ←→ fait défiler pour une option (la valeur actuelle comprise).
func valeursDe(c style.Cle, actuelle string) []string {
	var v []string
	switch c.Genre {
	case style.Choix, style.Barre:
		v = append(v, c.Valeurs...)
	case style.Glyphe:
		if n, ok := strings.CutPrefix(c.Nom, "icon."); ok {
			v = []string{style.Auto, style.GlypheDuJeu("nerd", n), style.GlypheDuJeu("text", n), ""}
		} else {
			v = []string{c.Defaut, "│", "·", "|", "/", "•", "»", ""}
		}
	case style.Libelle:
		v = []string{c.Defaut, ""}
	}
	var u []string
	for _, x := range append(v, actuelle) {
		if !contientTexte(u, x) {
			u = append(u, x)
		}
	}
	return u
}

func contientTexte(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// regler : ←→ sur l'option choisie.
func (o *ongletStyle) regler(d int) {
	c := o.cle()
	v := o.st.Valeur(c.Nom)
	if c.Genre == style.Nombre {
		n, _ := strconv.Atoi(v)
		v = strconv.Itoa(min(max(n+d, c.Min), c.Max))
	} else {
		l := valeursDe(c, v)
		k := 0
		for i, x := range l {
			if x == v {
				k = i
			}
		}
		v = l[(k+d+len(l))%len(l)]
	}
	if err := o.st.Definir(c.Nom, v); err != nil {
		o.message = "⚠ " + err.Error()
		return
	}
	o.changer()
}

// saisissable : l'option se règle aussi en tapant sa valeur.
func saisissable(c style.Cle) bool {
	return c.Genre == style.Libelle || c.Genre == style.Glyphe || c.Genre == style.Barre
}

// valider : Entrée pendant la saisie ; une valeur refusée (glyphe trop large…) le dit et reste à corriger.
func (o *ongletStyle) valider() {
	c := o.cle()
	if err := o.st.Definir(c.Nom, o.texte); err != nil {
		o.message = i18n.T("⚠ %s : %s", c.Nom, err.Error())
		return
	}
	o.saisie = false
	o.changer()
}

// remettre : r, les options de l'item à leur défaut (dans le panneau : l'option choisie seulement).
func (o *ongletStyle) remettre() {
	r := o.courante()
	if o.options {
		o.st.Retirer(o.cle().Nom)
	} else {
		for _, c := range o.cles() {
			o.st.Retirer(c.Nom)
		}
		if r.item != "" {
			o.st.Masquer(r.item, false)
		}
	}
	o.changer()
}

func (o *ongletStyle) modifie() bool { return !o.st.Egal(o.garde) }

// --- Dans le modèle --------------------------------------------------------------------------------

func (m *Modele) initStyle() {
	m.os.garde = style.LireFichier(m.sources.Config)
	m.os.st = m.os.garde.Copie()
	m.os.ignore = style.Force(m.getenv)
	m.os.rendus = map[string]string{}
	m.os.parcourir(0)
}

// OuvrirStyle : le sélecteur s'ouvre sur l'onglet Statusline (agentline style).
func (m *Modele) OuvrirStyle() {
	m.onglet = 1
	m.ajuster()
}

func (m *Modele) basculerOnglet() tea.Cmd {
	m.onglet = 1 - m.onglet
	m.reagir("surpris")
	return nil
}

// quitterOuAvertir : q ou Échap pour quitter ; des modifications du style non gardées le font dire une
// fois, la touche suivante quitte.
func (m *Modele) quitterOuAvertir() tea.Cmd {
	if m.os.modifie() && !m.avertiQuitter {
		m.avertiQuitter = true
		return nil
	}
	return tea.Quit
}

// garderStyle : d, le style en cours écrit pour tous les projets.
func (m *Modele) garderStyle() tea.Cmd {
	if err := style.Ecrire(m.sources.Config, m.os.st); err != nil {
		m.Resultat = erreur(err)
		return tea.Quit
	}
	n := 0
	for _, c := range style.Cles() {
		if m.os.st.Regle(c.Nom) {
			n++
		}
	}
	m.os.garde = m.os.st.Copie()
	if n == 0 {
		m.Resultat = i18n.T("✓ Style par défaut pour la statusline : %s", visibleBientot())
	} else {
		m.Resultat = i18n.N(n, "✓ Style de la statusline gardé (%d réglage, agentline style show) : %s", "✓ Style de la statusline gardé (%d réglages, agentline style show) : %s", n, visibleBientot())
	}
	if m.os.ignore {
		m.Resultat += "\n" + i18n.T("⚠ AGENTLINE_STYLE=default : il ne se verra pas tant qu'elle est définie.")
	}
	return tea.Quit
}

// --- Aperçus ---------------------------------------------------------------------------------------

// sessionApercu : la session fictive de l'aperçu k (complète, tendue, API) et son état git.
func sessionApercu(now int64, k int) (etat.Session, *render.GitFictif) {
	s := etat.Exemple(now, "/apercu/"+i18n.T("mon-projet"), 42)
	switch k {
	case 1: // tendue : alertes, conflit, plafond
		s.Effort, s.Rapide, s.Contexte = "max", true, "91"
		s.Quotas.H5 = etat.Quota{Utilise: "95", Reset: strconv.FormatInt(now+1800, 10)}
		s.Quotas.J7 = etat.Quota{Utilise: "80", Reset: strconv.FormatInt(now+86400, 10)}
		s.PR = etat.PR{Num: "7", Revue: "changes_requested"}
		return s, &render.GitFictif{Branche: "main", Modifie: 3, Conflit: 2, Retard: 2}
	case 2: // API : facturation à l'usage
		s.Modele, s.Effort, s.Contexte, s.Quotas, s.Cout = "Sonnet 5", "low", "12", etat.Quotas{}, "4.98"
		s.Agent, s.Style, s.Worktree = i18n.T("relecteur"), "Explanatory", i18n.T("essai")
		s.PR = etat.PR{Num: "12", Revue: "draft"}
		return s, &render.GitFictif{Branche: i18n.T("feat/essai"), Indexe: 1, Avance: 1}
	}
	s.Session, s.PR = i18n.T("refonte de la statusline"), etat.PR{Num: "12", Revue: "pending"}
	return s, &render.GitFictif{Branche: "main", Neuf: 1}
}

// rendre : la statusline de la session k au thème donné, avec le style en cours, sur w colonnes.
func (m *Modele) rendreStyle(nom string, w, k int, ssh bool) []string {
	cle := nom + "@" + strconv.Itoa(w) + "@" + strconv.Itoa(k) + "@" + strconv.FormatBool(ssh) + "@" + strconv.Itoa(m.os.ver)
	if r, ok := m.os.rendus[cle]; ok {
		return strings.Split(r, "\n")
	}
	if len(m.os.rendus) > 400 {
		m.os.rendus = map[string]string{}
	}
	e := m.apercu
	e.Columns, e.SSH, e.SSHRiche, e.Style = w+4, ssh, ssh, m.os.st
	e.Getenv = func(c string) string {
		if c == "AGENTLINE_THEME" {
			return nom
		}
		return ""
	}
	s, g := sessionApercu(e.Now, k)
	e.GitFictif = g
	r := strings.TrimRight(render.RenderSession(s, e), "\n")
	m.os.rendus[cle] = r
	return strings.Split(r, "\n")
}

// lignesApercu : les trois lignes de l'aperçu sur w colonnes : la statusline et sa ligne SSH, ou (aperçu
// SSH) la ligne courte des trois sessions.
func (m *Modele) lignesApercu(nom string, w int) []string {
	var l []string
	ssh := estompe + "SSH " + z
	a := m.os.apercuVu()
	if a == 3 {
		for k := 0; k < 3; k++ {
			x := m.rendreStyle(nom, w-4, k, true)
			l = append(l, ssh+x[0])
		}
		return l
	}
	long := m.rendreStyle(nom, w, a, false)
	for k := 0; k < 2; k++ {
		x := ""
		if k < len(long) {
			x = long[k]
		}
		l = append(l, x)
	}
	if !m.os.court() {
		return l
	}
	return append(l, ssh+m.rendreStyle(nom, w-4, a, true)[0])
}

// boiteApercu : l'aperçu dans un cadre de largeur w, Clawd (ses pattes) posé sur le cadre.
func (m *Modele) boiteApercu(nom string, w int, pattes string) []string {
	iw := w - 4
	titre := "╭─ " + i18n.T("Aperçu · %s (v) ", i18n.T(nomsApercus[m.os.apercuVu()]))
	if largeur(titre)+2 > w {
		titre = "╭─"
	}
	cadre := []rune(titre + strings.Repeat("─", max(w-largeur(titre)-1, 0)) + "╮")
	haut := string(cadre)
	if x := w - largeurClawd - 3; pattes != "" && x > largeur(titre) {
		haut = string(cadre[:x]) + strings.ReplaceAll(pattes, " ", "─") + z + string(cadre[x+largeurClawd:])
	}
	l := []string{haut}
	for _, c := range m.lignesApercu(nom, iw) {
		if largeur(c) > iw {
			c = couper(c, iw)
		}
		l = append(l, "│ "+c+z+espaces(iw-largeur(c))+" │")
	}
	return m.surFond(append(l, "╰"+strings.Repeat("─", max(w-2, 0))+"╯"))
}

// --- Dessin ----------------------------------------------------------------------------------------

// ligneRangee : une rangée de la liste, sur w colonnes.
func (m *Modele) ligneRangee(r rangee, sel bool, w int) string {
	o := &m.os
	if r.titre {
		return remplir(gras+" "+i18n.T(nomsGroupes[r.groupe])+z, w)
	}
	t, suite := "", ""
	switch {
	case r.groupe == gGeneral:
		t = "  " + i18n.T("icônes, séparateurs")
	case r.groupe == gCourt:
		boite := "☑ "
		if r.pos < 0 {
			boite, suite = "☐ ", i18n.T("absent")
		} else if o.st.Masques()[r.item] {
			suite = i18n.T("masqué")
		}
		t = boite + colle(r) + r.item
	default:
		boite := "☑ "
		if r.pos < 0 {
			boite, suite = "☐ ", i18n.T("absent")
		} else if o.st.Masques()[r.item] {
			boite = "☐ "
		}
		t = boite + colle(r) + r.item
	}
	if suite != "" && largeur(t)+largeur(suite)+6 <= w {
		t += " " + estompe + suite + z
		if sel {
			t += inverse
		}
	}
	t = tronquer(t, w-4)
	if sel {
		return inverse + " › " + t + espaces(w-3-largeur(t)) + z
	}
	return "   " + t + espaces(w-3-largeur(t))
}

func colle(r rangee) string {
	if r.colle {
		return "+ "
	}
	return ""
}

// colonneStyle : la liste des rangées, h lignes de w colonnes à partir de os.haut.
func (m *Modele) colonneStyle(w, h int) []string {
	l := m.os.rangees()
	var c []string
	for k := m.os.haut; k < len(l) && len(c) < h; k++ {
		c = append(c, m.ligneRangee(l[k], k == m.os.k, w))
	}
	if m.os.haut > 0 && len(c) > 0 {
		c[0] = indicateur(i18n.T("↑ %d de plus", m.os.haut), w)
	}
	if reste := len(l) - m.os.haut - h; reste > 0 && len(c) > 0 {
		c[len(c)-1] = indicateur(i18n.T("↓ %d de plus", reste+1), w)
	}
	for len(c) < h {
		c = append(c, espaces(w))
	}
	return c
}

// valeurAffichee : la valeur d'une option, telle que le panneau la montre.
func (m *Modele) valeurAffichee(c style.Cle) string {
	v := m.os.st.Valeur(c.Nom)
	if n, ok := strings.CutPrefix(c.Nom, "icon."); ok && v == style.Auto {
		return "auto (" + montre(m.os.st.Glyphe(n)) + ")"
	}
	if l := style.LibelleEnVigueur(c.Nom, v); l != v {
		return "auto (" + montre(l) + ")"
	}
	return montre(v)
}

func montre(v string) string {
	if v == "" {
		return i18n.T("aucun")
	}
	return v
}

// panneauOptions : le titre, une ligne par option (◂ valeur ▸ sur celle choisie), puis l'aide de l'option
// choisie, repliée sur w colonnes.
func (m *Modele) panneauOptions(w int) []string {
	o := &m.os
	r := o.courante()
	quoi := r.item
	if r.groupe == gGeneral {
		quoi = i18n.T("générales")
	}
	l := []string{gras + tronquer(i18n.T("Options · %s", quoi), w) + z}
	cles := o.cles()
	gauche := 0
	for _, c := range cles {
		gauche = max(gauche, largeur(c.Nom))
	}
	gauche = min(gauche, w/2)
	for k, c := range cles {
		v := m.valeurAffichee(c)
		nom := tronquer(c.Nom, gauche)
		x := "  " + nom + espaces(gauche-largeur(nom)) + "  "
		switch {
		case k == o.opt && o.saisie:
			x += "› " + gras + o.texte + z + "▏"
		case k == o.opt:
			x += "◂ " + gras + v + z + " ▸"
		default:
			x += estompe + v + z
		}
		if o.st.Regle(c.Nom) && !(k == o.opt && o.saisie) {
			x += estompe + " •" + z
		}
		if k == o.opt { // le curseur devant le nom, comme dans la liste ; l'aide en dessous sur la même colonne
			x = "› " + x[2:]
		}
		l = append(l, tronquer(x, w))
	}
	if len(cles) > 0 {
		c := o.cle()
		aide := c.Explication()
		if !c.SSH && o.court() {
			aide += i18n.T(" (pas en SSH)")
		}
		if m.d.agencement == compacte { // une ligne : l'aperçu passe avant
			return append(l, "  "+estompe+tronquer(aide, w-2)+z)
		}
		for _, x := range replier(strings.Fields(aide), " ", w-2) {
			l = append(l, "  "+estompe+x+z)
		}
	}
	return l
}

// indice : sous l'aperçu, quand le panneau est fermé : ce que l'item choisi permet.
func (m *Modele) indice(w int) []string {
	r := m.os.courante()
	var t string
	switch {
	case r.groupe == gGeneral:
		t = i18n.T("→ : jeu d'icônes, séparateurs, signal d'alerte")
	case len(m.os.cles()) > 0:
		var n []string
		for _, c := range m.os.cles() {
			n = append(n, c.Nom)
		}
		t = i18n.T("→ : %s", strings.Join(n, ", "))
	default:
		t = i18n.T("%s n'a pas d'option", r.item)
	}
	var l []string
	for _, x := range replier(strings.Fields(t), " ", w) {
		l = append(l, estompe+x+z)
	}
	return l
}

// hauteurOptions : les lignes du panneau (ou de l'indice) sous la liste, en empilé et en compact.
func (m *Modele) blocOptions(w int) []string {
	if m.os.options {
		return m.panneauOptions(w)
	}
	return m.indice(w)
}

// lignesApercuStyle : la hauteur du cadre de l'aperçu (la ligne SSH en plus avec la ligne courte).
func (m *Modele) lignesApercuStyle() int { return 4 + b2i(m.os.court()) }

// hauteurListeStyle : les rangées visibles de la liste dans la disposition en cours : en empilé et en
// compact, ce que laissent les options et l'aperçu (une ligne vide, puis le cadre, ou en compact les
// trois lignes seules).
func (m *Modele) hauteurListeStyle() int {
	d := m.d
	if d.agencement == large {
		return max(d.corps, 3)
	}
	apercu := 1 + m.lignesApercuStyle()
	if d.agencement == compacte {
		apercu = 1 + m.lignesApercuStyle() - 2
	}
	return max(d.corps-apercu-len(m.blocOptions(d.lw-1)), 3)
}

// corpsStyle : le corps de l'onglet, selon la disposition.
func (m *Modele) corpsStyle(nom string) []string {
	d := m.d
	h := m.hauteurListeStyle()
	if d.agencement == large {
		col := m.colonneStyle(d.lw, h)
		var droite []string
		pattes := ""
		if d.clawd && d.corps >= hauteurClawd+m.lignesApercuStyle()+4 {
			var cl [hauteurClawd]string
			cl, pattes, _ = m.Clawd(nom)
			x := espaces(d.ww - largeurClawd - 3)
			for _, l := range cl {
				droite = append(droite, x+l)
			}
		}
		droite = append(droite, m.boiteApercu(nom, d.ww, pattes)...)
		droite = append(droite, "")
		for _, x := range m.blocOptions(d.ww - 2) { // sur la colonne du texte de l'aperçu
			droite = append(droite, "  "+x)
		}
		var corps []string
		for k := 0; k < max(len(col), len(droite)); k++ {
			l := espaces(d.lw)
			if k < len(col) {
				l = col[k]
			}
			if k < len(droite) {
				l += "  " + droite[k]
			}
			corps = append(corps, l)
		}
		return corps
	}
	w := d.lw
	corps := m.colonneStyle(w, h)
	for _, x := range m.blocOptions(w - 1) {
		corps = append(corps, " "+x)
	}
	corps = append(corps, "")
	if d.agencement == compacte {
		for _, x := range m.lignesApercu(nom, w-1) {
			corps = append(corps, tronquer(x+z, w))
		}
		return corps
	}
	return append(corps, m.boiteApercu(nom, w, "")...)
}

// barreOnglets : « Thèmes   Statusline », l'onglet actif en inverse.
func (m *Modele) barreOnglets(w int) string {
	var t []string
	for k, n := range []string{i18n.T("Thèmes"), "Statusline"} {
		if k == m.onglet {
			t = append(t, inverse+gras+" "+n+" "+z)
		} else {
			t = append(t, estompe+" "+n+" "+z)
		}
	}
	marque := ""
	if m.os.modifie() {
		marque = "  " + estompe + i18n.T("• modifié") + z
	}
	s := strings.Join(t, " ") + "  " + estompe + "Tab" + z + marque
	return centrer(tronquer(s, w-1), w)
}

// --- Touches ---------------------------------------------------------------------------------------

// sens : +1 pour une touche vers le bas, -1 vers le haut.
func sens(t string) int {
	switch t {
	case "down", "j", "shift+down", "J":
		return 1
	}
	return -1
}

// sensH : +1 pour une touche vers la droite, -1 vers la gauche.
func sensH(t string) int {
	if t == "right" || t == "l" {
		return 1
	}
	return -1
}

// saut : le déplacement des touches Page préc./suiv., Début, Fin.
func saut(t string, page int) int {
	switch t {
	case "pgup":
		return -max(page-1, 1)
	case "pgdown":
		return max(page-1, 1)
	case "home":
		return -1000
	}
	return 1000
}

func (m *Modele) apercuSuivant() tea.Cmd {
	m.os.apercu = (m.os.apercuVu() + 1) % len(nomsApercus)
	if m.os.apercu == 3 && !m.os.court() {
		m.os.apercu = 0
	}
	return nil
}

// ajusterStyle : la rangée choisie reste visible.
func (m *Modele) ajusterStyle() {
	o := &m.os
	vis := m.hauteurListeStyle()
	o.lignesVues = vis
	n := len(o.rangees())
	o.k = min(max(o.k, 0), n-1)
	// les indicateurs « de plus » prennent la première et la dernière ligne quand la liste défile
	if o.k < o.haut+1 && o.haut > 0 {
		o.haut = max(o.k-1, 0)
	}
	if o.k >= o.haut+vis-1 && o.haut+vis < n {
		o.haut = o.k - vis + 2
	}
	if o.haut+vis > n {
		o.haut = max(n-vis, 0)
	}
	if o.k == 1 { // la première rangée : son en-tête aussi
		o.haut = 0
	}
}
