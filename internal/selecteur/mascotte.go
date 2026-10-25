package selecteur

import (
	"fmt"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
)

// Clawd, la mascotte de Claude Code, perché sur le bord de la mini-fenêtre : son vrai dessin (9 colonnes,
// 3 lignes), le corps aux couleurs du thème (dégradé MODELE → fin du dégradé du nom, ou couleur unie) et les
// pattes posées sur le cadre. Les creux des blocs forment les yeux ; « ▂ » est une paupière. Son humeur
// dépend de ce qui se passe : tes actions, la session fictive, ou l'ennui.

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

// Les pièces du dessin d'origine. Bras : début et fin de la ligne de la tête, puis de celle du corps.
var (
	bras = map[string][4]string{
		"bas":   {" ▐", "", "▝▜", "█▀"},
		"levés": {"▗▟", "▄", " ▜", "█▘"},
		"un":    {" ▐", "▄", "▝▜", "█▘"},
	}
	yeux = map[string]string{
		"ouverts": "▛███▛█",
		"gauche":  "▟███▟█",
		"droite":  "█▟███▟",
		"fermés":  "▂███▂█",
		"clin":    "▛███▂█",
		"dos":     "██████",
	}
	pieds = map[string]string{
		"deux":   " ▝▝   ▝▝ ",
		"gauche": " ▝▝      ",
		"droite": "      ▝▝ ",
	}
	// tour : il pivote sur lui-même, de face à de dos, puis revient de face
	tour = [][3]string{
		{" ▐█▜██▛█ ", "▝▜██████▀", " ▝▝   ▝▝ "},
		{"  █▛██▛▌ ", " ▝█████▛ ", "  ▘▘  ▘▘ "},
		{"  ▐█▛█▜  ", "  ▐████  ", "  ▝▝ ▝▝  "},
		{"   ██▛▌  ", "   ███▌  ", "   ▘  ▘  "},
		{"   ▐██   ", "   ▐██   ", "   ▝ ▝   "},
		{"   ███▌  ", "   ███▌  ", "   ▘  ▘  "},
		{"  ▐████  ", "  ▐████  ", "  ▝▝ ▝▝  "},
		{"  █████▌ ", " ▝█████▛ ", "  ▘▘  ▘▘ "},
		{" ▐██████ ", "▝▜██████▀", " ▝▝   ▝▝ "},
		{"   ▛██▌  ", "   ███▌  ", "   ▘  ▘  "},
		{"  ▐▜▛██  ", "  ▐████  ", "  ▝▝ ▝▝  "},
		{"  ▛██▛█▌ ", " ▝█████▛ ", "  ▘▘  ▘▘ "},
		{" ▐▛███▜█ ", "▝▜██████▀", " ▝▝   ▝▝ "},
	}
)

const (
	pasTour  = 110 // ms par pose du tour
	pasDanse = 300 // ms par pas de danse
)

// dessiner : les 3 lignes de Clawd (9 colonnes chacune) à partir de ses pièces.
func dessiner(b, y, p string) [3]string {
	a := bras[b]
	tete := a[0] + yeux[y] + a[1]
	return [3]string{tete + espaces(largeurMascotte-len([]rune(tete))), a[2] + "█████" + a[3], pieds[p]}
}

// expression : la pose (bras, yeux, pattes) et les effets.
type expression struct {
	bras, yeux, pieds string
	saut              bool // un petit bond
	sommeil           bool // des « z » au-dessus de la tête
}

var expressions = map[string]expression{
	"normal":    {bras: "bas", yeux: "ouverts", pieds: "deux"},
	"cligne":    {bras: "bas", yeux: "fermés", pieds: "deux"},
	"clin":      {bras: "bas", yeux: "clin", pieds: "deux"},
	"gauche":    {bras: "bas", yeux: "gauche", pieds: "deux"},
	"droite":    {bras: "bas", yeux: "droite", pieds: "deux"},
	"content":   {bras: "levés", yeux: "ouverts", pieds: "deux", saut: true},
	"surpris":   {bras: "un", yeux: "ouverts", pieds: "deux"},
	"reveil":    {bras: "levés", yeux: "ouverts", pieds: "deux", saut: true},
	"triste":    {bras: "bas", yeux: "dos", pieds: "deux"}, // il te tourne le dos
	"pensif":    {bras: "un", yeux: "droite", pieds: "deux"},
	"endormi":   {bras: "bas", yeux: "fermés", pieds: "deux", sommeil: true},
	"concentre": {bras: "bas", yeux: "gauche", pieds: "deux"}, // il regarde la frappe, dans la fenêtre
	"saute":     {bras: "levés", yeux: "ouverts", pieds: "deux"},
	"etire":     {bras: "levés", yeux: "fermés", pieds: "deux"},
	"suit":      {bras: "bas", yeux: "gauche", pieds: "deux"}, // il suit la liste des yeux quand on y navigue
	"baille":    {bras: "un", yeux: "fermés", pieds: "deux"},  // la main devant la bouche
	"somnole":   {bras: "bas", yeux: "fermés", pieds: "deux"},
}

// danse : d'une patte sur l'autre, un bras puis les deux en l'air
var danse = [2]expression{{bras: "levés", yeux: "ouverts", pieds: "gauche"}, {bras: "un", yeux: "fermés", pieds: "droite"}}

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
		e.bras = []string{"un", "bas"}[(ec/200)%2]
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

// Clawd : ses 3 lignes au-dessus de la fenêtre (place pour sauter comprise) et ses pattes, posées sur le
// cadre ou dans le vide pendant un bond. Rien si la mascotte est masquée.
func (m *Modele) Mascotte(nom string) (lignes [hauteurMascotte]string, pattes string, ok bool) {
	if m.mascotte == 2 {
		return lignes, "", false
	}
	h := m.humeur()
	e, haut := m.pose(h)
	var d [3]string
	if h == "tour" {
		_, ec := gesteA(m.t)
		d = tour[min(ec/pasTour, len(tour)-1)]
	} else {
		if ouverts[e.yeux] && cligneA(m.t) {
			e.yeux = "fermés"
		}
		d = dessiner(e.bras, e.yeux, e.pieds)
	}

	c := m.mascotteCouleurs(nom)
	colorer := func(l string) string {
		var s strings.Builder
		for k, r := range []rune(l) {
			switch r {
			case ' ':
				s.WriteString(" ")
			case paupiere, paupiereHaute: // paupière sombre sur le corps
				s.WriteString(ansi.Bg(c[k]) + ansi.Fg(visageSombre) + string(r) + z)
			default:
				s.WriteString(ansi.Fg(c[k]) + string(r) + z)
			}
		}
		return s.String()
	}

	dessus := espaces(largeurMascotte)
	if e.sommeil { // des « z » qui montent, au rythme de la respiration
		z1 := []string{"     z   ", "      z  ", "      Z  ", "       Z "}[(m.t/600)%4]
		dessus = estompe + z1 + z
	}
	l, p := poser(d, haut)
	if haut == 0 {
		l[0] = dessus
	} else {
		l[0] = colorer(l[0])
	}
	if p != "" {
		p = colorer(p)
	}
	return [hauteurMascotte]string{l[0], colorer(l[1]), colorer(l[2])}, p, true
}

// poser : Clawd (tête, corps, pattes) monté de haut demi-lignes au-dessus du cadre : ses 3 lignes et ce qui
// reste posé sur le cadre (vide : les pattes sont en l'air). À 2, le corps monte d'une ligne entière.
func poser(d [3]string, haut int) (lignes [hauteurMascotte]string, pattes string) {
	switch {
	case haut >= 2:
		return [hauteurMascotte]string{d[0], d[1], d[2]}, ""
	case haut == 1: // à mi-hauteur : chaque ligne prend le bas de celle du dessous et le haut de la suivante
		var r [hauteurMascotte + 1][]rune
		prec := []rune(espaces(largeurMascotte))
		for i := range r {
			suiv := []rune(espaces(largeurMascotte))
			if i < len(d) {
				suiv = []rune(d[i])
			}
			r[i] = make([]rune, largeurMascotte)
			for k := range r[i] {
				b := points(prec[k])>>2 | (points(suiv[k])&3)<<2
				r[i][k] = quadrants[b]
				if prec[k] == paupiere && b == 15 {
					r[i][k] = paupiereHaute
				}
			}
			prec = suiv
		}
		if p := string(r[hauteurMascotte]); strings.TrimSpace(p) != "" {
			pattes = p
		}
		return [hauteurMascotte]string{string(r[0]), string(r[1]), string(r[2])}, pattes
	}
	return [hauteurMascotte]string{espaces(largeurMascotte), d[0], d[1]}, d[2]
}

// quadrants : les blocs du dessin, indexés par leurs quarts pleins (1 haut gauche, 2 haut droite, 4 bas
// gauche, 8 bas droite) ; une paupière compte pour un bloc plein.
var quadrants = []rune(" ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█")

func points(r rune) int {
	if r == paupiere {
		return 15
	}
	for i, q := range quadrants {
		if q == r {
			return i
		}
	}
	return 0
}
