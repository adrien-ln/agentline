package selecteur

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Le dessin : View ne fait que lire l'état (la disposition est calculée par ajuster) ; seuls les caches
// (palettes, logos, rendus de la statusline) s'y remplissent.

func espaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

func (m *Modele) palette(nom string) theme.Palette {
	if p, ok := m.palettes[nom]; ok {
		return p
	}
	p, ok := m.sources.Charger(nom)
	if !ok {
		p = theme.Secours()
	}
	p = p.PourFond(m.clair())
	m.palettes[nom] = p
	return p
}

// --- Textes et couleurs du fond clair ---

// clair : le terminal a un fond clair (réglage, variable, ou détecté à l'ouverture).
func (m *Modele) clair() bool { return m.apercu.Couleurs.Fond == ansi.Clair }

// scenesClair : les couleurs de l'interface de Claude Code dans les scènes, pour un fond clair (le texte
// en noir, les diffs sur fond pâle comme dans le thème clair de GitHub, les autres assombries).
var scenesClair = strings.NewReplacer(
	cTexte, "\x1b[30m",
	cOrange, ansi.Fg(ansi.Ajuster("d77757", ansi.FondClair, 3, 0)),
	cVert, ansi.Fg(ansi.Ajuster("4eba65", ansi.FondClair, 3, 0)),
	cRouge, ansi.Fg(ansi.Ajuster("ff6b80", ansi.FondClair, 3, 0)),
	cEnCours, ansi.Fg(ansi.Ajuster("969696", ansi.FondClair, 3, 0)),
	cAjout, ansi.Fg("1f2328")+ansi.Bg("dafbe1"),
	cRetrait, ansi.Fg("1f2328")+ansi.Bg("ffebe9"),
)

// surFond : les lignes peintes sur le fond clair de référence (mini-fenêtre, aperçu), si le fond est clair :
// l'aperçu montre la statusline telle qu'elle sera, même dans un terminal sombre.
func (m *Modele) surFond(l []string) []string {
	if !m.clair() || m.apercu.Couleurs.Niveau == ansi.SansCouleur {
		return l
	}
	f := ansi.Bg(ansi.FondClair)
	r := make([]string, len(l))
	for k, x := range l {
		r[k] = f + strings.ReplaceAll(x, z, z+f) + z
	}
	return r
}

// degradeLigne : la ligne colorée colonne par colonne entre les couleurs données (espaces transparents).
func degradeLigne(ligne string, stops []string) string {
	r := []rune(ligne)
	n, k := max(len(r), 2), len(stops)
	var b strings.Builder
	for i, c := range r {
		if c == ' ' {
			b.WriteByte(' ')
			continue
		}
		t := i * (k - 1) * 1000 / (n - 1)
		j, f := t/1000, t%1000
		if j > k-2 {
			j, f = k-2, 1000
		}
		a, _ := strconv.ParseUint(stops[j], 16, 32)
		c2, _ := strconv.ParseUint(stops[j+1], 16, 32)
		var rgb [3]int
		for x, d := range []uint{16, 8, 0} {
			u, v := int(a>>d&255), int(c2>>d&255)
			rgb[x] = u + (v-u)*f/1000
		}
		b.WriteString(ansi.FgRGB(rgb[0], rgb[1], rgb[2]))
		b.WriteRune(c)
	}
	return b.String() + z
}

// logo : « agentline » en minuscules pixel, comme la bannière du README (docs/banniere.svg) ; la 1re ligne
// porte les ascendantes du t et du l et le point du i, la dernière la barre du g, sous la ligne de base.
var logo = []string{
	"                ▄  █  ▀        ",
	"▄▀█ █▀█ █▀▀ █▀▄ █▀ █  ▄ █▀▄ █▀▀",
	"▀▄█ ▀▀█ ██▄ █ █ ▀▄ ▀▄ █ █ █ ██▄",
	"    ▀▀▀                        ",
}

func (m *Modele) entete(nom string) []string {
	if l, ok := m.logos[nom]; ok {
		return strings.Split(l, "\n")
	}
	p := m.palette(nom)
	var l []string
	for _, x := range logo {
		l = append(l, degradeLigne(x, p.Intensite[:]))
	}
	m.logos[nom] = strings.Join(l, "\n")
	return l
}

// pastilles : les 9 couleurs du thème en pastilles ██, autant qu'il en tient dans la largeur.
func (m *Modele) pastilles(nom string, largeur int) string {
	p := m.palette(nom)
	var b strings.Builder
	for k, c := range append(p.Intensite[:], p.D0, p.D50, p.D75, p.D100) {
		if (k+1)*3 > largeur {
			break
		}
		b.WriteString(ansi.Fg(c) + "██" + z + " ")
	}
	return b.String()
}

// statusline : les deux lignes de la vraie statusline, pour une session fictive (mises en cache).
func (m *Modele) statusline(nom string, largeur, ctx int) []string {
	cle := nom + "@" + strconv.Itoa(largeur) + "@" + strconv.Itoa(ctx) + "@" + strconv.Itoa(m.os.ver)
	if r, ok := m.rendus[cle]; ok {
		return strings.Split(r, "\n")
	}
	if len(m.rendus) > 2000 { // le style a changé souvent : les anciens rendus ne servent plus
		m.rendus = map[string]string{}
	}
	e := m.apercu
	e.Columns, e.SSH, e.SSHRiche, e.Style = largeur+4, false, false, m.os.st
	e.GitFictif = &render.GitFictif{Branche: "main", Neuf: 1}
	e.Getenv = func(k string) string {
		if k == "AGENTLINE_THEME" {
			return nom
		}
		return ""
	}
	s := etat.Exemple(e.Now, "/apercu/"+i18n.T("mon-projet"), ctx)
	s.PR = etat.PR{Num: "12", Revue: "pending"}
	r := strings.TrimRight(render.RenderSession(s, e), "\n")
	m.rendus[cle] = r
	return strings.Split(r, "\n")
}

// fenetre : la mini-fenêtre Claude Code de largeur w et de hauteur h (au moins 9), à l'instant m.t de la
// session fictive ; pattes : celles de Pip, posées sur le cadre du haut (vide : aucune).
func (m *Modele) fenetre(nom string, w, h int, pattes string) []string {
	iw := w - 4
	trace := func(n int) string { return strings.Repeat("─", max(min(n, w), 0)) }
	sc, ts := sceneA(m.scenes, m.t)
	etat := sc.etatA(ts)
	titre := "╭─ Claude Code · " + i18n.T("mon-projet") + " "
	if largeur(titre)+2 > w {
		titre = "╭─"
	}
	cadre := []rune(titre + trace(w-largeur(titre)-1) + "╮")
	haut := string(cadre)
	if x := w - largeurMascotte - 3; pattes != "" && x > largeur(titre) {
		haut = string(cadre[:x]) + strings.ReplaceAll(pattes, " ", "─") + z + string(cadre[x+largeurMascotte:])
	}
	l := []string{haut}
	for _, c := range etat.lignesConversation(ts, max(h-7, 1), iw, historique(m.scenes, m.t)) {
		if largeur(c) > iw { // la ligne est coupée, couleurs préservées
			c = couper(c, iw)
		}
		if m.clair() {
			c = scenesClair.Replace(c)
		}
		l = append(l, "│ "+c+z+espaces(iw-largeur(c))+" │")
	}
	// Zone de saisie : la fin du texte tapé, puis le curseur qui clignote
	saisie := []rune(etat.saisie)
	if place := iw - 7; len(saisie) > place {
		saisie = saisie[len(saisie)-max(place, 0):]
	}
	curseur := " "
	if (m.t/530)%2 == 0 {
		curseur = "▌"
	}
	ligneSaisie := "> " + string(saisie) + curseur
	l = append(l, "│ ╭"+trace(iw-2)+"╮ │", "│ │ "+ligneSaisie+espaces(iw-4-largeur(ligneSaisie))+" │ │", "│ ╰"+trace(iw-2)+"╯ │")
	sl := m.statusline(nom, iw, min(42+3*etat.outils, 100))
	for k := 0; k < 2; k++ {
		c := ""
		if k < len(sl) {
			c = sl[k]
		}
		if largeur(c) > iw {
			c = couper(c, iw)
		}
		l = append(l, "│ "+c+z+espaces(iw-largeur(c))+" │")
	}
	return m.surFond(append(l, "╰"+trace(w-2)+"╯"))
}

// --- Mesure et pliage du texte ----------------------------------------------------------------------

// cellules : la largeur des caractères dans le terminal (deux colonnes pour le CJK et la plupart des
// emoji, aucune pour les accents combinants). Les caractères ambigus (cadres, blocs de Pip) comptent
// pour une colonne, quelle que soit la langue du système. Le rendu de la statusline (internal/render)
// garde sa propre mesure, celle de l'ancienne version bash.
var cellules = &runewidth.Condition{EastAsianWidth: false, StrictEmojiNeutral: true}

// largeur : les colonnes qu'occupe le texte affiché (séquences de couleur sautées).
func largeur(s string) int {
	n := 0
	for len(s) > 0 {
		if s[0] == 0x1b {
			if j := strings.IndexByte(s, 'm'); j >= 0 {
				s = s[j+1:]
				continue
			}
		}
		if c := s[0]; c >= 0x20 && c < 0x7f { // ASCII imprimable : une colonne, sans passer par runewidth
			n++
			s = s[1:]
			continue
		}
		r, t := utf8.DecodeRuneInString(s)
		s = s[t:]
		n += cellules.RuneWidth(r)
	}
	return n
}

// tronquer : le texte ramené à w colonnes, « … » à la fin s'il a fallu couper.
func tronquer(s string, w int) string {
	if largeur(s) <= w {
		return s
	}
	return couper(s, max(w-1, 0)) + "…"
}

// replier : les morceaux répartis sur des lignes de w colonnes au plus, séparés par sep ; un morceau trop
// long passe à la ligne à ses espaces, un mot trop long est coupé en tranches : rien n'est perdu.
func replier(morceaux []string, sep string, w int) []string {
	w = max(w, 1)
	var lignes []string
	cour := ""
	ajouter := func(x, s string) {
		switch {
		case cour == "":
			cour = x
		case largeur(cour)+largeur(s)+largeur(x) <= w:
			cour += s + x
		default:
			lignes = append(lignes, cour)
			cour = x
		}
	}
	for _, mc := range morceaux {
		if largeur(mc) <= w {
			ajouter(mc, sep)
			continue
		}
		for k, mot := range strings.Split(mc, " ") {
			s := " "
			if k == 0 {
				s = sep
			}
			for largeur(mot) > w {
				if cour != "" {
					lignes, cour = append(lignes, cour), ""
				}
				n := len(couper(mot, w)) - len(z) // ce qui tient dans w colonnes
				if n <= 0 {
					_, n = utf8.DecodeRuneInString(mot) // un caractère plus large que la ligne
				}
				lignes, mot = append(lignes, mot[:n]+z), mot[n:]
			}
			ajouter(mot, s)
		}
	}
	if cour != "" {
		lignes = append(lignes, cour)
	}
	return lignes
}

func remplir(s string, w int) string { return s + espaces(w-largeur(s)) }

// --- L'écran ---------------------------------------------------------------------------------------

func (m *Modele) ligneListe(k int, sel bool, lw int) string {
	t := tronquer(m.titres[k], lw-4)
	pad := espaces(lw - 4 - largeur(t))
	if sel {
		return inverse + " › " + t + pad + " " + z
	}
	if m.noms[k] == m.choix && m.themeEnAttente() { // choisi, pas encore enregistré
		return "  " + gras + "◆" + z + t + pad + " "
	}
	if m.noms[k] == m.avant {
		return "  " + estompe + "●" + z + t + pad + " "
	}
	return "   " + t + pad + " "
}

// indicateur aligne « ↑ n de plus » et « ↓ n de plus » sur les noms des thèmes (après la marge de « › »).
func indicateur(s string, lw int) string {
	if s == "" {
		return espaces(lw)
	}
	return "   " + estompe + s + z + espaces(lw-3-largeur(s))
}

func centrer(s string, w int) string { return espaces((w-largeur(s))/2) + s }

// lignesEntete : le titre en grandes lettres (s'il a sa place), centré d'un bloc, puis la ligne du dépôt :
// le thème en vigueur et pourquoi, la portée de s (onglet Thèmes), le thème survolé, le nombre de thèmes, le chemin, dans cet ordre de
// priorité (ce qui ne tient pas est retiré en partant de la fin), puis une ligne vide.
func (m *Modele) lignesEntete(nom string, w int) []string {
	var l []string
	if m.d.logo {
		titre, lt := m.entete(nom), 0
		for _, x := range titre {
			lt = max(lt, largeur(x))
		}
		for _, x := range titre {
			l = append(l, espaces((w-lt)/2)+x)
		}
	}
	depot, chemin := m.projet, ""
	if i := strings.LastIndex(m.projet, "/"); i >= 0 {
		chemin, depot = m.projet[:i+1], m.projet[i+1:]
	}
	p := m.palette(nom)
	pastille := ansi.Bg(p.Modele) + ansi.Fg(visageSombre) + gras + " " + depot + " " + z
	a := theme.Active(m.chaine)
	infos := []string{i18n.T("en vigueur : %s (%s)", m.titreDe(a.Nom), pourquoi(a))}
	if m.onglet == 0 {
		infos = append(infos, i18n.T("portée : %s", m.libellePortee()))
	}
	if nom != a.Nom {
		infos = append(infos, i18n.T("survolé : %s", m.titres[m.i]))
	}
	infos = append(infos, i18n.N(len(m.noms), "%d thème", "%d thèmes", len(m.noms)), strings.TrimSuffix(chemin, "/"))
	ligne := pastille
	for len(infos) > 0 {
		x := pastille + "  " + estompe + strings.Join(infos, "  ·  ") + z
		if largeur(x) <= w-1 {
			ligne = x
			break
		}
		infos = infos[:len(infos)-1]
	}
	if len(infos) == 0 { // même le thème en vigueur ne tient pas en entier
		ligne = tronquer(pastille+"  "+estompe+m.titreDe(a.Nom)+" ("+pourquoi(a)+")"+z, w-1)
	}
	return append(l, centrer(ligne, w), m.barreOnglets(w))
}

// colonneListe : le filtre, « ↑ de plus », les thèmes (ou pourquoi il n'y en a aucun), « ↓ de plus » et,
// sauf en compact, les pastilles du thème.
func (m *Modele) colonneListe(nom string) []string {
	d := m.d
	nv, vis := len(m.vue), d.liste
	barre := " ◂ " + i18n.T(m.filtres[m.f]) + " ▸ " + strconv.Itoa(nv)
	if m.recherche != "" {
		barre += " · /" + m.recherche
	}
	barre = tronquer(barre, d.lw)
	col := []string{gras + barre + z + espaces(d.lw-largeur(barre))}
	haut, bas := "", ""
	if m.haut > 0 {
		haut = i18n.T("↑ %d de plus", m.haut)
	}
	if m.haut+vis < nv {
		bas = i18n.T("↓ %d de plus", nv-m.haut-vis)
	}
	col = append(col, indicateur(haut, d.lw))
	var vide []string // aucun thème : ce qui permet d'en retrouver, replié dans la colonne
	if nv == 0 {
		pistes := []string{i18n.T("aucun thème")}
		if m.recherche != "" {
			pistes = append(pistes, i18n.T("Échap : effacer /%s", m.recherche))
		}
		if m.f != 0 {
			x := i18n.T("←→ : un autre filtre")
			if n := m.ailleurs(); m.recherche != "" && n > 0 {
				x += i18n.T(" (%d dans « tous »)", n)
			}
			pistes = append(pistes, x)
		}
		for k, p := range pistes {
			for j, x := range replier(strings.Fields(p), " ", d.lw-4) {
				if j > 0 && k > 0 {
					x = "  " + x // la suite d'une piste, sous son début
				}
				vide = append(vide, x)
			}
		}
	}
	for k := 0; k < vis; k++ {
		switch {
		case m.haut+k < nv:
			col = append(col, m.ligneListe(m.vue[m.haut+k], m.vue[m.haut+k] == m.i, d.lw))
		case k < len(vide):
			v := tronquer("  "+vide[k], d.lw)
			col = append(col, estompe+v+z+espaces(d.lw-largeur(v)))
		default:
			col = append(col, espaces(d.lw))
		}
	}
	col = append(col, indicateur(bas, d.lw))
	if d.agencement != compacte {
		pas := " " + m.pastilles(nom, d.lw-1)
		col = append(col, remplir(pas, d.lw))
	}
	return col
}

// apercuSeul : sans mini-fenêtre, la vraie statusline du thème, sous la liste.
func (m *Modele) apercuSeul(nom string, w int) []string {
	var l []string
	for _, x := range m.statusline(nom, w-4, 42) {
		l = append(l, tronquer(x+z, w))
	}
	return append([]string{""}, m.surFond(l)...)
}

// colonneFenetre : Pip perché au-dessus de la mini-fenêtre, ses pattes sur le cadre.
func (m *Modele) colonneFenetre(nom string) []string {
	var droite []string
	pattes := ""
	if m.d.mascotte {
		var cl [hauteurMascotte]string
		cl, pattes, _ = m.Mascotte(nom)
		x := espaces(m.d.ww - largeurMascotte - 3)
		for _, l := range cl {
			droite = append(droite, x+l)
		}
	}
	return append(droite, m.fenetre(nom, m.d.ww, m.d.wh, pattes)...)
}

// corpsAide : l'aide complète à la place de la liste ; sur deux colonnes si elle ne tient pas sur une.
func (m *Modele) corpsAide(w int) []string {
	l := lignesAide(w-2, modesAide[m.onglet])
	if len(l) <= m.d.corps || w < aideDeuxColonnes {
		return l
	}
	cw := (w - 3) / 2
	l = lignesAide(cw-1, modesAide[m.onglet])
	coupe := -1 // à la ligne vide la plus proche du milieu : une section ne se coupe pas
	for k, x := range l {
		if x == "" && (coupe < 0 || abs(k-len(l)/2) < abs(coupe-len(l)/2)) {
			coupe = k
		}
	}
	if coupe < 0 {
		coupe = len(l) / 2
	}
	g, dr := l[:coupe], l[min(coupe+1, len(l)):]
	var c []string
	for k := 0; k < max(len(g), len(dr)); k++ {
		x := espaces(cw)
		if k < len(g) {
			x = remplir(g[k], cw)
		}
		if k < len(dr) {
			x += "  " + dr[k]
		}
		c = append(c, x)
	}
	return c
}

func abs(x int) int { return max(x, -x) }

// View : l'écran complet, à la taille du terminal : en-tête, corps (selon la disposition), pied.
func (m *Modele) View() string {
	if m.Resultat != "" {
		return ""
	}
	nom := m.noms[m.i]
	d := m.d
	w := max(m.largeur, largeurMin)
	lignes := m.lignesEntete(nom, w)

	var corps []string
	switch {
	case m.aide:
		corps = m.corpsAide(w)
		if len(corps) > d.corps {
			corps = append(corps[:max(d.corps-1, 0)], estompe+tronquer(i18n.T("  … (agrandis le terminal pour tout voir)"), w-1)+z)
		}
	case m.onglet == 1:
		corps = m.corpsStyle(nom)
	case d.agencement == large:
		col, droite := m.colonneListe(nom), m.colonneFenetre(nom)
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
	case d.fenetre:
		corps = append(m.colonneListe(nom), m.colonneFenetre(nom)...)
	default:
		corps = append(m.colonneListe(nom), m.apercuSeul(nom, w-1)...)
	}
	for len(corps) < d.corps { // le pied reste en bas de l'écran
		corps = append(corps, "")
	}
	lignes = append(append(lignes, corps[:min(len(corps), max(d.corps, 0))]...), "")
	// Tout est colorié en 24 bits (sauf la statusline, rendue au bon niveau) : ramené ici au niveau du terminal
	return m.apercu.Couleurs.Adapter(strings.Join(append(lignes, d.pied...), "\n"))
}
