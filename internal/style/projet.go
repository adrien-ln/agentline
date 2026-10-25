package style

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Le style d'un projet : les seules options réglées pour lui, par-dessus le style général, dans
// $AGENTLINE_CONFIG/styles/<projet> (le nom tiré de l'identifiant du projet, que la première ligne rappelle
// pour qu'un nom ressemblant ne serve pas à un autre projet).

// DossierProjets : le dossier des styles de projet, dans le dossier des réglages.
const DossierProjets = "styles"

// Surcharge : les options réglées pour un projet. Une valeur égale au défaut compte : elle annule le
// réglage général de cette option.
type Surcharge struct {
	v map[string]string
}

// Definir : règle l'option pour le projet, après validation (même au défaut).
func (s *Surcharge) Definir(nom, v string) error {
	c, ok := Trouver(nom)
	if !ok {
		return Erreur(i18n.T("option inconnue : %s", nom))
	}
	v = valeurActuelle(c, v)
	if err := c.Valider(v); err != nil {
		return err
	}
	if c.Genre == Disposition || c.Genre == ListeItems {
		v = normaliser(c, v)
	}
	if s.v == nil {
		s.v = map[string]string{}
	}
	s.v[c.Nom] = v
	return nil
}

// Retirer : l'option reprend la valeur du style général.
func (s *Surcharge) Retirer(nom string) {
	if s != nil {
		delete(s.v, NomCle(nom))
	}
}

// Regle : l'option est-elle réglée pour le projet ?
func (s *Surcharge) Regle(nom string) bool {
	if s == nil {
		return false
	}
	_, ok := s.v[NomCle(nom)]
	return ok
}

// Nombre : combien d'options sont réglées pour le projet.
func (s *Surcharge) Nombre() int {
	if s == nil {
		return 0
	}
	return len(s.v)
}

// Superposer : le style général, puis les options du projet par-dessus.
func Superposer(g *Style, s *Surcharge) *Style {
	r := g.Copie()
	if s != nil {
		for _, c := range catalogue {
			if v, ok := s.v[c.Nom]; ok {
				r.Definir(c.Nom, v) // valide : la surcharge ne garde que des valeurs valides
			}
		}
	}
	return r
}

// Ecart : la surcharge qui, posée sur le style général, donne le style effectif (les options qui diffèrent).
func Ecart(effectif, general *Style) *Surcharge {
	s := &Surcharge{}
	for _, c := range catalogue {
		if v := effectif.Valeur(c.Nom); v != general.Valeur(c.Nom) {
			if s.v == nil {
				s.v = map[string]string{}
			}
			s.v[c.Nom] = v
		}
	}
	return s
}

func cheminProjet(dossier, id string) string {
	return filepath.Join(dossier, DossierProjets, theme.NomDeTitre(id))
}

// ChargerProjet : le style du projet id (nil s'il n'en a pas) ; ce qui est invalide est ignoré.
func ChargerProjet(dossier, id string) *Surcharge {
	if id == "" {
		return nil
	}
	b, err := os.ReadFile(cheminProjet(dossier, id))
	if err != nil {
		return nil
	}
	lignes := strings.Split(string(b), "\n")
	if strings.TrimSpace(strings.TrimPrefix(lignes[0], "#")) != id {
		return nil // un autre projet au nom ressemblant
	}
	s := &Surcharge{}
	for _, l := range lignes[1:] {
		t := strings.TrimSpace(l)
		if t == "" || t[0] == '#' {
			continue
		}
		if cle, val, ok := strings.Cut(t, "="); ok {
			s.Definir(strings.TrimSpace(cle), strings.TrimSpace(val))
		}
	}
	return s
}

// EcrireProjet : le style du projet id (écriture atomique) ; sans option, le fichier est retiré.
func EcrireProjet(dossier, id string, s *Surcharge) error {
	f := cheminProjet(dossier, id)
	if s.Nombre() == 0 {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	var b strings.Builder
	b.WriteString("# " + id + "\n")
	for _, c := range catalogue {
		if v, ok := s.v[c.Nom]; ok {
			b.WriteString(c.Nom + "=" + v + "\n")
		}
	}
	return theme.EcrireAtomique(filepath.Dir(f), filepath.Base(f), b.String())
}

// ChargerPour : le style en vigueur dans le projet id : le général et celui du projet ; le défaut si
// AGENTLINE_STYLE=default.
func ChargerPour(dossier, id string, getenv func(string) string) *Style {
	if Force(getenv) {
		return Defaut()
	}
	return Superposer(LireFichier(dossier), ChargerProjet(dossier, id))
}
