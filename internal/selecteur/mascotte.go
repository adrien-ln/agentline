package selecteur

import (
	"fmt"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
)

// Pip, la mascotte d'agentline, perché sur le bord de la mini-fenêtre comme un oiseau sur un fil. Il est dessiné
// en pixels, 18 de large sur 6 de haut, soit 9 colonnes et 3 lignes de terminal (2 × 2 pixels par cellule) : la
// tête, le corps, puis les pattes posées sur le cadre. Le corps prend les couleurs du thème (dégradé MODELE →
// fin du dégradé du nom, ou couleur unie), le bec et les pattes la couleur D75. Les creux forment les yeux ;
// « ▂ » est une paupière. Son humeur dépend de ce qui se passe : tes actions, la session fictive, ou l'ennui.

var modesMascotte = []string{i18n.M("dégradé"), i18n.M("unie"), i18n.M("aucune")} // traduits à l'affichage

const (
	largeurMascotte = 9
	hauteurMascotte = 3 // une ligne de place (sauts, « z »), puis la tête et le corps ; les pattes vont sur le cadre
	visageSombre    = "16161e"
	sommeil         = 20000 // ms sans touche avant de s'endormir
	paupiere        = '▂'
	// paupière à mi-hauteur, quand le corps est monté d'une demi-ligne
	paupiereHaute = '─'
)

// ouverts : les yeux qu'un clignement peut fermer.
var ouverts = map[string]bool{"ouverts": true, "gauche": true, "droite": true}

// Les pièces du dessin, en rangées de 18 pixels : « # » le corps, « a » le bec et les pattes, « o » un œil (un
// creux), « p » une paupière, « . » le vide. Rangée 0 : le haut de la tête ; 1 : les yeux ; 2 : le bec ; 3 : le
// bas du corps ; 4 : les pattes ; 5 : vide. Ailes : les rangées 1 à 3, avant d'y poser les yeux.
var (
	ailes = map[string][3]string{
		"bas":   {"....##########....", "..######aa######..", "...############..."},
		"levés": {"..##############..", "...#####aa#####...", "...############..."},
		"un":    {"..############....", "...#####aa######..", "...############..."},
	}
	// yeux : la colonne de chaque œil dans la rangée 1, en creux (o) ou fermé (p)
	yeux = map[string]map[int]byte{
		"ouverts": {6: 'o', 11: 'o'},
		"gauche":  {5: 'o', 10: 'o'},
		"droite":  {7: 'o', 12: 'o'},
		"fermés":  {6: 'p', 11: 'p'},
		"clin":    {6: 'o', 11: 'p'},
		"dos":     {}, // de dos : ni yeux ni bec
	}
	pieds = map[string]string{
		"deux":   "......a....a......",
		"gauche": "......a...........",
		"droite": "...........a......",
	}
	// profils : de côté, le bec en avant, pour le tour sur lui-même
	profilDroit  = [6]string{"......#######.....", ".....#####o###....", ".....#########aa..", "......########....", ".........a..a.....", vide18}
	profilGauche = [6]string{".....#######......", "....###o#####.....", "..aa#########.....", "....########......", ".....a..a.........", vide18}
	// tour : il se retourne, de face à de dos, puis revient de face
	tour = [][6]string{
		composer("bas", "droite", "deux"), profilDroit, composer("bas", "dos", "deux"), profilGauche, composer("bas", "gauche", "deux"),
	}
)

const vide18 = ".................."

const (
	pasTour  = 160 // ms par pose du tour
	pasDanse = 300 // ms par pas de danse
)

// composer : les 6 rangées de Pip à partir de ses pièces.
func composer(a, y, p string) [6]string {
	l := ailes[a]
	oeil := []byte(l[0])
	for k, v := range yeux[y] {
		oeil[k] = v
	}
	bec := l[1]
	if y == "dos" {
		bec = strings.ReplaceAll(bec, "a", "#")
	}
	return [6]string{".....########.....", string(oeil), bec, l[2], pieds[p], vide18}
}

// expression : la pose (ailes, yeux, pattes) et les effets.
type expression struct {
	ailes, yeux, pieds string
	saut               bool // un petit bond
	sommeil            bool // des « z » au-dessus de la tête
}

var expressions = map[string]expression{
	"normal":    {ailes: "bas", yeux: "ouverts", pieds: "deux"},
	"cligne":    {ailes: "bas", yeux: "fermés", pieds: "deux"},
	"clin":      {ailes: "bas", yeux: "clin", pieds: "deux"},
	"gauche":    {ailes: "bas", yeux: "gauche", pieds: "deux"},
	"droite":    {ailes: "bas", yeux: "droite", pieds: "deux"},
	"content":   {ailes: "levés", yeux: "ouverts", pieds: "deux", saut: true},
	"surpris":   {ailes: "un", yeux: "ouverts", pieds: "deux"},
	"reveil":    {ailes: "levés", yeux: "ouverts", pieds: "deux", saut: true},
	"triste":    {ailes: "bas", yeux: "dos", pieds: "deux"}, // il te tourne le dos
	"pensif":    {ailes: "un", yeux: "droite", pieds: "deux"},
	"endormi":   {ailes: "bas", yeux: "fermés", pieds: "deux", sommeil: true},
	"concentre": {ailes: "bas", yeux: "gauche", pieds: "deux"}, // il regarde la frappe, dans la fenêtre
	"saute":     {ailes: "levés", yeux: "ouverts", pieds: "deux"},
	"etire":     {ailes: "levés", yeux: "fermés", pieds: "deux"},
	"suit":      {ailes: "bas", yeux: "gauche", pieds: "deux"}, // il suit la liste des yeux quand on y navigue
	"baille":    {ailes: "un", yeux: "fermés", pieds: "deux"},  // l'aile devant le bec
	"somnole":   {ailes: "bas", yeux: "fermés", pieds: "deux"},
}

// danse : d'une patte sur l'autre, une aile puis les deux en l'air
var danse = [2]expression{{ailes: "levés", yeux: "ouverts", pieds: "gauche"}, {ailes: "un", yeux: "fermés", pieds: "droite"}}

// durées des réactions (ms) ; le bond ne dure que le début
var dureesReaction = map[string]int{"content": 1100, "surpris": 800, "reveil": 700, "clin": 700, "suit": 1200}

const (
	dureeSaut         = 350
	dureeAtterrissage = 100  // à mi-hauteur, avant de reposer les pattes
	dureeSomnolence   = 2600 // ms avant de s'endormir : il bâille, puis ses yeux se ferment
	dureeBaillement   = 1400
)

func (m *Modele) reagir(r string) { m.reaction, m.reactionT = r, m.t }

func (m *Modele) dort() bool { return m.t-m.derniereTouche > sommeil }

// humeur : l'expression du moment, de la plus prioritaire à la plus banale.
func (m *Modele) humeur() string {
	if len(m.vue) == 0 && m.onglet == 0 && m.reaction != "reveil" { // aucun résultat : la tristesse passe avant les réactions
		return "triste"
	}
	if d, ok := dureesReaction[m.reaction]; ok && m.t-m.reactionT < d {
		return m.reaction
	}
	switch {
	case m.confirmer || m.os.confirmer:
		return "danse"
	case m.dort():
		return "endormi"
	case m.t-m.derniereTouche > sommeil-dureeSomnolence+dureeBaillement:
		return "somnole"
	case m.t-m.derniereTouche > sommeil-dureeSomnolence:
		return "baille"
	}
	sc, ts := sceneA(m.scenes, m.t)
	e := sc.etatA(ts)
	switch {
	case e.reflexion >= 0:
		return "pensif"
	case e.saisie != "":
		return "concentre"
	}
	if g, _ := gesteA(m.t); g != "" {
		return g
	}
	return "normal"
}

// --- Gestes au repos -------------------------------------------------------------------------------
//
// Le temps est découpé en époques ; chacune tire au hasard, d'une graine fixe et de son numéro (donc de m.t
// seul : même instant, même geste), un geste et son début. Une époque qui tire le même geste que la
// précédente reste calme : deux gestes de suite ne se ressemblent jamais, et ces pauses rendent le rythme
// irrégulier. Les clignements suivent leur propre tirage, plus serré, par-dessus tout le reste.

type geste struct {
	nom          string
	duree, poids int // ms ; chances relatives d'être tiré
}

var gestes = []geste{
	{"gauche", 1200, 3}, // un coup d'œil d'un côté ou de l'autre
	{"droite", 1000, 3},
	{"alentour", 2200, 2}, // à gauche, puis à droite
	{"pietine", 1000, 2},  // d'une patte sur l'autre
	{"clin", 600, 1},
	{"coucou", 1200, 1}, // il agite le bras
	{"saute", 700, 1},   // un petit bond sur place
	{"etire", 1400, 1},  // bras levés, yeux fermés, sur la pointe des pieds
	{"tour", len(tour) * pasTour, 1},
}

const (
	epoqueGeste   = 4200 // ms ; au plus un geste par époque
	margeGeste    = 800  // ms de calme au moins au début et à la fin d'une époque
	epoqueCligne  = 3700 // ms ; un clignement (parfois double) par époque
	dureeCligne   = 150
	graineGeste   = 0xc1a3d
	graineDebut   = 0xdeb07
	graineCligne  = 0xc119
	chancesDouble = 4 // un clignement sur 4 est double
)

// hasard : un nombre pseudo-aléatoire tiré de la graine et de n (mélange splitmix64), toujours le même.
func hasard(graine, n uint64) uint64 {
	z := n*0x9e3779b97f4a7c15 + graine
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}

// tirage : l'index (dans gestes) tiré pour l'époque k, selon les poids.
func tirage(k int) int {
	total := 0
	for _, g := range gestes {
		total += g.poids
	}
	r := int(hasard(graineGeste, uint64(k)) % uint64(total))
	for i, g := range gestes {
		if r -= g.poids; r < 0 {
			return i
		}
	}
	return 0
}

// gesteA : le geste au repos en cours à l'instant t (ms) et le temps écoulé depuis son début ; "" si aucun.
// La première époque reste calme : on vient d'ouvrir le sélecteur.
func gesteA(t int) (nom string, ecoule int) {
	k := t / epoqueGeste
	if t < 0 || k == 0 {
		return "", 0
	}
	i := tirage(k)
	if i == tirage(k-1) {
		return "", 0
	}
	g := gestes[i]
	debut := margeGeste + int(hasard(graineDebut, uint64(k))%uint64(epoqueGeste-g.duree-2*margeGeste+1))
	if d := t%epoqueGeste - debut; d >= 0 && d < g.duree {
		return g.nom, d
	}
	return "", 0
}

// cligneA : les paupières sont-elles baissées à l'instant t ? Un clignement par époque, à un moment tiré au
// hasard, et de temps en temps un double.
func cligneA(t int) bool {
	if t < 0 {
		return false
	}
	h := hasard(graineCligne, uint64(t/epoqueCligne))
	debut := margeGeste/2 + int(h%uint64(epoqueCligne-2*margeGeste))
	d := t%epoqueCligne - debut
	if (h>>32)%chancesDouble == 0 {
		return d >= 0 && d < dureeCligne || d >= 2*dureeCligne && d < 3*dureeCligne
	}
	return d >= 0 && d < dureeCligne
}

// pose : l'expression de l'humeur h à cet instant, et la hauteur du corps en demi-lignes (0 : posé).
func (m *Modele) pose(h string) (expression, int) {
	e := expressions[h]
	_, ec := gesteA(m.t)
	switch h {
	case "danse":
		return danse[(m.t/pasDanse)%2], 0
	case "alentour":
		e = expressions["normal"]
		switch {
		case ec < 1000:
			e.yeux = "gauche"
		case ec >= 1300:
			e.yeux = "droite"
		}
	case "pietine":
		e = expressions["normal"]
		e.pieds = []string{"gauche", "droite"}[(ec/250)%2]
	case "coucou":
		e = expressions["normal"]
		e.ailes = []string{"un", "bas"}[(ec/200)%2]
	case "saute": // il prend son élan, monte, redescend
		return e, []int{1, 2, 2, 1, 0, 0, 0}[min(ec/100, 6)]
	case "etire":
		if ec >= 400 && ec < 1000 {
			return e, 1
		}
	}
	if e.saut && m.reaction != "" {
		switch d := m.t - m.reactionT; {
		case d < dureeSaut:
			return e, 2
		case d < dureeSaut+dureeAtterrissage:
			return e, 1
		}
	}
	return e, 0
}

// mascotteCouleurs : la couleur de chacune des 9 colonnes du corps.
func (m *Modele) mascotteCouleurs(nom string) [largeurMascotte]string {
	p := m.palette(nom)
	var c [largeurMascotte]string
	for k := range c {
		c[k] = p.Modele
		if m.mascotte == 0 {
			a, b := p.Modele, p.FinDegrade()
			r1, g1, b1 := ansi.RGB(a)
			r2, g2, b2 := ansi.RGB(b)
			c[k] = fmt.Sprintf("%02x%02x%02x", r1+(r2-r1)*k/(largeurMascotte-1), g1+(g2-g1)*k/(largeurMascotte-1), b1+(b2-b1)*k/(largeurMascotte-1))
		}
	}
	return c
}

// Mascotte : ses 3 lignes au-dessus de la fenêtre (place pour sauter comprise) et ses pattes, posées sur le
// cadre ou dans le vide pendant un bond. Rien si la mascotte est masquée.
func (m *Modele) Mascotte(nom string) (lignes [hauteurMascotte]string, pattes string, ok bool) {
	if m.mascotte == 2 {
		return lignes, "", false
	}
	h := m.humeur()
	e, haut := m.pose(h)
	var d [6]string
	if h == "tour" {
		_, ec := gesteA(m.t)
		d = tour[min(ec/pasTour, len(tour)-1)]
	} else {
		if ouverts[e.yeux] && cligneA(m.t) {
			e.yeux = "fermés"
		}
		d = composer(e.ailes, e.yeux, e.pieds)
	}
	p, c := m.palette(nom), m.mascotteCouleurs(nom)
	l := colorer(poser(d, haut), c, couleurBec(p.D75, p.D100, c[largeurMascotte/2]))
	if haut == 0 {
		l[0] = espaces(largeurMascotte)
		if e.sommeil { // des « z » qui montent, au rythme de la respiration
			l[0] = estompe + []string{"     z   ", "      z  ", "      Z  ", "       Z "}[(m.t/600)%4] + z
		}
	}
	if strings.TrimSpace(l[3]) != "" {
		pattes = l[3]
	}
	return [hauteurMascotte]string{l[0], l[1], l[2]}, pattes, true
}

// ecartBec : en dessous de cet écart CIELAB avec le corps, le bec ne se distingue plus (Tropique : 23).
const ecartBec = 30

// couleurBec : D75, sauf quand elle se confond avec le corps sous le bec (thèmes chauds : Tropique, Gruvbox) ;
// D100, le rouge des seuils, la remplace alors si elle s'en écarte davantage.
func couleurBec(d75, d100, corps string) string {
	if e := ansi.Ecart(d75, corps); e < ecartBec && ansi.Ecart(d100, corps) > e {
		return d100
	}
	return d75
}

// poser : le dessin (6 rangées) dans une grille de 8, soit 4 lignes de terminal : la ligne de place, la tête, le
// corps et celle du cadre. Posé, les pattes sont sur le cadre ; haut le monte d'autant de pixels (de demi-lignes).
// À mi-hauteur, les pattes rentrent : sinon elles partageraient une cellule avec le corps et le vide.
func poser(d [6]string, haut int) (g [8]string) {
	if haut%2 == 1 {
		d[4] = vide18
	}
	for i := range g {
		g[i] = vide18
	}
	for i, r := range d {
		g[2-haut+i] = r
	}
	return g
}

// quadrants : les blocs, indexés par leurs quarts pleins (1 haut gauche, 2 haut droite, 4 bas gauche, 8 bas droite).
var quadrants = []rune(" ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█")

// cellule : ce que le terminal dessine pour 4 pixels (haut gauche, haut droite, bas gauche, bas droite) : le
// caractère, la sorte de pixel qui donne sa couleur au texte et celle qui donne le fond (0 : le fond du
// terminal). Une cellule n'a que deux couleurs : ok est faux quand le dessin en demanderait trois.
func cellule(p [4]byte) (r rune, texte, fond byte, ok bool) {
	vide, corps, bec, pau := false, false, false, -1
	for i, x := range p {
		switch x {
		case '.', 'o':
			vide = true
		case '#':
			corps = true
		case 'a':
			bec = true
		case 'p':
			pau = i
		}
	}
	switch {
	case pau >= 0: // une paupière sombre sur le corps
		r = paupiereHaute
		if pau >= 2 {
			r = paupiere
		}
		return r, 'p', '#', !vide && !bec
	case corps && bec: // le bec sur le corps
		b := 0
		for i, x := range p {
			if x == 'a' {
				b |= 1 << i
			}
		}
		return quadrants[b], 'a', '#', !vide
	}
	b := 0
	for i, x := range p {
		if x == '#' || x == 'a' {
			b |= 1 << i
		}
	}
	if bec {
		return quadrants[b], 'a', 0, true
	}
	return quadrants[b], '#', 0, true
}

// colorer : les 4 lignes de la grille, chaque cellule à la couleur de sa colonne (corps), du bec ou de la paupière.
func colorer(g [8]string, c [largeurMascotte]string, bec string) (l [4]string) {
	couleur := func(x byte, k int) string {
		switch x {
		case 'a':
			return bec
		case 'p':
			return visageSombre
		}
		return c[k]
	}
	for i := range l {
		var s strings.Builder
		for k := range largeurMascotte {
			r, texte, fond, _ := cellule([4]byte{g[2*i][2*k], g[2*i][2*k+1], g[2*i+1][2*k], g[2*i+1][2*k+1]})
			switch {
			case r == ' ':
				s.WriteString(" ")
			case fond != 0:
				s.WriteString(ansi.Bg(couleur(fond, k)) + ansi.Fg(couleur(texte, k)) + string(r) + z)
			default:
				s.WriteString(ansi.Fg(couleur(texte, k)) + string(r) + z)
			}
		}
		l[i] = s.String()
	}
	return l
}
