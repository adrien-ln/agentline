// Package render produit les deux lignes de la statusline, octet pour octet comme l'ancienne version bash
// (vérifié par les sorties de référence de tests/golden/).
package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/entree/claude"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Env : tout ce que le rendu lit en dehors de l'entrée JSON.
type Env struct {
	Now      int64          // heure courante (AGENTLINE_NOW en test)
	Loc      *time.Location // fuseau des heures affichées
	Columns  int            // largeur du terminal (0 : inconnue)
	SSH      bool           // SSH_CONNECTION défini : ligne courte, sans glyphes Nerd Font
	SSHRiche bool           // AGENTLINE_SSH_RICH : les couleurs du thème même en SSH
	// Couleurs : niveau de couleurs (zéro : 24 bits) et fond du terminal. En SSH, sans AGENTLINE_SSH_RICH,
	// la ligne courte garde ses couleurs ANSI fixes, sauf si le niveau est imposé (CouleursImposees).
	Couleurs         ansi.Profil
	CouleursImposees bool          // AGENTLINE_COLORS ou NO_COLOR
	TempDir          string        // dossier du cache git
	GitDelai         time.Duration // AGENTLINE_GIT_TIMEOUT
	Themes           theme.Sources
	Getenv           func(string) string // AGENTLINE_THEME, AGENTLINE_PALETTE
	// GitFictif : état git imposé (aperçu du sélecteur, sans dépôt ni processus) ; nil : lu dans le dépôt.
	GitFictif *GitFictif
	Style     *style.Style // nil : le style par défaut
	// Palette : la palette imposée (aperçu de l'éditeur de thèmes, un thème pas encore enregistré) ; nil : celle
	// du thème en vigueur. Elle passe quand même par la correction pour l'autre fond.
	Palette *theme.Palette
}

// GitFictif : un état git décrit à la main.
type GitFictif struct {
	Branche                        string
	Indexe, Modifie, Neuf, Conflit int
	Avance, Retard                 int
}

const (
	z      = ansi.Z
	gras   = "\x1b[1m"
	maigre = "\x1b[22m"
	ital   = "\x1b[3m"
)

// fixes : les couleurs ANSI de base de la ligne courte en SSH (c'est la palette du terminal qui décide du
// rendu), sur fond sombre puis sur fond clair (les versions vives, jaune et cyan surtout, y sont
// illisibles : on prend les sombres).
type fixes struct {
	texte, discret, sep, eteint, dossier, lieu string
	vert, jaune, orange, rouge                 string
	intensite                                  [5]string
	haut, moyen, bas                           string // teinte d'un pourcentage : 80 % et plus, 50 % et plus, sinon
	modele                                     string
}

var (
	fixesSombre = fixes{
		texte: "\x1b[97m", discret: "\x1b[37m", sep: "\x1b[37m", eteint: "\x1b[90m", dossier: "\x1b[94m", lieu: "\x1b[92m",
		vert: "\x1b[92m", jaune: "\x1b[93m", orange: "\x1b[33m", rouge: "\x1b[91m",
		intensite: [5]string{"\x1b[96m", "\x1b[94m", "\x1b[95m", "\x1b[1;95m", "\x1b[1;33m"},
		haut:      "\x1b[91m", moyen: "\x1b[93m", bas: "\x1b[96m", modele: "\x1b[1;96m",
	}
	fixesClair = fixes{
		texte: "\x1b[30m", discret: "\x1b[90m", sep: "\x1b[90m", eteint: "\x1b[37m", dossier: "\x1b[34m", lieu: "\x1b[32m",
		vert: "\x1b[32m", jaune: "\x1b[33m", orange: "\x1b[91m", rouge: "\x1b[31m",
		intensite: [5]string{"\x1b[36m", "\x1b[34m", "\x1b[35m", "\x1b[1;35m", "\x1b[1;33m"},
		haut:      "\x1b[31m", moyen: "\x1b[33m", bas: "\x1b[36m", modele: "\x1b[1;36m",
	}
)

// melange : la couleur à pos/total du chemin de a vers b (arithmétique entière, comme en bash).
func melange(a, b string, pos, total int) string {
	va, _ := strconv.ParseUint(a, 16, 32)
	vb, _ := strconv.ParseUint(b, 16, 32)
	var r strings.Builder
	for _, d := range []uint{16, 8, 0} {
		x, y := int(va>>d&255), int(vb>>d&255)
		fmt.Fprintf(&r, "%02x", x+(y-x)*pos/total)
	}
	return r.String()
}

type rendu struct {
	env   Env
	p     theme.Palette // déjà adaptée au fond
	riche bool
	c     ansi.Profil
	f     *fixes // sans les couleurs du thème (SSH) : les couleurs ANSI fixes
	// places : non nil (RenderPlaces), composer y note où chaque item est dessiné
	places *[]Place

	texte, discret, sep, eteint, dossier, lieu, vert, jaune, orange, rouge string
	intensite                                                              [5]string

	iDossier, iBranche, iWorktree, iPR, iAlerte, iReset, iRapide, iEffortBas, iEffort string

	o    options
	g    etatGit
	sale bool   // modifications en cours (indexées, non indexées, nouveaux fichiers, conflits)
	nom  string // nom du dossier courant

	indice     bool // un indice facultatif (/compact) a été mis dans la ligne en cours
	sansIndice bool // la ligne ne tenait pas : la recomposer sans indice
}

func (r *rendu) degrade(p int) string {
	p = max(0, min(100, p))
	switch {
	case p <= 50:
		return melange(r.p.D0, r.p.D50, p, 50)
	case p <= 75:
		return melange(r.p.D50, r.p.D75, p-50, 25)
	default:
		return melange(r.p.D75, r.p.D100, p-75, 25)
	}
}

func (r *rendu) teinte(p int) string {
	switch {
	case r.riche:
		return r.c.Fg(r.degrade(p))
	case p >= 80:
		return r.c.SGR(r.f.haut)
	case p >= 50:
		return r.c.SGR(r.f.moyen)
	default:
		return r.c.SGR(r.f.bas)
	}
}

func (r *rendu) nomModele(t string) string {
	if !r.riche {
		return r.c.SGR(r.f.modele) + t + z
	}
	if r.o.modeleUnie {
		return gras + r.c.Fg(r.p.Modele) + t + z
	}
	c := []rune(t)
	total := max(len(c)-1, 1)
	var b strings.Builder
	b.WriteString(gras)
	for i, x := range c {
		b.WriteString(r.c.Fg(melange(r.p.Modele, r.p.FinDegrade(), i, total)))
		b.WriteRune(x)
	}
	return b.String() + z
}

// niveauxEffort : le rang de chaque niveau d'effort, dans la jauge.
var niveauxEffort = map[string]int{"low": 0, "medium": 1, "high": 2, "xhigh": 3, "max": 4}

func (r *rendu) puissance(niveau string) string {
	i, ok := niveauxEffort[niveau]
	if !ok {
		return ""
	}
	switch {
	case r.env.SSH:
		return r.intensite[i] + []string{"lo", "med", "hi", "xh", "max"}[i] + z
	case i < 2:
		return r.intensite[i] + r.iEffortBas + z
	default:
		return r.intensite[i] + r.iEffort + z
	}
}

// pct : comme printf '%.0f' (arrondi au plus proche, à égalité vers le pair).
func pct(s string) string {
	f, _ := strconv.ParseFloat(s, 64)
	return fmt.Sprintf("%.0f", f)
}

func entier(s string) int64 { // comme ${s%.*} puis l'arithmétique bash
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// jours : les jours de la semaine, en abrégé (en français, traduits à l'affichage).
var jours = []string{i18n.M("dim"), i18n.M("lun"), i18n.M("mar"), i18n.M("mer"), i18n.M("jeu"), i18n.M("ven"), i18n.M("sam")}

func (r *rendu) heure(t int64) string {
	h := time.Unix(t, 0).In(r.env.Loc)
	if t-r.env.Now < 86400 {
		return h.Format("15:04")
	}
	return i18n.T(jours[h.Weekday()]) + " " + h.Format("15:04")
}

func (r *rendu) signal(s string) string {
	if r.o.signal == "" {
		return s
	}
	return r.rouge + r.o.signal + z + " " + s
}

// etiquette : le libellé discret d'une valeur, suivi d'une espace ; rien s'il est vide.
func (r *rendu) etiquette(l string) string {
	if l == "" {
		return ""
	}
	return r.discret + l + z + " "
}

// ico : l'icône suivie d'une espace ; rien si elle est vide.
func ico(i string) string {
	if i == "" {
		return ""
	}
	return i + " "
}

func (r *rendu) valeur(v string, alerte bool) string {
	if alerte {
		return r.rouge + gras + v + "%" + z
	}
	n, _ := strconv.Atoi(v)
	return r.teinte(n) + gras + v + "%" + z
}

// largeurBarre : le nombre de cases de la barre de contexte.
const largeurBarre = 8

// barre : p % sur n cases, chacune pleine (━), à moitié (╸) ou éteinte (─), à la couleur de son milieu.
func (r *rendu) barre(p, n int) string {
	u := (p*2*n + 50) / 100 // demi-cases allumées
	var b strings.Builder
	for i := 0; i < n; i++ {
		if u > i*2 {
			b.WriteString(r.c.Fg(r.degrade((i*100 + 50) / n)))
			if u >= (i+1)*2 {
				b.WriteString(r.o.barre[0])
			} else {
				b.WriteString(r.o.barre[1])
			}
		} else {
			b.WriteString(r.eteint + r.o.barre[2])
		}
	}
	return b.String() + z
}

// quota : le segment d'une fenêtre d'utilisation, et s'il est en alerte (signal ● ajouté par le composeur).
func (r *rendu) quota(libelle, utilise, reset string, fenetre int64) (string, bool) {
	u := pct(utilise)
	ui, _ := strconv.ParseInt(u, 10, 64)
	ind, plafond := ui, ""
	if reset != "" {
		// Rythme : au moins 10 % de la fenêtre écoulé et 10 % consommé, sinon la projection n'a pas de sens
		if e := fenetre - (entier(reset) - r.env.Now); e*10 >= fenetre && ui >= 10 {
			ind = ui * fenetre / e
			if ind > 100 && ui < 100 && r.o.plafond {
				plafond = r.heure(r.env.Now + (100-ui)*e/ui)
			}
		}
	}
	alerte := ui >= int64(r.o.quotaAlerte) || plafond != ""
	s := r.etiquette(libelle) + r.valeur(u, alerte)
	// Quota épuisé : la projection n'apprend plus rien (elle affichait ↗103 % à 100 %)
	if r.o.projection && ind != ui && ind >= 75 && ui < 100 && plafond == "" {
		s += " " + r.teinte(int(ind)) + "↗" + strconv.FormatInt(ind, 10) + "%" + z
	}
	if reset != "" && (alerte && r.o.reset == "alert" || r.o.reset == "always") {
		s += " " + r.discret + ico(r.iReset) + r.heure(entier(reset)) + z
	}
	if plafond != "" {
		a := ""
		if r.iAlerte != "" {
			a = r.iAlerte + " "
		}
		s += " " + r.rouge + a + i18n.T("plafond ~%s", plafond) + z
	}
	return s, alerte
}

func cout(c string, anglais bool) string { // « 4,98 $ » (ou « $4.98 ») ; vide si absent ou nul
	if c == "" {
		return ""
	}
	f, _ := strconv.ParseFloat(c, 64)
	s := strings.Replace(fmt.Sprintf("%.2f", f), ".", ",", 1)
	if s == "0,00" {
		return ""
	}
	if anglais {
		return "$" + strings.Replace(s, ",", ".", 1)
	}
	return s + " $"
}

// afficher : une ligne, coupée avant le premier segment qui dépasse la largeur du terminal, et la colonne
// visible où commence chaque segment écrit.
func (r *rendu) afficher(sep string, segments []string) (string, []int) {
	largeur := 1000
	if r.env.Columns > 0 {
		largeur = r.env.Columns - 4
	}
	vsep := utf8.RuneCountInString(ansi.Visible(sep))
	var ligne strings.Builder
	var debuts []int
	vue := 0
	for _, s := range segments {
		t := utf8.RuneCountInString(ansi.Visible(s))
		if vue > 0 && vue+vsep+t > largeur {
			break
		}
		if ligne.Len() > 0 {
			ligne.WriteString(sep)
			vue += vsep
		}
		debuts = append(debuts, vue)
		ligne.WriteString(s)
		vue += t
	}
	return ligne.String() + "\n", debuts
}

// Render : l'entrée JSON de Claude Code -> la statusline (une ligne en SSH, deux sinon).
func Render(j string, env Env) string { return RenderSession(claude.Lire([]byte(j)), env) }

// RenderSession : la session -> la statusline (une ligne en SSH, deux sinon).
func RenderSession(ss etat.Session, env Env) string {
	r := preparer(ss, env)
	return r.composer(&ss, r.o.disp)
}

// RenderPlaces : comme RenderSession, avec la place de chaque item dessiné (le sélecteur s'en sert pour
// montrer l'item choisi sur l'aperçu).
func RenderPlaces(ss etat.Session, env Env) (string, []Place) {
	r := preparer(ss, env)
	places := []Place{}
	r.places = &places
	return r.composer(&ss, r.o.disp), places
}

// preparer : l'état git, le thème et les couleurs du rendu de la session.
func preparer(ss etat.Session, env Env) *rendu {
	rep, projet := ss.Dossier, ss.Projet

	// --- Git
	racine := theme.DepotRacine(rep)
	nettoyer(env.TempDir, env.Now)
	var g etatGit
	if f := env.GitFictif; f != nil {
		g = etatGit{f.Branche, f.Indexe, f.Modifie, f.Neuf, f.Conflit, f.Avance, f.Retard, true}
	} else if racine != "" {
		g = lireGit(racine, env.TempDir, env.Now, env.GitDelai)
	}
	sale := g.indexe+g.modifie+g.neuf+g.conflit != 0

	// --- Thème et couleurs
	d := projet
	if d == "" {
		d = rep
	}
	clair := env.Couleurs.Fond == ansi.Clair
	palette := env.Palette
	if palette == nil {
		p := env.Themes.ChoisirSurBranche(d, g.branche, env.Getenv).Palette
		palette = &p
	}
	r := &rendu{env: env, o: optionsDe(env.Style, env.SSH), p: palette.PourFond(clair),
		riche: !env.SSH || env.SSHRiche || env.CouleursImposees, c: env.Couleurs}
	c := r.c
	if r.riche {
		p := r.p
		r.texte, r.discret, r.sep, r.eteint = c.Fg(p.Texte), c.Fg(p.Discret), c.Fg(p.Sep), c.Fg(p.Eteint)
		r.dossier, r.lieu = c.Fg(p.Dossier), c.Fg(p.Lieu)
		r.vert, r.jaune, r.orange, r.rouge = c.Fg(p.D0), c.Fg(p.D50), c.Fg(p.D75), c.Fg(p.D100)
		for i, x := range p.Intensite {
			r.intensite[i] = c.Fg(x)
		}
	} else { // 16 couleurs ANSI fixes : c'est la palette du terminal qui décide du rendu
		f := &fixesSombre
		if clair {
			f = &fixesClair
		}
		r.f = f
		r.texte, r.discret, r.sep, r.eteint = c.SGR(f.texte), c.SGR(f.discret), c.SGR(f.sep), c.SGR(f.eteint)
		r.dossier, r.lieu = c.SGR(f.dossier), c.SGR(f.lieu)
		r.vert, r.jaune, r.orange, r.rouge = c.SGR(f.vert), c.SGR(f.jaune), c.SGR(f.orange), c.SGR(f.rouge)
		for i, x := range f.intensite {
			r.intensite[i] = c.SGR(x)
		}
	}
	r.iReset = "↻" // hors Nerd Font : toujours suivi d'une espace
	if !env.SSH {  // la ligne courte garde ses glyphes : le style ne les règle pas
		st := env.Style
		r.iDossier, r.iBranche, r.iWorktree, r.iPR = st.Glyphe("folder"), st.Glyphe("branch"), st.Glyphe("worktree"), st.Glyphe("pr")
		r.iAlerte, r.iReset = st.Glyphe("alert"), st.Glyphe("reset")
		r.iRapide = st.Glyphe("fast")
		r.iEffortBas, r.iEffort = st.Glyphe("effort.low"), st.Glyphe("effort.high")
	}
	r.g, r.sale, r.nom = g, sale, rep[strings.LastIndex(rep, "/")+1:]
	return r
}
