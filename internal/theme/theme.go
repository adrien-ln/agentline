// Package theme lit les fichiers *.theme et choisit le thème en vigueur, comme l'ancienne version bash :
// AGENTLINE_THEME (forcé) > thème de la branche > thème du projet > thème par défaut > AGENTLINE_PALETTE (ancien réglage) > tokyo.
package theme

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Couleurs : les couleurs d'un thème, en hexadécimal sans « # ».
type Couleurs struct {
	Texte, Discret, Sep, Eteint      string
	Modele, ModeleFin, Dossier, Lieu string
	D0, D50, D75, D100               string
	Intensite                        [5]string
}

// Palette : les couleurs d'un thème, le fond pour lequel il est fait, et celles qu'il donne pour l'autre
// fond (vides : calculées par PourFond).
type Palette struct {
	Couleurs
	Clair bool     // « # fond: clair » : le thème est fait pour un fond clair
	Autre Couleurs // TEXTE_CLAIR=… (thème sombre) ou TEXTE_SOMBRE=… (thème clair)
}

// FinDegrade : fin du dégradé du nom du modèle (MODELE_FIN, ou LIEU si absente).
func (p Palette) FinDegrade() string {
	if p.ModeleFin != "" {
		return p.ModeleFin
	}
	return p.Lieu
}

var (
	nomValide   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	affectation = regexp.MustCompile(`([A-Z0-9_]+)=([0-9a-f]{6})`)
	intensite   = regexp.MustCompile(`^(?:INTENSITY|INTENSITE)(_LIGHT|_DARK|_CLAIR|_SOMBRE)?=\(([^)]*)\)`)
)

// champ : le champ d'un nom de couleur des fichiers de thème (nil : nom inconnu). Chaque couleur a un nom
// anglais (documenté) et son nom français d'origine, celui des thèmes intégrés : TEXT/TEXTE,
// MUTED/DISCRET, SEP, DIM/ETEINT, MODEL/MODELE, MODEL_END/MODELE_FIN, FOLDER/DOSSIER, BRANCH/LIEU, D0…D100.
func (c *Couleurs) champ(nom string) *string {
	switch nom {
	case "TEXT", "TEXTE":
		return &c.Texte
	case "MUTED", "DISCRET":
		return &c.Discret
	case "SEP":
		return &c.Sep
	case "DIM", "ETEINT":
		return &c.Eteint
	case "MODEL", "MODELE":
		return &c.Modele
	case "MODEL_END", "MODELE_FIN":
		return &c.ModeleFin
	case "FOLDER", "DOSSIER":
		return &c.Dossier
	case "BRANCH", "LIEU":
		return &c.Lieu
	case "D0":
		return &c.D0
	case "D50":
		return &c.D50
	case "D75":
		return &c.D75
	case "D100":
		return &c.D100
	}
	return nil
}

// FondClair : la ligne d'en-tête déclare-t-elle un thème fait pour un fond clair (« # background: light »,
// ou « # fond: clair », le nom d'origine) ?
func FondClair(ligne string) bool {
	if v, ok := strings.CutPrefix(ligne, "# background:"); ok {
		return strings.TrimSpace(v) == "light"
	}
	v, ok := strings.CutPrefix(ligne, "# fond:")
	return ok && strings.TrimSpace(v) == "clair"
}

// Lire : le contenu d'un fichier de thème -> sa palette. « # background: light » (« # fond: clair »)
// déclare un thème fait pour un fond clair ; une couleur suffixée _LIGHT/_CLAIR (thème sombre) ou
// _DARK/_SOMBRE (thème clair) la remplace sur l'autre fond, à la place de la correction automatique (les
// autres suffixes sont ignorés).
func Lire(contenu []byte) Palette {
	var p Palette
	var clair, sombre Couleurs
	ens := func(suffixe string) *Couleurs {
		switch suffixe {
		case "_LIGHT", "_CLAIR":
			return &clair
		case "_DARK", "_SOMBRE":
			return &sombre
		}
		return &p.Couleurs
	}
	for _, l := range strings.Split(string(contenu), "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.HasPrefix(l, "#") {
			if FondClair(l) {
				p.Clair = true
			}
			continue
		}
		if m := intensite.FindStringSubmatch(l); m != nil {
			copy(ens(m[1]).Intensite[:], strings.Fields(m[2]))
			continue
		}
		for _, m := range affectation.FindAllStringSubmatch(l, -1) {
			nom, suffixe := m[1], ""
			for _, s := range []string{"_LIGHT", "_DARK", "_CLAIR", "_SOMBRE"} {
				if n, ok := strings.CutSuffix(nom, s); ok {
					nom, suffixe = n, s
				}
			}
			if c := ens(suffixe).champ(nom); c != nil {
				*c = m[2]
			}
		}
	}
	p.Autre = clair
	if p.Clair {
		p.Autre = sombre
	}
	return p
}

// Secours : Tokyo Night, pour que la barre ne reste jamais vide.
func Secours() Palette {
	return Palette{Couleurs: Couleurs{
		Texte: "c0caf5", Discret: "737aa2", Sep: "3b4261", Eteint: "545c7e",
		Modele: "bb9af7", Dossier: "7aa2f7", Lieu: "7dcfff",
		D0: "9ece6a", D50: "e0af68", D75: "ff9e64", D100: "f7768e",
		Intensite: [5]string{"7dcfff", "7aa2f7", "bb9af7", "ff007c", "ff9e64"},
	}}
}

// Sources : où chercher les thèmes et les réglages.
type Sources struct {
	Config  string // $AGENTLINE_CONFIG : defaut, projets, themes/ de l'utilisateur
	Integre fs.FS  // thèmes de l'installation, sous « themes/ »
	Windows bool   // chemins insensibles à la casse, « /c/x » = « c:/x »
}

// Fichier : le contenu du thème <nom> (celui de l'utilisateur d'abord) ; faux si le nom est invalide ou absent.
func (s Sources) Fichier(nom string) ([]byte, bool) {
	if !nomValide.MatchString(nom) {
		return nil, false
	}
	if b, err := os.ReadFile(filepath.Join(s.Config, "themes", nom+".theme")); err == nil {
		return b, true
	}
	if s.Integre != nil {
		if b, err := fs.ReadFile(s.Integre, "themes/"+nom+".theme"); err == nil {
			return b, true
		}
		if r, ok := Remplaces[nom]; ok {
			if b, err := fs.ReadFile(s.Integre, "themes/"+r+".theme"); err == nil {
				return b, true
			}
		}
	}
	return nil, false
}

// Remplaces : les thèmes intégrés retirés (0.8.0) et celui qui les remplace. Un réglage qui en nomme un
// (défaut, projet, branche, AGENTLINE_THEME) prend son remplaçant, sauf thème de l'utilisateur du même nom.
var Remplaces = map[string]string{
	"lave": "braise", "mangue": "tropique", "peche": "coucher-de-soleil", "guimauve": "barbe-a-papa",
	"pastel": "barbe-a-papa", "instagram": "coucher-de-soleil", "violet-electrique": "ultraviolet",
	"plasma": "ultraviolet", "laser": "neon", "eclair": "glitch", "lagune": "ocean-profond", "miami": "vaporwave",
	"emeraude": "foret", "clairiere": "foret", "prairie": "foret", "mojito": "foret", "pasteque": "tropique",
}

// Remplacant : le nom à utiliser pour <nom> : son remplaçant s'il désigne un thème intégré retiré (et pas un
// thème de l'utilisateur), sinon lui-même.
func (s Sources) Remplacant(nom string) string {
	if r, ok := Remplaces[nom]; ok && !s.DeLUtilisateur(nom) {
		return r
	}
	return nom
}

// Charger : la palette du thème <nom> ; faux s'il n'existe pas.
func (s Sources) Charger(nom string) (Palette, bool) {
	b, ok := s.Fichier(nom)
	if !ok {
		return Palette{}, false
	}
	return Lire(b), true
}

// Choix : le thème retenu, d'où il vient, et sa palette.
type Choix struct {
	Nom, Source string
	Palette     Palette
}

// Choisir : le thème en vigueur pour le dossier du projet (vide : pas de projet), sur sa branche courante.
func (s Sources) Choisir(dossier string, getenv func(string) string) Choix {
	return s.choisir(dossier, func() string { return BrancheDe(dossier) }, getenv)
}

// ChoisirSurBranche : comme Choisir, quand la branche est déjà connue (le rendu l'a lue avec l'état Git).
func (s Sources) ChoisirSurBranche(dossier, branche string, getenv func(string) string) Choix {
	return s.choisir(dossier, func() string { return branche }, getenv)
}

// choisir : la chaîne de priorité parcourue jusqu'à la première étape valable seulement (le rendu ne lit
// rien de plus que nécessaire).
func (s Sources) choisir(dossier string, branche func() string, getenv func(string) string) Choix {
	e := Active(s.resoudre(dossier, branche, getenv, false))
	return Choix{e.Nom, e.Source, e.Palette}
}

// ThemeDuProjet : le thème associé à l'identifiant dans $AGENTLINE_CONFIG/projets (« identifiant<TAB>thème »).
func (s Sources) ThemeDuProjet(id string) string {
	b, err := os.ReadFile(filepath.Join(s.Config, "projets"))
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.Trim(l, "\t") // comme « IFS=$'\t' read -r id nom »
		if l == "" {
			continue
		}
		cle, nom, _ := strings.Cut(l, "\t")
		if cle == id {
			return strings.TrimRight(strings.Trim(nom, "\t"), "\r")
		}
	}
	return ""
}

// ThemeDeBranche : le thème de la règle de $AGENTLINE_CONFIG/branches qui correspond à la branche (voir RegleDeBranche).
func (s Sources) ThemeDeBranche(branche string) string {
	_, nom := s.RegleDeBranche(branche)
	return nom
}

// RegleDeBranche : la règle de $AGENTLINE_CONFIG/branches qui correspond à la branche, motif et thème : le nom
// exact d'abord, sinon le motif le plus long (« release/* », « feat/* », « * »).
func (s Sources) RegleDeBranche(branche string) (motif, nom string) {
	if branche == "" {
		return "", ""
	}
	long := -1
	for _, r := range s.ReglesBranches() {
		if r[0] == branche {
			return r[0], r[1]
		}
		if Correspond(r[0], branche) && len(r[0]) > long {
			motif, nom, long = r[0], r[1], len(r[0])
		}
	}
	return motif, nom
}

// Correspond : la branche est-elle couverte par le motif ? « * » couvre n'importe quelle suite de
// caractères, « / » compris, et « ? » un seul caractère.
func Correspond(motif, branche string) bool {
	// « * » : n'importe quelle suite (barres obliques comprises), « ? » : un caractère ; sans expression
	// régulière à compiler pour chaque règle, à chaque rendu. Retour arrière sur la dernière étoile.
	m, b := []rune(motif), []rune(branche)
	i, j, etoile, reprise := 0, 0, -1, 0
	for j < len(b) {
		switch {
		case i < len(m) && (m[i] == '?' || m[i] == b[j]):
			i++
			j++
		case i < len(m) && m[i] == '*':
			etoile, reprise = i, j
			i++
		case etoile >= 0:
			reprise++
			i, j = etoile+1, reprise
		default:
			return false
		}
	}
	for i < len(m) && m[i] == '*' {
		i++
	}
	return i == len(m)
}

func premiereLigne(f string) string {
	b, err := os.ReadFile(f)
	if err != nil {
		return ""
	}
	l, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimRight(l, "\r")
}
