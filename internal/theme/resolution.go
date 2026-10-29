package theme

import (
	"os"
	"path/filepath"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Les sources du thème, de la plus prioritaire à la moins prioritaire.
const (
	SourceForce   = "forcé"             // la variable AGENTLINE_THEME
	SourceBranche = "branche"           // une règle de $AGENTLINE_CONFIG/branches
	SourceProjet  = "projet"            // une association de $AGENTLINE_CONFIG/projets
	SourceDefaut  = "défaut"            // $AGENTLINE_CONFIG/defaut
	SourcePalette = "AGENTLINE_PALETTE" // l'ancien réglage, par variable
	SourceIntegre = "intégré"           // tokyo
)

// Ordre : les sources, de la plus prioritaire à la moins prioritaire.
var Ordre = []string{SourceForce, SourceBranche, SourceProjet, SourceDefaut, SourcePalette, SourceIntegre}

// Etape : un maillon de la chaîne de priorité, avec ce qui y est réglé.
type Etape struct {
	Source  string  // une des sources de Ordre
	Cle     string  // la branche courante, ou l'identifiant du projet
	Motif   string  // branche : la règle qui correspond (le nom exact ou un motif)
	Nom     string  // le thème réglé à cette étape ; vide : rien
	Existe  bool    // ce thème existe (sinon l'étape est ignorée)
	Active  bool    // c'est l'étape qui l'emporte
	Palette Palette // sa palette, si le thème existe
}

// Libelle : l'étape pour un humain (« branche main (règle *) », « projet gitlab.com/x/y »).
func (e Etape) Libelle() string {
	switch e.Source {
	case SourceForce:
		return "AGENTLINE_THEME"
	case SourceBranche:
		l := i18n.T("branche %s", e.Cle)
		if e.Cle == "" {
			l = i18n.T("branche (aucune)")
		}
		if e.Motif != "" && e.Motif != e.Cle {
			l += i18n.T(" (règle %s)", e.Motif)
		}
		return l
	case SourceProjet:
		return i18n.T("projet %s", e.Cle)
	}
	return NomSource(e.Source)
}

// NomSource : la source pour un humain, dans la langue des textes (les constantes Source* restent les
// identifiants internes).
func NomSource(source string) string {
	switch source {
	case SourceForce:
		return i18n.T("forcé")
	case SourceBranche:
		return i18n.T("branche")
	case SourceProjet:
		return i18n.T("projet")
	case SourceDefaut:
		return i18n.T("défaut")
	case SourceIntegre:
		return i18n.T("intégré")
	}
	return source
}

// Rang : la place de la source dans Ordre (0 : la plus prioritaire).
func Rang(source string) int {
	for i, s := range Ordre {
		if s == source {
			return i
		}
	}
	return len(Ordre)
}

// Active : l'étape qui l'emporte.
func Active(etapes []Etape) Etape {
	for _, e := range etapes {
		if e.Active {
			return e
		}
	}
	return Etape{Source: SourceIntegre, Nom: "tokyo", Existe: true, Active: true, Palette: Secours()}
}

// MasquePar : l'étape plus prioritaire que la source qui l'emporterait sur un thème réglé à cette source
// (faux si un thème réglé là serait bien celui en vigueur).
func MasquePar(etapes []Etape, source string) (Etape, bool) {
	r := Rang(source)
	for _, e := range etapes {
		if Rang(e.Source) < r && e.Nom != "" && e.Existe {
			return e, true
		}
	}
	return Etape{}, false
}

// Resoudre : toute la chaîne de priorité pour le dossier (vide : pas de projet) sur la branche donnée, de
// AGENTLINE_THEME à tokyo : une étape par source, ce qui y est réglé, celle qui l'emporte. Choisir n'en parcourt
// que le début.
func (s Sources) Resoudre(dossier, branche string, getenv func(string) string) []Etape {
	return s.resoudre(dossier, func() string { return branche }, getenv, true)
}

// resoudre : la chaîne de priorité ; tout faux : arrêtée à la première étape valable, sans lire les
// réglages inutiles (pas de fichier branches : la branche n'est pas lue).
func (s Sources) resoudre(dossier string, branche func() string, getenv func(string) string, tout bool) []Etape {
	var etapes []Etape
	trouve := false
	ajouter := func(e Etape) bool { // vrai : on peut s'arrêter là
		if e.Nom != "" {
			e.Nom = s.Remplacant(e.Nom)
			e.Palette, e.Existe = s.Charger(e.Nom)
		}
		if e.Existe && !trouve {
			e.Active, trouve = true, true
		}
		etapes = append(etapes, e)
		return trouve && !tout
	}
	existe := func(f string) bool { _, err := os.Stat(filepath.Join(s.Config, f)); return err == nil }

	if ajouter(Etape{Source: SourceForce, Nom: getenv("AGENTLINE_THEME")}) {
		return etapes
	}
	e := Etape{Source: SourceBranche}
	if existe("branches") {
		e.Cle = branche()
		e.Motif, e.Nom = s.RegleDeBranche(e.Cle)
	} else if tout {
		e.Cle = branche()
	}
	if ajouter(e) {
		return etapes
	}
	e = Etape{Source: SourceProjet}
	if dossier != "" && (tout || existe("projets")) {
		e.Cle = s.ProjetID(dossier)
		e.Nom = s.ThemeDuProjet(e.Cle)
	}
	if ajouter(e) {
		return etapes
	}
	if ajouter(Etape{Source: SourceDefaut, Nom: premiereLigne(filepath.Join(s.Config, "defaut"))}) {
		return etapes
	}
	if ajouter(Etape{Source: SourcePalette, Nom: getenv("AGENTLINE_PALETTE")}) {
		return etapes
	}
	e = Etape{Source: SourceIntegre, Nom: "tokyo"}
	e.Palette, e.Existe = s.Charger(e.Nom)
	if !e.Existe { // même sans fichier, la barre ne reste jamais vide
		e.Palette, e.Existe = Secours(), true
	}
	if !trouve {
		e.Active = true
	}
	return append(etapes, e)
}
