package theme

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
)

// D'où vient le fond retenu (Sources.Fond), du plus fort au plus faible.
const (
	FondVariable  = "AGENTLINE_BACKGROUND"
	FondReglage   = "réglage"   // $AGENTLINE_CONFIG/fond
	FondCOLORFGBG = "COLORFGBG" // posée par certains terminaux (rxvt, Konsole)
	FondDefaut    = "défaut"
)

// Fond : le fond du terminal et d'où il vient : AGENTLINE_BACKGROUND (dark, light ; auto : détecter, sans
// lire le réglage) > réglage enregistré (fichier fond : dark ou light) > COLORFGBG > sombre. La
// statusline n'a pas de terminal à interroger : la vraie détection (OSC 11) se fait dans le sélecteur et
// à l'installation, qui l'enregistrent.
func (s Sources) Fond(getenv func(string) string) (ansi.Fond, string) {
	v := getenv("AGENTLINE_BACKGROUND")
	f, auto, ok := ansi.LireFond(v) // valeur inconnue : ignorée
	if ok && !auto {
		return f, FondVariable
	}
	if !(ok && auto && v != "") { // AGENTLINE_BACKGROUND=auto : le réglage est ignoré aussi
		if f, ok := s.FondEnregistre(); ok {
			return f, FondReglage
		}
	}
	if f, ok := ansi.FondDeCOLORFGBG(getenv("COLORFGBG")); ok {
		return f, FondCOLORFGBG
	}
	return ansi.Sombre, FondDefaut
}

// FondEnregistre : le fond du réglage ($AGENTLINE_CONFIG/fond) ; faux s'il n'y en a pas (ou illisible).
func (s Sources) FondEnregistre() (ansi.Fond, bool) {
	f, auto, ok := ansi.LireFond(premiereLigne(filepath.Join(s.Config, "fond")))
	return f, ok && !auto
}

// EcrireFond : enregistre le fond (écriture atomique, comme EcrireDefaut) ; auto retire le réglage (la
// détection reprend la main).
func (s Sources) EcrireFond(f ansi.Fond, auto bool) error {
	if auto {
		if err := os.Remove(filepath.Join(s.Config, "fond")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return EcrireAtomique(s.Config, "fond", strings.ToLower(f.String())+"\n")
}

// Un thème est fait pour un fond, sombre (les 46 premiers) ou clair (« # fond: clair »). Sur l'autre fond,
// chaque couleur prend sa variante déclarée (TEXTE_CLAIR=… ou TEXTE_SOMBRE=…), sinon une correction de
// contraste : même teinte, clarté ajustée jusqu'à un contraste WCAG minimal avec le fond de référence
// (ansi.FondClair, ansi.FondSombre). Les séparateurs et les cases éteintes ont aussi un plafond : discrets
// sur fond sombre, ils le restent sur fond clair au lieu d'y devenir aussi visibles que le texte.

// Bornes de contraste (minimum, plafond ; 0 : pas de plafond) de chaque rôle de couleur.
var (
	bornesTexte   = [2]float64{7, 0} // le texte : WCAG AAA, net au-dessus des libellés
	bornesDiscret = [2]float64{3, 0} // libellés, heures
	bornesAccent  = [2]float64{3, 0} // modèle, dossier, branche, dégradés : WCAG AA en gras, éléments d'interface
	bornesSep     = [2]float64{1.4, 2}
	bornesEteint  = [2]float64{1.8, 2.8}
)

// PourFond : la palette pour ce fond (clair ou non) ; telle quelle si le thème est fait pour lui.
func (p Palette) PourFond(clair bool) Palette {
	if p.Clair == clair {
		return p
	}
	ref := ansi.FondSombre
	if clair {
		ref = ansi.FondClair
	}
	q := Palette{Clair: clair, Couleurs: p.Couleurs, Autre: p.Couleurs}
	ajuster := func(c *string, autre string, b [2]float64) {
		switch {
		case autre != "":
			*c = autre
		case *c != "":
			*c = ansi.Ajuster(*c, ref, b[0], b[1])
		}
	}
	a := p.Autre
	ajuster(&q.Texte, a.Texte, bornesTexte)
	ajuster(&q.Discret, a.Discret, bornesDiscret)
	ajuster(&q.Sep, a.Sep, bornesSep)
	ajuster(&q.Eteint, a.Eteint, bornesEteint)
	for _, x := range []struct {
		c     *string
		autre string
	}{{&q.Modele, a.Modele}, {&q.ModeleFin, a.ModeleFin}, {&q.Dossier, a.Dossier}, {&q.Lieu, a.Lieu},
		{&q.D0, a.D0}, {&q.D50, a.D50}, {&q.D75, a.D75}, {&q.D100, a.D100}} {
		ajuster(x.c, x.autre, bornesAccent)
	}
	for i := range q.Intensite {
		ajuster(&q.Intensite[i], a.Intensite[i], bornesAccent)
	}
	return q
}
