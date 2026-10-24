package selecteur

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
)

// Sessions Claude Code fictives, jouées en boucle dans la mini-fenêtre.
// Chaque scène est une chronologie calculée d'avance (générateur pseudo-aléatoire à graine fixe) :
// le timing est irrégulier comme chez un humain, mais identique d'une exécution à l'autre, donc testable.
// L'état à l'instant t s'obtient en rejouant les événements jusqu'à t.

// Couleurs de l'interface de Claude Code (fixes : seule la statusline suit le thème). Valeurs reprises
// de mémoire, à vérifier sur une vraie session.
var (
	cTexte   = "\x1b[97m"
	cOrange  = ansi.Fg("d77757")                     // orange de Claude (réflexion), à vérifier
	cVert    = ansi.Fg("4eba65")                     // outil terminé, test réussi
	cRouge   = ansi.Fg("ff6b80")                     //
	cAjout   = ansi.Fg("dcffdc") + ansi.Bg("225c2b") // ligne ajoutée d'un diff, à vérifier
	cRetrait = ansi.Fg("ffdcdc") + ansi.Bg("7a2936") // ligne retirée d'un diff, à vérifier
	cEnCours = ansi.Fg("969696")
)

// couleurs : les séquences de couleur, pour mesurer ou comparer le texte visible.
var couleurs = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var spinner = []string{"·", "✢", "✳", "✶", "✻", "✽", "✻", "✶", "✳", "✢"}

// etatScene : ce que montre la mini-fenêtre à un instant donné.
type etatScene struct {
	lignes    []string // conversation, déjà colorée
	saisie    string   // texte en cours de frappe dans la zone de saisie
	reflexion int      // début de la réflexion en cours (ms), -1 si aucune
	verbe     string   // « Cogitating… », « Thinking… »
	outils    int      // outils terminés : le contexte augmente
}

type evenement struct {
	t int
	f func(*etatScene, int)
}

type scene struct {
	nom     string
	evts    []evenement
	duree   int
	frappe  []int    // délais entre deux frappes (pour les tests de réalisme)
	finales []string // la conversation à la fin de la scène (historique des scènes suivantes)
}

type constructeur struct {
	s   scene
	t   int
	rnd *rand.Rand
}

func nouvelleScene(nom string, graine int64) *constructeur {
	return &constructeur{s: scene{nom: nom}, rnd: rand.New(rand.NewSource(graine))}
}

func (c *constructeur) entre(a, b int) int { return a + c.rnd.Intn(b-a+1) }

func (c *constructeur) a(f func(*etatScene, int)) { c.s.evts = append(c.s.evts, evenement{c.t, f}) }

func (c *constructeur) pause(a, b int) { c.t += c.entre(a, b) }

// taper : frappe humaine, avec des fautes corrigées aussitôt (retour arrière).
func (c *constructeur) taper(texte string) {
	voisins := "azertyuiopqsdfghjklmwxcvbn" // un clavier français, ou anglais
	if !i18n.Francais() {
		voisins = "qwertyuiopasdfghjklzxcvbnm"
	}
	precedent := ' '
	for _, r := range texte {
		d := c.entre(45, 135)
		switch {
		case precedent == ' ':
			d += c.entre(0, 90) // on cherche le mot suivant
		case strings.ContainsRune(",.;:!?'", precedent):
			d += c.entre(150, 380)
		}
		if r != ' ' && c.rnd.Intn(30) == 0 { // faute de frappe, vue puis corrigée
			faux := string(voisins[c.rnd.Intn(len(voisins))])
			c.t += d
			c.s.frappe = append(c.s.frappe, d)
			c.a(func(e *etatScene, _ int) { e.saisie += faux })
			d = c.entre(180, 420)
			c.t += d
			c.s.frappe = append(c.s.frappe, d)
			c.a(func(e *etatScene, _ int) { e.saisie = e.saisie[:len(e.saisie)-len(faux)] })
			d = c.entre(70, 160)
		}
		c.t += d
		c.s.frappe = append(c.s.frappe, d)
		x := string(r)
		c.a(func(e *etatScene, _ int) { e.saisie += x })
		precedent = r
	}
}

// effacer : la saisie vidée d'un coup (Ctrl+U), après une hésitation.
func (c *constructeur) effacer() {
	c.pause(300, 700)
	c.a(func(e *etatScene, _ int) { e.saisie = "" })
}

// rappeler : ↑ rappelle le message précédent dans la zone de saisie.
func (c *constructeur) rappeler(texte string) {
	c.pause(400, 900)
	c.a(func(e *etatScene, _ int) { e.saisie = texte })
}

// valider : Entrée ; le message passe dans la conversation.
func (c *constructeur) valider() {
	c.pause(250, 700)
	c.a(func(e *etatScene, _ int) {
		e.lignes = append(e.lignes, estompe+"> "+e.saisie+z, "")
		e.saisie = ""
	})
}

func (c *constructeur) reflechir(verbe string, a, b int) {
	debut := c.t
	c.a(func(e *etatScene, _ int) { e.reflexion, e.verbe = debut, verbe })
	c.pause(a, b)
	c.a(func(e *etatScene, _ int) { e.reflexion = -1 })
}

// outil : « ⏺ Nom(argument) » en cours, puis son résultat « ⎿ … ».
func (c *constructeur) outil(nom, arg string, a, b int, resultat ...string) {
	c.a(func(e *etatScene, _ int) { e.lignes = append(e.lignes, cEnCours+"⏺"+z+" "+gras+nom+z+"("+arg+")") })
	c.pause(a, b)
	c.a(func(e *etatScene, _ int) {
		e.lignes[len(e.lignes)-1] = cVert + "⏺" + z + " " + gras + nom + z + "(" + arg + ")"
		for k, r := range resultat {
			prefixe := "     "
			if k == 0 {
				prefixe = "  ⎿  "
			}
			e.lignes = append(e.lignes, prefixe+r)
		}
		e.outils++
	})
	c.pause(200, 600)
}

// repondre : la réponse arrive mot par mot.
func (c *constructeur) repondre(texte string) {
	mots := strings.Fields(texte)
	c.a(func(e *etatScene, _ int) { e.lignes = append(e.lignes, cTexte+"⏺"+z) })
	for k := range mots {
		n := k + 1
		c.t += c.entre(25, 85)
		c.a(func(e *etatScene, _ int) {
			e.lignes[len(e.lignes)-1] = cTexte + "⏺" + z + " " + strings.Join(mots[:n], " ")
		})
	}
	c.a(func(e *etatScene, _ int) { e.lignes = append(e.lignes, "") })
}

func (c *constructeur) fin() scene {
	c.pause(2500, 3500) // le temps de lire avant la scène suivante
	c.s.duree = c.t
	c.s.finales = c.s.etatA(c.s.duree).lignes
	return c.s
}

func ajout(s string) string   { return cAjout + "+ " + s + z }
func retrait(s string) string { return cRetrait + "- " + s + z }

// Scenes : les trois sessions jouées en boucle, dans la langue des textes (l'interface de Claude Code
// reste en anglais, comme la vraie).
func Scenes() []scene {
	var s []scene
	sombre := i18n.T("ajoute un mode sombre à la page d'accueil, qui suit le réglage du système")

	c := nouvelleScene("code", 11)
	c.pause(900, 1500)
	c.taper(sombre)
	c.valider()
	c.reflechir("Cogitating…", 1800, 3200)
	c.outil("Read", "site/index.html", 300, 700, "Read 48 lines")
	c.outil("Read", "site/style.css", 250, 600, "Read 112 lines")
	c.outil("Update", "site/style.css", 900, 1600, "Updated site/style.css with 9 additions and 1 removal",
		ajout("@media (prefers-color-scheme: dark) {"), ajout(i18n.T("  :root { --fond: #0d1117; --texte: #e6edf3; }")), retrait("body { background: white; }"))
	c.repondre(i18n.T("C'est fait : les couleurs passent par des variables, et le mode sombre suit le réglage du système."))
	s = append(s, c.fin())

	c = nouvelleScene("revue", 23)
	c.pause(1200, 1800)
	c.rappeler(sombre)
	c.effacer()
	c.taper(i18n.T("relis la PR #12 avant que je la fusionne"))
	c.valider()
	c.reflechir("Thinking…", 1500, 2600)
	c.outil("Bash", i18n.T("git diff main...feat/filtres --stat"), 700, 1400, "4 files changed, 142 insertions(+), 37 deletions(-)")
	c.outil("Read", "internal/selecteur/touches.go", 400, 900, "Read 412 lines")
	c.a(func(e *etatScene, _ int) {
		e.lignes = append(e.lignes, cTexte+"⏺"+z+" Update Todos", "  ⎿  ☒ "+i18n.T("Lire le diff"), "     ☒ "+i18n.T("Vérifier les tests"), "     ☐ "+i18n.T("Rédiger la revue"))
	})
	c.pause(900, 1500)
	c.repondre(i18n.T("Deux remarques : la recherche ignore les accents du titre, et un test vérifie encore l'ancien nombre de lignes. Le reste est bon."))
	s = append(s, c.fin())

	c = nouvelleScene("erreur", 37)
	c.pause(800, 1400)
	c.taper(i18n.T("lance les tests"))
	c.valider()
	c.outil("Bash", "go test ./...", 1500, 2600, cRouge+"--- FAIL: TestFiltres (0.01s)"+z, i18n.T("    filtre populaire : 10 thèmes, attendu 11"), cRouge+"FAIL"+z)
	c.reflechir("Pondering…", 2000, 3400)
	c.outil("Update", "internal/selecteur/selecteur.go", 700, 1300, "Updated internal/selecteur/selecteur.go with 1 addition and 1 removal",
		retrait("for k := 1; k < len(m.noms); k++ {"), ajout("for k := range m.noms {"))
	c.outil("Bash", "go test ./...", 1400, 2300, cVert+"ok"+z+"  github.com/adrien-ln/agentline/internal/selecteur  0.48s")
	c.repondre(i18n.T("Corrigé : la boucle sautait le premier thème, d'où un thème populaire en moins."))
	s = append(s, c.fin())
	return s
}

// etatA : l'état de la scène à l'instant t (ms depuis son début).
func (s scene) etatA(t int) etatScene {
	e := etatScene{reflexion: -1}
	for _, ev := range s.evts {
		if ev.t > t {
			break
		}
		ev.f(&e, t)
	}
	return e
}

// sceneA : la scène jouée à l'instant t (ms depuis l'ouverture), et le temps écoulé dans cette scène.
func sceneA(sc []scene, t int) (scene, int) {
	s, ts, _ := sceneIndexA(sc, t)
	return s, ts
}

func sceneIndexA(sc []scene, t int) (scene, int, int) {
	total := 0
	for _, s := range sc {
		total += s.duree
	}
	t %= total
	for k, s := range sc {
		if t < s.duree {
			return s, t, k
		}
		t -= s.duree
	}
	return sc[0], 0, 0
}

// historique : la conversation des scènes jouées avant celle de l'instant t, comme si la session
// continuait (les plus anciennes d'abord) ; une grande fenêtre est ainsi remplie naturellement.
func historique(sc []scene, t int) []string {
	_, _, k := sceneIndexA(sc, t)
	var l []string
	for j := len(sc) - 1; j >= 1; j-- {
		p := sc[(k-j+len(sc))%len(sc)]
		if p.finales == nil {
			p.finales = p.etatA(p.duree).lignes
		}
		l = append(l, p.finales...)
	}
	return l
}

// lignesConversation : les h dernières lignes de la conversation, réflexion en cours comprise.
func (e etatScene) lignesConversation(t, h, largeur int, avant []string) []string {
	brutes := append(append([]string(nil), avant...), e.lignes...)
	if e.reflexion >= 0 {
		d := t - e.reflexion
		brutes = append(brutes, cOrange+spinner[(t/120)%len(spinner)]+" "+e.verbe+z+estompe+" ("+strconv.Itoa(d/1000)+"s · esc to interrupt)"+z)
	}
	var l []string
	for _, x := range brutes {
		l = append(l, envelopper(x, largeur)...)
	}
	if len(l) > h {
		l = l[len(l)-h:]
	}
	for len(l) < h {
		l = append(l, "")
	}
	return l
}

// envelopper : la ligne répartie sur plusieurs lignes de n caractères visibles au plus, coupée de
// préférence à une espace, comme le fait Claude Code ; la suite est décalée de 5 colonnes (sous « ⎿  »)
// et garde la couleur en cours (fond des lignes de diff compris).
func envelopper(s string, n int) []string {
	if n < 10 || largeur(s) <= n {
		return []string{s}
	}
	retraitSuite := "  " // sous le texte d'un message ou d'une réponse
	if strings.HasPrefix(couleurs.ReplaceAllString(s, ""), "  ") {
		retraitSuite = "     " // sous « ⎿  » : résultat d'outil, diff
	}
	var lignes []string
	var ligne, actif strings.Builder // la ligne en cours, et les couleurs en vigueur
	vus, derniereEspace, retrait := 0, -1, ""
	courant := func() string { return ligne.String() }
	for len(s) > 0 {
		if s[0] == 0x1b {
			if j := strings.IndexByte(s, 'm'); j >= 0 {
				seq := s[:j+1]
				if seq == z {
					actif.Reset()
				} else {
					actif.WriteString(seq)
				}
				ligne.WriteString(seq)
				s = s[j+1:]
				continue
			}
		}
		r, t := utf8.DecodeRuneInString(s)
		if vus+cellules.RuneWidth(r) > n { // ligne pleine : on la coupe à la dernière espace si possible
			c := courant()
			reste := ""
			if derniereEspace > len(c)/2 {
				reste, c = c[derniereEspace+1:], c[:derniereEspace]
			}
			lignes = append(lignes, c+z)
			retrait = retraitSuite
			ligne.Reset()
			ligne.WriteString(retrait + actif.String() + reste)
			vus, derniereEspace = largeur(retrait)+largeur(reste), -1
		}
		if r == ' ' {
			derniereEspace = ligne.Len()
		}
		ligne.WriteString(s[:t])
		s = s[t:]
		vus += cellules.RuneWidth(r)
	}
	return append(lignes, courant())
}

// couper : le texte ramené à n colonnes, séquences d'échappement préservées (un caractère large qui
// déborderait n'est pas écrit).
func couper(s string, n int) string {
	var b strings.Builder
	vus := 0
	for len(s) > 0 {
		if s[0] == 0x1b {
			if j := strings.IndexByte(s, 'm'); j >= 0 {
				b.WriteString(s[:j+1])
				s = s[j+1:]
				continue
			}
		}
		r, t := utf8.DecodeRuneInString(s)
		if vus+cellules.RuneWidth(r) > n {
			break
		}
		b.WriteString(s[:t])
		s = s[t:]
		vus += cellules.RuneWidth(r)
	}
	return b.String() + z
}
