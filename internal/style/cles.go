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
	}
	glyphesTexte = map[string]string{
		"folder": "▸", "branch": "⎇", "worktree": "⊕", "pr": "PR", "alert": "!",
		"reset": "↻", "fast": "»", "effort.low": "▿", "effort.high": "▴", "signal": "●",
	}
	// nomsGlyphes : les icônes réglables une à une (icon.<nom>), dans l'ordre de « agentline style keys ».
	nomsGlyphes = []string{"folder", "branch", "worktree", "pr", "alert", "reset", "fast", "effort.low", "effort.high", "signal"}
	// itemsGlyphes : l'item de chaque icône (vide : générale).
	itemsGlyphes = map[string][]string{
		"folder": {"folder"}, "branch": {"git"}, "worktree": {"worktree"}, "pr": {"pr"},
		"alert": {"5h", "7d"}, "reset": {"5h", "7d"}, "fast": {"fast"}, "effort.low": {"effort"}, "effort.high": {"effort"},
	}
	// Barres nommées de ctx.bar : pleine, à moitié, éteinte.
	barres = map[string]string{"lines": "━╸─", "blocks": "█▌░", "dots": "●◐○", "none": ""}
)

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
		c = append(c, Cle{Nom: "icon." + n, Genre: Glyphe, Defaut: Auto, Items: itemsGlyphes[n],
			Aide: i18n.M("icône %s (auto : selon icons ; vide : aucune)"), aideArg: strings.ReplaceAll(n, ".", " ")})
	}
	return append(c,
		Cle{Nom: "model.name", Genre: Choix, Valeurs: []string{"full", "short"}, Defaut: "full", Items: []string{"model"}, Aide: i18n.M("nom du modèle en entier, ou son premier mot")},
		Cle{Nom: "model.color", Genre: Choix, Valeurs: []string{"gradient", "solid"}, Defaut: "gradient", Items: []string{"model"}, Aide: i18n.M("nom du modèle en dégradé, ou d'une couleur")},
		Cle{Nom: "effort.display", Genre: Choix, Valeurs: []string{"icon+text", "icon", "text"}, Defaut: "icon+text", Items: []string{"effort"}, Aide: i18n.M("niveau d'effort : jauge et nom, jauge seule, nom seul")},
		Cle{Nom: "git.status", Genre: Choix, Valeurs: []string{"detail", "summary"}, Defaut: "detail", Items: []string{"git"}, Aide: i18n.M("compteurs (+1 ●2 ?3), ou un seul signe (✓ ou *)")},
		Cle{Nom: "git.upstream", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"git"}, Aide: i18n.M("commits d'avance et de retard (↑ ↓)")},
		Cle{Nom: "pr.status", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"pr"}, Aide: i18n.M("état de la revue (approuvée, en revue…)")},
		Cle{Nom: "pr.link", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: []string{"pr"}, Aide: i18n.M("numéro cliquable (lien OSC 8)")},
		Cle{Nom: "session.max", Genre: Nombre, Min: 8, Max: 120, Defaut: "40", Items: []string{"session"}, Aide: i18n.M("longueur du nom de session avant coupure")},
		Cle{Nom: "ctx.bar", Genre: Barre, Valeurs: []string{"lines", "blocks", "dots", "none"}, Defaut: "lines", Items: []string{"ctx"}, Aide: i18n.M("barre du contexte : lines, blocks, dots, none, ou 3 caractères (plein, moitié, vide)")},
		Cle{Nom: "ctx.width", Genre: Nombre, Min: 4, Max: 20, Defaut: "8", Items: []string{"ctx"}, Aide: i18n.M("cases de la barre du contexte")},
		Cle{Nom: "ctx.alert", Genre: Nombre, Min: 1, Max: 100, Defaut: "85", Items: []string{"ctx"}, Aide: i18n.M("seuil d'alerte du contexte, en %"), SSH: true},
		Cle{Nom: "ctx.label", Genre: Libelle, Defaut: "ctx", Items: []string{"ctx"}, Aide: i18n.M("libellé du contexte"), SSH: true},
		Cle{Nom: "quota.alert", Genre: Nombre, Min: 1, Max: 100, Defaut: "90", Items: quotas, Aide: i18n.M("seuil d'alerte des quotas, en %"), SSH: true},
		Cle{Nom: "quota.projection", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: quotas, Aide: i18n.M("projection à la fin de la fenêtre (↗)")},
		Cle{Nom: "quota.limit", Genre: Choix, Valeurs: ouiNon, Defaut: Oui, Items: quotas, Aide: i18n.M("heure prévue du plafond, en alerte")},
		Cle{Nom: "quota.reset", Genre: Choix, Valeurs: []string{"alert", "always", "never"}, Defaut: "alert", Items: quotas, Aide: i18n.M("heure de remise à zéro : en alerte, toujours, jamais")},
		Cle{Nom: "5h.label", Genre: Libelle, Defaut: "5h", Items: []string{"5h"}, Aide: i18n.M("libellé du quota de 5 heures"), SSH: true},
		Cle{Nom: "7d.label", Genre: Libelle, Defaut: Auto, Items: []string{"7d"}, Aide: i18n.M("libellé du quota de 7 jours (auto : 7j en français, 7d en anglais)"), SSH: true},
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
	maxLibelle = 12
	maxGlyphe  = 4
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
