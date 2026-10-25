package theme

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// tagsFrancais : tag français -> tag anglais, pour écrire les thèmes avec les noms documentés.
var tagsFrancais = func() map[string]string {
	m := map[string]string{}
	for en, fr := range tagsAnglais {
		if en != "grey" {
			m[fr] = en
		}
	}
	return m
}()

// Format : la palette en fichier de thème, avec les noms anglais documentés (docs/themes.md) : le titre, les
// tags, le fond, les couleurs, puis celles de l'autre fond quand elles sont réglées. Lire et Entete
// redonnent la palette, le titre et les tags.
func Format(p Palette, titre string, tags []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", titre)
	if len(tags) > 0 {
		en := make([]string, len(tags))
		for k, t := range tags {
			en[k] = t
			if a, ok := tagsFrancais[t]; ok {
				en[k] = a
			}
		}
		fmt.Fprintf(&b, "# tags: %s\n", strings.Join(en, ", "))
	}
	if p.Clair {
		b.WriteString("# background: light\n")
	}
	ecrireCouleurs(&b, p.Couleurs, "")
	suffixe := "_LIGHT"
	if p.Clair {
		suffixe = "_DARK"
	}
	ecrireCouleurs(&b, p.Autre, suffixe)
	return []byte(b.String())
}

// ecrireCouleurs : les couleurs réglées, par lignes de rôles voisins ; l'intensité seulement si ses cinq
// niveaux le sont (Lire les prend dans l'ordre).
func ecrireCouleurs(b *strings.Builder, c Couleurs, suffixe string) {
	for _, ligne := range [][][2]string{
		{{"TEXT", c.Texte}, {"MUTED", c.Discret}, {"SEP", c.Sep}, {"DIM", c.Eteint}},
		{{"MODEL", c.Modele}, {"MODEL_END", c.ModeleFin}, {"FOLDER", c.Dossier}, {"BRANCH", c.Lieu}},
		{{"D0", c.D0}, {"D50", c.D50}, {"D75", c.D75}, {"D100", c.D100}},
	} {
		var l []string
		for _, x := range ligne {
			if x[1] != "" {
				l = append(l, x[0]+suffixe+"="+x[1])
			}
		}
		if len(l) > 0 {
			b.WriteString(strings.Join(l, " ") + "\n")
		}
	}
	for _, x := range c.Intensite {
		if x == "" {
			return
		}
	}
	fmt.Fprintf(b, "INTENSITY%s=(%s)\n", suffixe, strings.Join(c.Intensite[:], " "))
}

// ErrNomTheme : un nom de thème doit être en minuscules, chiffres et tirets (comme les fichiers).
var ErrNomTheme = errors.New("nom de thème invalide")

// EcrireTheme : le thème <nom> de l'utilisateur ($AGENTLINE_CONFIG/themes/<nom>.theme), écrit d'un coup.
func (s Sources) EcrireTheme(nom string, contenu []byte) error {
	if !nomValide.MatchString(nom) {
		return ErrNomTheme
	}
	return EcrireAtomique(filepath.Join(s.Config, "themes"), nom+".theme", string(contenu))
}

// DeLUtilisateur : le thème <nom> est-il un fichier de l'utilisateur (sinon intégré, ou absent) ?
func (s Sources) DeLUtilisateur(nom string) bool {
	if !nomValide.MatchString(nom) {
		return false
	}
	_, err := os.Stat(filepath.Join(s.Config, "themes", nom+".theme"))
	return err == nil
}

// NomValide : le nom peut-il être celui d'un thème (minuscules, chiffres et tirets) ?
func NomValide(nom string) bool { return nomValide.MatchString(nom) }
