package selecteur

import (
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/style"
)

// L'onglet « Statusline » : la vraie statusline en grand, au style en cours et au thème survolé dans
// l'onglet Thèmes. On y choisit un item sur l'aperçu même (←→ ↑↓, l'item choisi est surligné), Entrée ouvre
// son menu (ses options, coller au précédent, retirer), + en ajoute un, g ouvre les options générales.
// Rien n'est écrit avant s (puis d).

// Les lignes de la statusline (la ligne SSH seulement avec ssh.short=yes).
const (
	gLigne1 = iota
	gLigne2
	gCourt
)

// nomsGroupes : les noms des lignes (en français, traduits à l'affichage).
var nomsGroupes = [...]string{i18n.M("Ligne 1"), i18n.M("Ligne 2"), i18n.M("Ligne SSH")}

// clesGroupes : l'option de disposition de chaque ligne.
var clesGroupes = [...]string{"line1", "line2", "short"}

// Les aperçus (touche v), en français, traduits à l'affichage.
var nomsApercus = []string{i18n.M("complète"), i18n.M("tendue"), "API", "SSH"}

// Les menus ouverts sous l'aperçu.
const (
	sansMenu    = iota
	menuItem    // Entrée : les options de l'item, coller, retirer
	menuGeneral // g : les options générales
	menuAjout   // + : les items à ajouter à la ligne
)

// Les rangées d'action du menu d'un item (les autres sont ses options).
const (
	actionColler  = "coller"
	actionRetirer = "retirer"
)

// rangeeMenu : une rangée du menu : une option, ou une action.
type rangeeMenu struct {
	cle    style.Cle
	action string
}

// ongletStyle : l'état de l'onglet.
type ongletStyle struct {
	st, garde *style.Style // le style en cours, et celui du fichier
	ver       int          // incrémenté à chaque modification (caches des aperçus)
	g, pos    int          // l'item choisi : sa ligne, et sa place dans la ligne
	menu      int          // sansMenu, menuItem, menuGeneral ou menuAjout
	opt       int          // rangée choisie dans le menu
	saisie    bool         // on tape une valeur (libellé, glyphe)
	texte     string       // ce qui est tapé
	apercu    int          // index dans nomsApercus
	message   string       // une information ou une erreur, effacée à la touche suivante
	ignore    bool         // AGENTLINE_STYLE=default : le style gardé ne se verra pas
	rendus    map[string]apercuRendu
}

// apercuRendu : une statusline de l'aperçu, ses lignes et la place de chaque item.
type apercuRendu struct {
	lignes []string
	places []render.Place
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

func contient(p []place, n string) bool {
	for _, x := range p {
		if x.nom == n {
			return true
		}
	}
	return false
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

// lignes : les lignes qu'on peut parcourir.
func (o *ongletStyle) lignes() int { return 2 + b2i(o.court()) }

// courant : l'item choisi (vide si sa ligne n'en a aucun).
func (o *ongletStyle) courant() (place, bool) {
	p := o.places(o.g)
	if o.pos < 0 || o.pos >= len(p) {
		return place{}, false
	}
	return p[o.pos], true
}

// borner : la ligne et la place choisies restent valides (après un changement de disposition).
func (o *ongletStyle) borner() {
	o.g = min(max(o.g, 0), o.lignes()-1)
	o.pos = min(max(o.pos, 0), max(len(o.places(o.g))-1, 0))
}

// viser : l'item dans cette ligne.
func (o *ongletStyle) viser(g int, item string) {
	o.g = g
	for k, x := range o.places(g) {
		if x.nom == item {
			o.pos = k
		}
	}
	o.borner()
}

// choisir : ←→, l'item d'à côté ; en bout de ligne, la ligne voisine.
func (o *ongletStyle) choisir(d int) {
	n := len(o.places(o.g))
	switch j := o.pos + d; {
	case j >= 0 && j < n:
		o.pos = j
	case d > 0 && o.g < o.lignes()-1:
		o.g, o.pos = o.g+1, 0
	case d < 0 && o.g > 0:
		o.g--
		o.pos = len(o.places(o.g)) - 1
	}
	o.borner()
}

// changerLigne : ↑↓, la ligne du dessus ou du dessous, à la même place si elle existe.
func (o *ongletStyle) changerLigne(d int) {
	o.g = (o.g + d + o.lignes()) % o.lignes()
	o.borner()
}

func (o *ongletStyle) changer() { o.ver++ }

// deplacer : Maj+←→, l'item échange sa place avec son voisin (les segments gardent leur forme).
func (o *ongletStyle) deplacer(d int) {
	x, ok := o.courant()
	if !ok {
		return
	}
	p := o.places(o.g)
	j := o.pos + d
	if j < 0 || j >= len(p) {
		o.message = i18n.T("déjà en bout de %s", strings.ToLower(i18n.T(nomsGroupes[o.g])))
		return
	}
	p[o.pos].nom, p[j].nom = p[j].nom, p[o.pos].nom
	o.poser(o.g, p)
	o.changer()
	o.viser(o.g, x.nom)
}

// changerDeLigne : Maj+↑↓, l'item passe de la ligne 1 au début de la ligne 2, ou de la ligne 2 à la fin
// de la ligne 1, seul dans son segment.
func (o *ongletStyle) changerDeLigne(d int) {
	x, ok := o.courant()
	if !ok {
		return
	}
	if o.g == gCourt || o.g == gLigne1 && d < 0 || o.g == gLigne2 && d > 0 {
		o.message = i18n.T("Maj+↑↓ passe un item de la ligne 1 à la ligne 2, et inversement")
		return
	}
	autre := 1 - o.g
	p, q := o.places(o.g), o.places(autre)
	if d > 0 {
		q = append([]place{{nom: x.nom}}, q...)
	} else {
		q = append(q, place{nom: x.nom})
	}
	o.poser(o.g, append(p[:o.pos:o.pos], p[o.pos+1:]...))
	o.poser(autre, q)
	o.changer()
	o.viser(autre, x.nom)
}

// retirer : x, l'item quitte sa ligne (+ l'y remet).
func (o *ongletStyle) retirer() {
	if _, ok := o.courant(); !ok {
		return
	}
	p := o.places(o.g)
	o.poser(o.g, append(p[:o.pos:o.pos], p[o.pos+1:]...))
	o.changer()
	o.borner()
}

// coller : l'item rejoint le segment de l'item précédent, ou s'en détache.
func (o *ongletStyle) coller() {
	x, ok := o.courant()
	if !ok {
		return
	}
	if o.pos == 0 {
		o.message = i18n.T("premier de la ligne : rien à quoi le coller")
		return
	}
	p := o.places(o.g)
	p[o.pos].colle = !p[o.pos].colle
	o.poser(o.g, p)
	o.changer()
	o.viser(o.g, x.nom)
}

// candidats : les items qu'on peut ajouter à la ligne choisie (absents des deux lignes longues, ou de la
// ligne courte).
func (o *ongletStyle) candidats() []string {
	var l []string
	for _, n := range style.Items {
		if o.g == gCourt && !contient(o.places(gCourt), n) ||
			o.g != gCourt && !contient(o.places(gLigne1), n) && !contient(o.places(gLigne2), n) {
			l = append(l, n)
		}
	}
	return l
}

// ajouter : l'item après celui choisi (au début d'une ligne vide), seul dans son segment ; masqué
// partout, il est réaffiché.
func (o *ongletStyle) ajouter(n string) {
	p := o.places(o.g)
	k := min(o.pos+1, len(p))
	p = append(p[:k:k], append([]place{{nom: n}}, p[k:]...)...)
	o.poser(o.g, p)
	o.st.Masquer(n, false)
	o.changer()
	o.viser(o.g, n)
}

// rangeesMenu : les rangées du menu ouvert.
func (o *ongletStyle) rangeesMenu() []rangeeMenu {
	var l []rangeeMenu
	switch o.menu {
	case menuGeneral:
		for _, c := range style.DesItems("") {
			l = append(l, rangeeMenu{cle: c})
		}
	case menuItem:
		x, _ := o.courant()
		for _, c := range style.DesItems(x.nom) {
			l = append(l, rangeeMenu{cle: c})
		}
		if o.pos > 0 {
			l = append(l, rangeeMenu{action: actionColler})
		}
		l = append(l, rangeeMenu{action: actionRetirer})
	case menuAjout:
		for _, n := range o.candidats() {
			l = append(l, rangeeMenu{action: n})
		}
	}
	return l
}

// rangeeMenu : la rangée choisie du menu.
func (o *ongletStyle) rangeeMenu() rangeeMenu {
	l := o.rangeesMenu()
	if len(l) == 0 {
		return rangeeMenu{}
	}
	return l[min(max(o.opt, 0), len(l)-1)]
}

// cle : l'option de la rangée choisie (vide sur une action).
func (o *ongletStyle) cle() style.Cle { return o.rangeeMenu().cle }

// ouvrir : un menu, sur sa première rangée.
func (o *ongletStyle) ouvrir(menu int) {
	switch {
	case menu == menuItem:
		if _, ok := o.courant(); !ok {
			o.message = i18n.T("cette ligne est vide : + y ajoute un item")
			return
		}
	case menu == menuAjout && len(o.candidats()) == 0:
		o.message = i18n.T("tous les items sont déjà dans les lignes")
		return
	}
	o.menu, o.opt = menu, 0
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
	case style.Libelle, style.Branches:
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

// regler : ←→ sur la rangée choisie : la valeur de l'option, ou coller (oui, non).
func (o *ongletStyle) regler(d int) {
	r := o.rangeeMenu()
	switch {
	case r.action == actionColler:
		o.coller()
		return
	case r.action != "":
		return
	}
	c := r.cle
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

// entrer : Entrée dans un menu : taper la valeur (libellé, glyphe, barre), sinon la suivante ; coller,
// retirer ; ajouter l'item choisi.
func (o *ongletStyle) entrer() {
	r := o.rangeeMenu()
	switch {
	case o.menu == menuAjout:
		o.ajouter(r.action)
		o.menu = sansMenu
	case r.action == actionRetirer:
		o.retirer()
		o.menu = sansMenu
	case r.action == actionColler:
		o.coller()
	case saisissable(r.cle):
		o.saisie, o.texte = true, o.st.Valeur(r.cle.Nom)
	case r.cle.Nom != "":
		o.regler(1)
	}
}

// saisissable : l'option se règle aussi en tapant sa valeur.
func saisissable(c style.Cle) bool {
	return c.Genre == style.Libelle || c.Genre == style.Glyphe || c.Genre == style.Barre || c.Genre == style.Branches
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

// remettre : r, les options de l'item à leur défaut, et l'item réaffiché s'il était masqué (dans un menu :
// l'option choisie seulement).
func (o *ongletStyle) remettre() {
	if o.menu != sansMenu {
		if c := o.cle(); c.Nom != "" {
			o.st.Retirer(c.Nom)
		}
	} else if x, ok := o.courant(); ok {
		for _, c := range style.DesItems(x.nom) {
			o.st.Retirer(c.Nom)
		}
		o.st.Masquer(x.nom, false)
	}
	o.changer()
}

func (o *ongletStyle) modifie() bool { return !o.st.Egal(o.garde) }

// --- Dans le modèle --------------------------------------------------------------------------------

func (m *Modele) initStyle() {
	m.os.garde = style.LireFichier(m.sources.Config)
	m.os.st = m.os.garde.Copie()
	m.os.ignore = style.Force(m.getenv)
	m.os.rendus = map[string]apercuRendu{}
}

// OuvrirStyle : le sélecteur s'ouvre sur l'onglet Statusline (agentline style).
func (m *Modele) OuvrirStyle() {
	m.onglet = 1
	m.ajuster()
}

func (m *Modele) basculerOnglet() tea.Cmd {
	d := 1
	if m.derniere == "shift+tab" {
		d = 2
	}
	m.onglet = (m.onglet + d) % 3
	if m.onglet == 2 && !m.ed.actif {
		m.editer(m.noms[m.i])
	}
	m.reagir("surpris")
	return nil
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

// rendreStyle : la statusline de la session k au thème donné, avec le style en cours, sur w colonnes, et la
// place de ses items.
func (m *Modele) rendreStyle(nom string, w, k int, ssh bool) apercuRendu {
	cle := nom + "@" + strconv.Itoa(w) + "@" + strconv.Itoa(k) + "@" + strconv.FormatBool(ssh) + "@" + strconv.Itoa(m.os.ver)
	if r, ok := m.os.rendus[cle]; ok {
		return r
	}
	if len(m.os.rendus) > 400 {
		m.os.rendus = map[string]apercuRendu{}
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
	sortie, places := render.RenderPlaces(s, e)
	r := apercuRendu{strings.Split(strings.TrimRight(sortie, "\n"), "\n"), places}
	m.os.rendus[cle] = r
	return r
}

// ligne : la ligne k (vide si la statusline en a moins), l'item surligné (marque) s'il y est dessiné.
func (r apercuRendu) ligne(k int, item, marque string) string {
	if k >= len(r.lignes) {
		return ""
	}
	for _, p := range r.places {
		if item != "" && p.Item == item && p.Ligne == k {
			return surligner(r.lignes[k], p.Debut, p.Fin, marque)
		}
	}
	return r.lignes[k]
}

// dessine : l'item est-il dessiné dans cette statusline ?
func (r apercuRendu) dessine(item string) bool {
	for _, p := range r.places {
		if p.Item == item {
			return true
		}
	}
	return false
}

// surligner : les caractères visibles de debut à fin (fin exclue) surlignés par la séquence marque (finie par
// finSurligne) ; la marque est remise après chaque séquence de la portion (un z l'annule).
func surligner(s string, debut, fin int, marque string) string {
	var b strings.Builder
	vus := 0
	dedans := func() bool { return vus >= debut && vus < fin }
	for len(s) > 0 {
		if n := longueurSequence(s); n > 0 {
			b.WriteString(s[:n])
			s = s[n:]
			if dedans() && vus > debut {
				b.WriteString(marque)
			}
			continue
		}
		if vus == debut {
			b.WriteString(marque)
		}
		if vus == fin {
			b.WriteString(finSurligne)
		}
		_, t := utf8.DecodeRuneInString(s)
		b.WriteString(s[:t])
		s = s[t:]
		vus++
	}
	if vus == fin {
		b.WriteString(finSurligne)
	}
	return b.String()
}

// longueurSequence : la longueur de la séquence d'échappement en tête de s (CSI, ou OSC fini par BEL ou
// ESC \), 0 s'il n'y en a pas.
func longueurSequence(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
	}
	return len(s)
}

// lignesApercu : les lignes de l'aperçu sur w colonnes, l'item choisi surligné : la statusline et, avec
// la ligne courte, sa ligne SSH ; en aperçu SSH, la ligne courte des trois sessions.
func (m *Modele) lignesApercu(nom string, w int) []string {
	o := &m.os
	x, _ := o.courant()
	marque := ansi.Bg(m.palette(nom).Sep) + souligne // un fond discret : les couleurs de l'item restent
	sur := func(g int) string {                      // l'item à surligner sur la ligne g
		if o.g == g {
			return x.nom
		}
		return ""
	}
	ssh := estompe + "SSH " + z
	a := o.apercuVu()
	var l []string
	if a == 3 {
		for k := 0; k < 3; k++ {
			l = append(l, ssh+m.rendreStyle(nom, w-4, k, true).ligne(0, sur(gCourt), marque))
		}
		return l
	}
	long := m.rendreStyle(nom, w, a, false)
	for k := 0; k < 2; k++ {
		item := ""
		if o.g != gCourt {
			item = x.nom
		}
		l = append(l, long.ligne(k, item, marque))
	}
	if o.court() {
		l = append(l, ssh+m.rendreStyle(nom, w-4, a, true).ligne(0, sur(gCourt), marque))
	}
	return l
}

// dessineIci : l'item choisi est-il dessiné dans l'aperçu en cours ?
func (m *Modele) dessineIci(nom string, w int) bool {
	o := &m.os
	x, ok := o.courant()
	if !ok {
		return false
	}
	a := o.apercuVu()
	if o.g == gCourt {
		if a == 3 {
			a = 0
		}
		return m.rendreStyle(nom, w-4, a, true).dessine(x.nom)
	}
	if a == 3 {
		return false
	}
	return m.rendreStyle(nom, w, a, false).dessine(x.nom)
}

// boiteApercu : l'aperçu dans un cadre de largeur w, Pip (ses pattes) posé sur le cadre.
func (m *Modele) boiteApercu(nom string, w int, pattes string) []string {
	iw := w - 4
	titre := "╭─ " + i18n.T("Aperçu · %s (v) ", i18n.T(nomsApercus[m.os.apercuVu()]))
	if largeur(titre)+2 > w {
		titre = "╭─"
	}
	cadre := []rune(titre + strings.Repeat("─", max(w-largeur(titre)-1, 0)) + "╮")
	haut := string(cadre)
	if x := w - largeurMascotte - 3; pattes != "" && x > largeur(titre) {
		haut = string(cadre[:x]) + strings.ReplaceAll(pattes, " ", "─") + z + string(cadre[x+largeurMascotte:])
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

// valeurAffichee : la valeur d'une option, telle que le menu la montre.
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

// ligneChoix : sous l'aperçu, l'item choisi, sa ligne et ce qu'il faut savoir de lui.
func (m *Modele) ligneChoix(nom string, w, wApercu int) string {
	o := &m.os
	x, ok := o.courant()
	ligne := i18n.T(nomsGroupes[o.g])
	if !ok {
		return tronquer(estompe+i18n.T("%s : vide, + y ajoute un item", ligne)+z, w)
	}
	var notes []string
	switch {
	case o.st.Masques()[x.nom]:
		notes = append(notes, i18n.T("masqué partout (r le réaffiche)"))
	case !m.dessineIci(nom, wApercu):
		notes = append(notes, i18n.T("vide dans cet aperçu (v : un autre)"))
	}
	if x.colle {
		notes = append(notes, i18n.T("collé au précédent"))
	}
	t := "▲ " + gras + x.nom + z + estompe + " · " + ligne
	for _, n := range notes {
		t += " · " + n
	}
	return tronquer(t+z, w)
}

// panneauMenu : le menu ouvert, une rangée par option ou action (◂ valeur ▸ sur celle choisie), puis l'aide
// de la rangée choisie, repliée sur w colonnes.
func (m *Modele) panneauMenu(w int) []string {
	o := &m.os
	x, _ := o.courant()
	titre := i18n.T("Options · %s", x.nom)
	switch o.menu {
	case menuGeneral:
		titre = i18n.T("Options · %s", i18n.T("générales"))
	case menuAjout:
		titre = i18n.T("Ajouter à la %s", strings.ToLower(i18n.T(nomsGroupes[o.g])))
	}
	l := []string{gras + tronquer(titre, w) + z}
	rangees := o.rangeesMenu()
	gauche := 0
	for _, r := range rangees {
		gauche = max(gauche, largeur(m.nomRangee(r)))
	}
	gauche = min(gauche, w/2)
	for k, r := range rangees {
		nom := tronquer(m.nomRangee(r), gauche)
		t := "  " + nom
		if v, ok := m.valeurRangee(r); ok {
			t += espaces(gauche-largeur(nom)) + "  "
			switch {
			case k == o.opt && o.saisie:
				t += "› " + gras + o.texte + z + "▏"
			case k == o.opt:
				t += "◂ " + gras + v + z + " ▸"
			default:
				t += estompe + v + z
			}
		}
		if r.cle.Nom != "" && o.st.Regle(r.cle.Nom) && !(k == o.opt && o.saisie) {
			t += estompe + " •" + z
		}
		if k == o.opt { // le curseur devant le nom ; l'aide en dessous sur la même colonne
			t = "› " + t[2:]
		}
		l = append(l, tronquer(t, w))
	}
	aide := ""
	switch r := o.rangeeMenu(); {
	case r.cle.Nom != "":
		aide = r.cle.Explication()
		if !r.cle.SSH && o.court() {
			aide += i18n.T(" (pas en SSH)")
		}
	case r.action == actionColler:
		aide = i18n.T("même segment que l'item précédent, sans séparateur entre eux")
	case r.action == actionRetirer:
		aide = i18n.T("l'item quitte la ligne ; + l'y remet")
	}
	if aide != "" {
		if m.d.agencement == compacte { // une ligne : l'aperçu passe avant
			return append(l, "  "+estompe+tronquer(aide, w-2)+z)
		}
		for _, x := range replier(strings.Fields(aide), " ", w-2) {
			l = append(l, "  "+estompe+x+z)
		}
	}
	return l
}

// nomRangee : le nom d'une rangée du menu.
func (m *Modele) nomRangee(r rangeeMenu) string {
	switch {
	case r.cle.Nom != "":
		return r.cle.Nom
	case r.action == actionColler:
		return i18n.T("coller au précédent")
	case r.action == actionRetirer:
		return i18n.T("retirer de la ligne")
	}
	return r.action // un item à ajouter
}

// valeurRangee : la valeur montrée à droite d'une rangée (aucune pour retirer ou un item à ajouter).
func (m *Modele) valeurRangee(r rangeeMenu) (string, bool) {
	switch {
	case r.cle.Nom != "":
		return m.valeurAffichee(r.cle), true
	case r.action == actionColler:
		if x, _ := m.os.courant(); x.colle {
			return style.Oui, true
		}
		return style.Non, true
	}
	return "", false
}

// indice : sous l'aperçu, quand aucun menu n'est ouvert : ce que l'item choisi permet.
func (m *Modele) indice(w int) []string {
	x, ok := m.os.courant()
	if !ok {
		return nil
	}
	var n []string
	for _, c := range style.DesItems(x.nom) {
		n = append(n, c.Nom)
	}
	t := i18n.T("Entrée : coller, retirer")
	if len(n) > 0 {
		t = i18n.T("Entrée : %s, coller, retirer", strings.Join(n, ", "))
	}
	var l []string
	for _, x := range replier(strings.Fields(t), " ", w) {
		l = append(l, estompe+x+z)
	}
	return l
}

// corpsStyle : le corps de l'onglet : l'aperçu (dans son cadre, Pip posé dessus, sauf en compact),
// l'item choisi, puis le menu ouvert ou ce que l'item permet.
func (m *Modele) corpsStyle(nom string) []string {
	d := m.d
	w := max(m.largeur, largeurMin) - 1
	var corps []string
	if d.agencement == compacte {
		for _, x := range m.lignesApercu(nom, w-1) {
			corps = append(corps, " "+tronquer(x+z, w-1))
		}
		corps = append(corps, " "+m.ligneChoix(nom, w-1, w-1))
	} else {
		bw := w
		if l := m.largeurNaturelle(nom); l+4 < bw {
			bw = max(l+4, min(w, 72))
		}
		marge := espaces((w - bw) / 2)
		pattes := ""
		if d.mascotte && d.corps >= hauteurMascotte+m.lignesApercuStyle()+6 {
			var cl [hauteurMascotte]string
			cl, pattes, _ = m.Mascotte(nom)
			x := marge + espaces(bw-largeurMascotte-3)
			for _, l := range cl {
				corps = append(corps, x+l)
			}
		}
		for _, l := range m.boiteApercu(nom, bw, pattes) {
			corps = append(corps, marge+l)
		}
		corps = append(corps, marge+"  "+m.ligneChoix(nom, bw-2, bw-4))
		// centré en hauteur, sur la place du plus long menu : rien ne saute quand un menu s'ouvre
		haut := max((d.corps-len(corps)-1-hauteurMenuMax())/2, 0)
		corps = append(make([]string, haut), corps...)
		corps = append(corps, "")
		for _, x := range m.blocMenu(bw - 2) {
			corps = append(corps, marge+"  "+x)
		}
		return corps
	}
	corps = append(corps, "")
	for _, x := range m.blocMenu(w - 1) {
		corps = append(corps, " "+x)
	}
	return corps
}

// hauteurMenuMax : la hauteur du plus long menu : son titre, ses rangées (options, coller, retirer ; ou les
// items à ajouter) et trois lignes d'aide.
func hauteurMenuMax() int {
	n := max(len(style.DesItems("")), len(style.Items))
	for _, i := range style.Items {
		n = max(n, len(style.DesItems(i))+2)
	}
	return 1 + n + 3
}

// blocMenu : le menu ouvert, sinon ce que l'item choisi permet.
func (m *Modele) blocMenu(w int) []string {
	if m.os.menu != sansMenu {
		return m.panneauMenu(w)
	}
	return m.indice(w)
}

// largeurNaturelle : la plus longue ligne de l'aperçu sans limite de largeur (le cadre s'y ajuste).
func (m *Modele) largeurNaturelle(nom string) int {
	n := 0
	for _, l := range m.lignesApercu(nom, 400) {
		n = max(n, largeur(l))
	}
	return n
}

// lignesApercuStyle : la hauteur du cadre de l'aperçu (la ligne SSH en plus avec la ligne courte).
func (m *Modele) lignesApercuStyle() int { return 4 + b2i(m.os.court()) }

// barreOnglets : « Thèmes   Statusline   Créer », l'onglet actif en inverse.
func (m *Modele) barreOnglets(w int) string {
	var t []string
	for k, n := range []string{i18n.T("Thèmes"), "Statusline", i18n.T("Créer")} {
		if k == m.onglet {
			t = append(t, inverse+gras+" "+n+" "+z)
		} else {
			t = append(t, estompe+" "+n+" "+z)
		}
	}
	marque := ""
	if m.os.modifie() && m.onglet == 1 || m.ed.modifie() && m.onglet == 2 {
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
	switch t {
	case "right", "l", "shift+right", "L":
		return 1
	}
	return -1
}

func (m *Modele) apercuSuivant() tea.Cmd {
	m.os.apercu = (m.os.apercuVu() + 1) % len(nomsApercus)
	if m.os.apercu == 3 && !m.os.court() {
		m.os.apercu = 0
	}
	return nil
}

// ajusterStyle : le choix reste valide, la rangée du menu aussi.
func (m *Modele) ajusterStyle() {
	o := &m.os
	o.borner()
	if n := len(o.rangeesMenu()); o.menu != sansMenu && n == 0 {
		o.menu = sansMenu
	} else {
		o.opt = min(max(o.opt, 0), max(n-1, 0))
	}
}
