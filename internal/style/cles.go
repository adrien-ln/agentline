package style

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Items : les items de la statusline, dans l'ordre de la disposition par défaut (le registre du rendu,
// internal/render/items.go, a les mêmes noms : un test le vérifie). Les anciens noms français sont encore
// lus (anciens.go).
var Items = []string{"model", "effort", "fast", "folder", "git", "worktree", "pr", "agent", "style", "session",
	"ctx", "5h", "7d", "cost"}

// Les dispositions par défaut (« | » sépare les segments, l'espace colle les items d'un segment).
const (
	Ligne1Defaut = "model effort fast | folder git worktree | pr | agent | style | session"
	Ligne2Defaut = "ctx | 5h | 7d | cost"
	CourtDefaut  = "model effort | folder git | ctx | cost | 5h"
)

// Genre : la forme des valeurs d'une clé.
type Genre int

const (
	Choix       Genre = iota // une valeur de Valeurs
	Nombre                   // un entier de Min à Max
	Libelle                  // un texte court
	Glyphe                   // quelques caractères d'une colonne (vide permis)
	Barre                    // un nom de Valeurs, ou trois caractères d'une colonne
	Disposition              // des items, « | » entre les segments
	ListeItems               // des items séparés par des espaces
	Branches                 // des noms de branches séparés par des espaces (vide permis)
)

// Cle : une option du style. C'est la seule description des options : la CLI (agentline style keys), le
// sélecteur et la validation la lisent.
type Cle struct {
	Nom      string
	Genre    Genre
	Valeurs  []string // Choix, Barre : les valeurs nommées, dans l'ordre où le sélecteur les fait défiler
	Min, Max int      // Nombre
	Defaut   string
	Aide     string   // en français, traduite par Explication
	Items    []string // les items que l'option règle (vide : options générales) ; le sélecteur les y range
	SSH      bool     // la ligne courte (SSH) en tient compte ; sinon elle garde sa variante compacte
	aideArg  string   // ce que Aide formate (%s), s'il y a lieu
}

// Les glyphes Nerd Font par défaut, et leurs remplaçants en texte (icons=text).
var (
	glyphesNerd = map[string]string{
		"folder": "", "branch": "", "worktree": "", "pr": "", "alert": "",
		"reset": "", "fast": "\U000f0211", "effort.low": "\U000f140c", "effort.high": "\U000f140b", "signal": "●",
		// effort sur 5 niveaux (effort.levels=5) : compteurs de vitesse, chrono rapide, fusée
		"level.low": "\U000f0f86", "level.medium": "\U000f0f85", "level.high": "\U000f04c5", "level.xhigh": "\U000f0152", "level.max": "\U000f14de",
		// compteurs git (git.counts=icons) : indexés, modifiés, nouveaux, en conflit, avance, retard
		"git.staged": "", "git.modified": "", "git.untracked": "", "git.conflict": "", "git.ahead": "", "git.behind": "",
		// état de la PR (pr.display=icon) : approuvée, à corriger, en revue, brouillon
		"pr.approved": "", "pr.changes": "", "pr.review": "", "pr.draft": "",
		// libellés en icônes (labels=icons) : puce mémoire, chronomètre, calendrier de la semaine, billets
		"ctx": "\U000f035b", "5h": "\U000f051b", "7d": "\U000f0a33", "cost": "\U000f0114",
	}
	glyphesTexte = map[string]string{
		"folder": "▸", "branch": "⎇", "worktree": "⊕", "pr": "PR", "alert": "!",
		"reset": "↻", "fast": "»", "effort.low": "▿", "effort.high": "▴", "signal": "●",
		"git.staged": "+", "git.modified": "●", "git.untracked": "?", "git.conflict": "×", "git.ahead": "↑", "git.behind": "↓",
		"level.low": "▁", "level.medium": "▃", "level.high": "▅", "level.xhigh": "▇", "level.max": "█",
		// sans Nerd Font, le mot ou le libellé en texte reste
		"pr.approved": "", "pr.changes": "", "pr.review": "", "pr.draft": "",
		"ctx": "", "5h": "", "7d": "", "cost": "",
	}
	// nomsGlyphes : les icônes réglables une à une (icon.<nom>), dans l'ordre de « agentline style keys ».
	nomsGlyphes = []string{"folder", "branch", "worktree", "pr", "alert", "reset", "fast", "effort.low", "effort.high", "signal",
		"level.low", "level.medium", "level.high", "level.xhigh", "level.max",
		"git.staged", "git.modified", "git.untracked", "git.conflict", "git.ahead", "git.behind",
		"pr.approved", "pr.changes", "pr.review", "pr.draft", "ctx", "5h", "7d", "cost"}
	// itemsGlyphes : l'item de chaque icône (vide : générale).
	itemsGlyphes = map[string][]string{
		"folder": {"folder"}, "branch": {"git"}, "worktree": {"worktree"}, "pr": {"pr"},
		"alert": {"5h", "7d"}, "reset": {"5h", "7d"}, "fast": {"fast"}, "effort.low": {"effort"}, "effort.high": {"effort"},
		"ctx": {"ctx"}, "5h": {"5h"}, "7d": {"7d"}, "cost": {"cost"},
		"level.low": {"effort"}, "level.medium": {"effort"}, "level.high": {"effort"}, "level.xhigh": {"effort"}, "level.max": {"effort"},
		"git.staged": {"git"}, "git.modified": {"git"}, "git.untracked": {"git"}, "git.conflict": {"git"}, "git.ahead": {"git"}, "git.behind": {"git"},
		"pr.approved": {"pr"}, "pr.changes": {"pr"}, "pr.review": {"pr"}, "pr.draft": {"pr"},
	}
	// glyphesLibelles : les icônes qui remplacent un libellé (labels=icons) ; vides, le libellé reste.
	glyphesLibelles = map[string]bool{"ctx": true, "5h": true, "7d": true, "cost": true}
	// glyphesEtatPR : les icônes qui remplacent l'état de la PR (pr.display=icon) ; vides, le mot reste.
	glyphesEtatPR = map[string]bool{"pr.approved": true, "pr.changes": true, "pr.review": true, "pr.draft": true}
	// Barres nommées de ctx.bar : pleine, à moitié, éteinte ; pie : pas de barre, un camembert (Jauge).
	barres = map[string]string{"lines": "━╸─", "blocks": "█▌░", "dots": "●◐○", "pie": "", "none": ""}
	// jauges : les glyphes d'une jauge en un caractère, du vide au plein, par forme et jeu d'icônes ; la
	// batterie se vide (elle montre ce qui reste), le camembert se remplit.
	jauges = map[string]map[string][]string{
		"pie": {
			"nerd": {"\U000f0130", "\U000f0a9e", "\U000f0a9f", "\U000f0aa0", "\U000f0aa1", "\U000f0aa2", "\U000f0aa3", "\U000f0aa4", "\U000f0aa5"},
			"text": {"○", "◔", "◑", "◕", "●"},
		},
		"battery": {
			"nerd": {"\U000f008e", "\U000f007a", "\U000f007b", "\U000f007c", "\U000f007d", "\U000f007e", "\U000f007f", "\U000f0080", "\U000f0081", "\U000f0082", "\U000f0079"},
		},
	}
)

// iconesModeles : les jeux d'icônes de model.icon, par famille de modèle Claude. Écriture : plume (haïku),
// stylo-plume (sonnet), parchemin (opus), livre ouvert (fable) ; poids : plume, diamant creux, diamant, couronne.
var iconesModeles = map[string]map[string]string{
	"writing": {"haiku": "\U000f06d3", "sonnet": "\U000f0d13", "opus": "\U000f0bc2", "fable": "\U000f05da"},
	"weight":  {"haiku": "\U000f06d3", "sonnet": "\U000f0b8b", "opus": "\U000f0b8a", "fable": "\U000f01a5"},
}

// IconeModele : l'icône du modèle nommé (« Opus 5.5 », « Claude Sonnet 4.5 ») dans le jeu de model.icon ;
// vide pour un autre modèle, sans jeu (none) ou sans Nerd Font (icons autre que nerd).
func IconeModele(jeuModele, icons, modele string) string {
	if icons != "nerd" {
		return ""
	}
	for _, mot := range strings.Fields(strings.ToLower(modele)) {
		if g, ok := iconesModeles[jeuModele][mot]; ok {
			return g
		}
	}
	return ""
}

// GlyphesModeles : les icônes de model.icon, jeu par jeu, pour « agentline font test ».
func GlyphesModeles() string {
	var b strings.Builder
	for _, jeu := range []string{"writing", "weight"} {
		for _, f := range []string{"haiku", "sonnet", "opus", "fable"} {
			b.WriteString(iconesModeles[jeu][f])
		}
	}
	return b.String()
}

// GlyphesJauges : tous les glyphes des jauges, pour « agentline font test ».
func GlyphesJauges() string {
	var b strings.Builder
	for _, forme := range []string{"pie", "battery"} {
		for _, jeu := range []string{"nerd", "text"} {
			b.WriteString(strings.Join(jauges[forme][jeu], ""))
		}
	}
	return b.String()
}

// Jauge : le glyphe de la forme (pie, battery) pour p % utilisés, dans le jeu d'icônes ; vide si la forme
// n'existe pas dans ce jeu (battery sans Nerd Font, icons=none). Une part non nulle n'est jamais vide.
func Jauge(forme, jeu string, p int) string {
	g := jauges[forme][jeu]
	if len(g) == 0 {
		return ""
	}
	p = max(0, min(100, p))
	if forme == "battery" {
		p = 100 - p
	}
	n := len(g) - 1
	k := (p*n + 50) / 100
	if k == 0 && p > 0 {
		k = 1
	}
	return g[k]
}

// Auto : la valeur des icônes « selon le réglage icons », et des libellés et formats qui suivent la langue.
const Auto = "auto"

// Oui, Non : les valeurs des options à deux états.
const (
	Oui = "yes"
	Non = "no"
)

var ouiNon = []string{Oui, Non}

var catalogue = func() []Cle {
	quotas := []string{"5h", "7d"}
	c := []Cle{
		{Nom: "line1", Genre: Disposition, Defaut: Ligne1Defaut, Aide: i18n.M("items de la 1re ligne ; « | » entre les segments, l'espace colle")},
		{Nom: "line2", Genre: Disposition, Defaut: Ligne2Defaut, Aide: i18n.M("items de la 2e ligne")},
		{Nom: "short", Genre: Disposition, Defaut: CourtDefaut, Aide: i18n.M("items de la ligne courte (SSH)"), SSH: true},
		{Nom: "hide", Genre: ListeItems, Defaut: "", Aide: i18n.M("items jamais affichés, séparés par des espaces"), SSH: true},

		{Nom: "icons", Genre: Choix, Valeurs: []string{"nerd", "text", "none"}, Defaut: "nerd", Aide: i18n.M("jeu d'icônes : Nerd Font, caractères ordinaires, aucune")},
		{Nom: "sep.line1", Genre: Glyphe, Defaut: "│", Aide: i18n.M("séparateur des segments de la 1re ligne")},
		{Nom: "sep.line2", Genre: Glyphe, Defaut: "·", Aide: i18n.M("séparateur des segments de la 2e ligne")},
		{Nom: "sep.short", Genre: Glyphe, Defaut: "·", Aide: i18n.M("séparateur de la ligne courte (SSH)"), SSH: true},
		{Nom: "ssh.short", Genre: Choix, Valeurs: ouiNon, Defaut: Non, Aide: i18n.M("en SSH (SSH_CONNECTION défini) : la ligne courte, sans icônes, au lieu de la statusline complète"), SSH: true},
	}
	for _, n := range nomsGlyphes {
		aide := i18n.M("icône %s (auto : selon icons ; vide : aucune)")
		if glyphesLibelles[n] {
			aide = i18n.M("icône %s, avec labels=icons (auto : selon icons ; vide : le libellé)")
		}
		if glyphesEtatPR[n] {
			aide = i18n.M("icône %s, avec pr.display=icon (auto : selon icons ; vide : le mot)")
		}
		if strings.HasPrefix(n, "git.") {
			aide = i18n.M("icône %s, avec git.counts=icons (auto : selon icons ; vide : le symbole)")
		}
		c = append(c, Cle{Nom: "icon." + n, Genre: Glyphe, Defaut: Auto, Items: itemsGlyphes[n], Aide: aide, aideArg: strings.ReplaceAll(n, ".", " ")})
	}
	return append(c,
		Cle{Nom: "model.name", Genre: Choix, Valeurs: []string{"full", "short"}, Defaut: "full", Items: []string{"model"}, Aide: i18n.M("nom du modèle en entier, ou son premier mot")},
		Cle{Nom: "model.color", Genre: Choix, Valeurs: []string{"gradient", "solid"}, Defaut: "gradient", Items: []string{"model"}, Aide: i18n.M("nom du modèle en dégradé, ou d'une couleur")},
		Cle{Nom: "model.icon", Genre: Choix, Valeurs: []string{"none", "writing", "weight"}, Defaut: "none", Items: []string{"model"}, Aide: i18n.M("icône devant le modèle, selon sa famille (Haiku, Sonnet, Opus, Fable) : aucune, écriture, poids")},
		Cle{Nom: "effort.display", Genre: Choix, Valeurs: []string{"icon+text", "icon", "text"}, Defaut: "icon+text", Items: []string{"effort"}, Aide: i18n.M("niveau d'effort : jauge et nom, jauge seule, nom seul")},
		Cle{Nom: "effort.levels", Genre: Choix, Valeurs: []string{"2", "5"}, Defaut: "2", Items: []string{"effort"}, Aide: i18n.M("icônes de l'effort : 2 (bas, haut), ou une par niveau (icon.level.low … icon.level.max)")},
		Cle{Nom: "git.status", Genre: Choix, Valeurs: []string{"detail", "summary"}, Defaut: "detail", Items: []string{"git"}, Aide: i18n.M("compteurs (+1 ●2 ?3), ou un seul signe (✓ ou *)")},
		Cle{Nom: "git.counts", Genre: Choix, Valeurs: []string{"symbols", "icons"}, Defaut: "symbols", Items: []string{"git"}, Aide: i18n.M("signes des compteurs : symboles (+1 ●2), ou icônes espacées (icon.git.staged…)")},
		Cle{Nom: "git.protected", Genre: Branches, Defaut: "main master", Items: []string{"git"}, Aide: i18n.M("branches où des changements non commités mettent la statusline en alerte (vide : jamais)"), SSH: true},
		Cle{Nom: "git.upstream", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"git"}, Aide: i18n.M("commits d'avance et de retard (↑ ↓)")},
		Cle{Nom: "pr.status", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"pr"}, Aide: i18n.M("état de la revue (approuvée, en revue…)")},
		Cle{Nom: "pr.display", Genre: Choix, Valeurs: []string{"text", "icon"}, Defaut: "text", Items: []string{"pr"}, Aide: i18n.M("état de la revue en mots, ou en icône (icon.pr.approved, .changes, .review, .draft)")},
		Cle{Nom: "pr.link", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"pr"}, Aide: i18n.M("numéro cliquable (lien OSC 8)")},
		Cle{Nom: "session.max", Genre: Nombre, Min: 8, Max: 120, Defaut: "40", Items: []string{"session"}, Aide: i18n.M("longueur du nom de session avant coupure")},
		Cle{Nom: "ctx.bar", Genre: Barre, Valeurs: []string{"lines", "blocks", "dots", "pie", "none"}, Defaut: "lines", Items: []string{"ctx"}, Aide: i18n.M("barre du contexte : lines, blocks, dots, pie (un camembert), none, ou 3 caractères (plein, moitié, vide)")},
		Cle{Nom: "ctx.width", Genre: Nombre, Min: 4, Max: 20, Defaut: "8", Items: []string{"ctx"}, Aide: i18n.M("cases de la barre du contexte")},
		Cle{Nom: "ctx.alert", Genre: Nombre, Min: 1, Max: 100, Defaut: "85", Items: []string{"ctx"}, Aide: i18n.M("seuil d'alerte du contexte, en %"), SSH: true},
		Cle{Nom: "ctx.compact", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"ctx"}, Aide: i18n.M("contexte en alerte : rappeler /compact (Claude Code)")},
		Cle{Nom: "ctx.label", Genre: Libelle, Defaut: "ctx", Items: []string{"ctx"}, Aide: i18n.M("libellé du contexte"), SSH: true},
		Cle{Nom: "quota.alert", Genre: Nombre, Min: 1, Max: 100, Defaut: "90", Items: quotas, Aide: i18n.M("seuil d'alerte des quotas, en %"), SSH: true},
		Cle{Nom: "quota.projection", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: quotas, Aide: i18n.M("projection à la fin de la fenêtre (↗)")},
		Cle{Nom: "quota.limit", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: quotas, Aide: i18n.M("heure prévue du plafond, en alerte")},
		Cle{Nom: "quota.gauge", Genre: Choix, Valeurs: []string{"none", "battery", "pie"}, Defaut: "none", Items: quotas, Aide: i18n.M("jauge devant chaque quota : aucune, batterie qui se vide (Nerd Font), camembert")},
		Cle{Nom: "quota.reset", Genre: Choix, Valeurs: []string{"alert", "always", "never"}, Defaut: "alert", Items: quotas, Aide: i18n.M("heure de remise à zéro : en alerte, toujours, jamais")},
		Cle{Nom: "5h.label", Genre: Libelle, Defaut: "5h", Items: []string{"5h"}, Aide: i18n.M("libellé du quota de 5 heures"), SSH: true},
		Cle{Nom: "7d.label", Genre: Libelle, Defaut: Auto, Items: []string{"7d"}, Aide: i18n.M("libellé du quota de 7 jours (auto : 7j en français, 7d en anglais)"), SSH: true},
		Cle{Nom: "labels", Genre: Choix, Valeurs: []string{"text", "icons"}, Defaut: "text", Items: []string{"ctx", "5h", "7d", "cost"}, Aide: i18n.M("libellés du contexte, des quotas et du coût : texte, ou icônes (icon.ctx, icon.5h, icon.7d, icon.cost)")},
		Cle{Nom: "cost.when", Genre: Choix, Valeurs: []string{"api", "always"}, Defaut: "api", Items: []string{"cost"}, Aide: i18n.M("coût : sans abonnement seulement (api), ou toujours (always)")},
		Cle{Nom: "cost.format", Genre: Choix, Valeurs: []string{Auto, "fr", "en"}, Defaut: Auto, Items: []string{"cost"}, Aide: i18n.M("4,98 $ (fr) ou $4.98 (en) ; auto : selon la langue")},
	)
}()

// libellesAuto : ce que vaut « auto » pour les libellés qui suivent la langue.
var libellesAuto = map[string]func() string{"7d.label": Libelle7j}

// Libelle7j : le libellé du quota de 7 jours dans la langue des textes (7j, 7d).
func Libelle7j() string { return i18n.T("7j") }

// LibelleEnVigueur : la valeur v de l'option nom, « auto » remplacé par le libellé de la langue.
func LibelleEnVigueur(nom, v string) string {
	if f, ok := libellesAuto[NomCle(nom)]; ok && v == Auto {
		return f()
	}
	return v
}

// CoutAnglais : le coût s'écrit-il à l'anglaise ($4.98) pour cette valeur de cost.format ?
func CoutAnglais(v string) bool { return v == "en" || v == Auto && !i18n.Francais() }

// Explication : l'aide de l'option, dans la langue des textes.
func (c Cle) Explication() string {
	if c.aideArg != "" {
		return i18n.T(c.Aide, c.aideArg)
	}
	return i18n.T(c.Aide)
}

var index = func() map[string]int {
	m := map[string]int{}
	for k, c := range catalogue {
		m[c.Nom] = k
	}
	return m
}()

// Cles : toutes les options, dans l'ordre de « agentline style keys » (une copie).
func Cles() []Cle { return append([]Cle(nil), catalogue...) }

// Trouver : l'option de ce nom (ou de son ancien nom français).
func Trouver(nom string) (Cle, bool) {
	k, ok := index[NomCle(nom)]
	if !ok {
		return Cle{}, false
	}
	return catalogue[k], true
}

// DesItems : les options qui règlent l'item (vide : les options générales), dans l'ordre du catalogue,
// ses icônes en dernier.
func DesItems(item string) []Cle {
	if n, ok := NomItem(item); ok {
		item = n
	}
	var l, icones []Cle
	for _, c := range desItems(item) {
		if strings.HasPrefix(c.Nom, "icon.") && item != "" {
			icones = append(icones, c)
		} else {
			l = append(l, c)
		}
	}
	return append(l, icones...)
}

func desItems(item string) []Cle {
	var l []Cle
	for _, c := range catalogue {
		if c.Genre == Disposition || c.Genre == ListeItems {
			continue
		}
		if item == "" && len(c.Items) == 0 {
			l = append(l, c)
		}
		for _, i := range c.Items {
			if i == item {
				l = append(l, c)
			}
		}
	}
	return l
}

// EstItem : le nom est-il celui d'un item ?
func EstItem(n string) bool {
	for _, i := range Items {
		if i == n {
			return true
		}
	}
	return false
}

// Limites des textes libres, en colonnes.
const (
	maxLibelle  = 12
	maxGlyphe   = 4
	maxBranches = 80
)

// cellules : la mesure du sélecteur (caractères ambigus sur une colonne, quelle que soit la langue).
var cellules = &runewidth.Condition{EastAsianWidth: false, StrictEmojiNeutral: true}

// Erreur : une valeur refusée, et pourquoi (message destiné à l'utilisateur).
type Erreur string

func (e Erreur) Error() string { return string(e) }

// etroit : chaque caractère occupe exactement une colonne (ni CJK, ni emoji, ni contrôle, ni combinant).
func etroit(v string) error {
	for _, r := range v {
		if unicode.IsControl(r) || cellules.RuneWidth(r) != 1 && !(r >= 0xe000 && r <= 0xf8ff || r >= 0xf0000) {
			return Erreur(i18n.T("« %s » occupe %d colonnes : un caractère d'une colonne seulement", string(r), cellules.RuneWidth(r)))
		}
	}
	return nil
}

// Valider : nil si la valeur convient à l'option, sinon pourquoi elle est refusée.
func (c Cle) Valider(v string) error {
	if c.Genre != Disposition && c.Genre != ListeItems && v != strings.TrimSpace(v) {
		return Erreur(i18n.T("pas d'espace au début ni à la fin"))
	}
	switch c.Genre {
	case Choix:
		for _, x := range c.Valeurs {
			if v == x {
				return nil
			}
		}
		return Erreur(i18n.T("valeurs permises : %s", strings.Join(c.Valeurs, ", ")))
	case Nombre:
		n, err := strconv.Atoi(v)
		if err != nil || n < c.Min || n > c.Max {
			return Erreur(i18n.T("un nombre de %d à %d", c.Min, c.Max))
		}
		return nil
	case Libelle:
		if len([]rune(v)) > maxLibelle {
			return Erreur(i18n.T("%d caractères au plus", maxLibelle))
		}
		return etroit(v)
	case Glyphe:
		if len([]rune(v)) > maxGlyphe {
			return Erreur(i18n.T("%d caractères au plus", maxGlyphe))
		}
		return etroit(v)
	case Barre:
		if _, ok := barres[v]; ok {
			return nil
		}
		if len([]rune(v)) != 3 {
			return Erreur(i18n.T("%s, ou exactement 3 caractères (plein, moitié, vide)", strings.Join(c.Valeurs, ", ")))
		}
		return etroit(v)
	case Disposition:
		_, err := LireDisposition(v)
		return err
	case Branches:
		if len([]rune(v)) > maxBranches {
			return Erreur(i18n.T("%d caractères au plus", maxBranches))
		}
		for _, n := range strings.Fields(v) {
			if strings.ContainsAny(n, "~^:?*[\\") || strings.ContainsFunc(n, unicode.IsControl) {
				return Erreur(i18n.T("nom de branche invalide : %s", n))
			}
		}
		return nil
	case ListeItems:
		for _, n := range strings.Fields(v) {
			if _, ok := NomItem(n); !ok {
				return itemInconnu(n)
			}
		}
		return nil
	}
	return Erreur(i18n.T("option inconnue"))
}

func itemInconnu(n string) error {
	return Erreur(i18n.T("item inconnu : %s (items : %s)", n, strings.Join(Items, ", ")))
}

// LireDisposition : « a b | c » -> les segments {a b} et {c} ; un item inconnu est une erreur, un ancien
// nom (français) est lu sous son nom actuel. Les segments vides sont ignorés.
func LireDisposition(texte string) ([][]string, error) {
	var segs [][]string
	for _, seg := range strings.Split(texte, "|") {
		noms := strings.Fields(seg)
		for k, n := range noms {
			i, ok := NomItem(n)
			if !ok {
				return nil, itemInconnu(n)
			}
			noms[k] = i
		}
		if len(noms) > 0 {
			segs = append(segs, noms)
		}
	}
	return segs, nil
}

// EcrireDisposition : l'inverse de LireDisposition (« a b | c »).
func EcrireDisposition(segs [][]string) string {
	var t []string
	for _, s := range segs {
		if len(s) > 0 {
			t = append(t, strings.Join(s, " "))
		}
	}
	return strings.Join(t, " | ")
}
