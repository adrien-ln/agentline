// Package style : ce que l'utilisateur règle dans la statusline (quels items, dans quel ordre, avec quels
// glyphes, seuils et libellés). Les couleurs restent l'affaire des thèmes (internal/theme).
//
// Le style vit dans $AGENTLINE_CONFIG/style, une ligne « clé=valeur » par option, « # » en tête de ligne pour
// un commentaire ; seules les options qui s'écartent du défaut y sont écrites. La lecture est tolérante :
// une clé inconnue ou une valeur invalide est ignorée (l'option garde son défaut), le rendu ne casse
// jamais. AGENTLINE_STYLE=default fait ignorer le fichier.
package style

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Fichier : le nom du fichier du style, dans le dossier des réglages.
const Fichier = "style"

// Style : les options qui s'écartent du défaut (toutes valides). Le zéro et nil valent le défaut.
type Style struct {
	v map[string]string
}

// Defaut : le style intégré.
func Defaut() *Style { return &Style{} }

// Lire : le style décrit par le contenu d'un fichier ; ce qui est invalide est ignoré.
func Lire(b []byte) *Style {
	s := &Style{}
	for len(b) > 0 {
		var l []byte
		if i := indexOctet(b, '\n'); i >= 0 {
			l, b = b[:i], b[i+1:]
		} else {
			l, b = b, nil
		}
		t := strings.TrimSpace(string(l))
		if t == "" || t[0] == '#' {
			continue
		}
		cle, val, ok := strings.Cut(t, "=")
		if !ok {
			continue
		}
		s.Definir(strings.TrimSpace(cle), strings.TrimSpace(val)) // erreur : ligne ignorée
	}
	return s
}

func indexOctet(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

// Charger : le style du dossier des réglages (un seul ReadFile) ; le défaut si le fichier manque ou si
// AGENTLINE_STYLE=default.
func Charger(dossier string, getenv func(string) string) *Style {
	if Force(getenv) {
		return Defaut()
	}
	return LireFichier(dossier)
}

// LireFichier : le style écrit dans le dossier des réglages, même si AGENTLINE_STYLE=default.
func LireFichier(dossier string) *Style {
	b, err := os.ReadFile(filepath.Join(dossier, Fichier))
	if err != nil {
		return Defaut()
	}
	return Lire(b)
}

// Force : AGENTLINE_STYLE=default, le fichier du style est ignoré.
func Force(getenv func(string) string) bool {
	v := ""
	if getenv != nil {
		v = getenv("AGENTLINE_STYLE")
	}
	return v == "default" || v == "defaut" // defaut : l'ancien nom, encore lu
}

// Ecrire : le style dans le dossier des réglages (écriture atomique) ; sans écart, le fichier est retiré.
func Ecrire(dossier string, s *Style) error {
	if s.EstDefaut() {
		err := os.Remove(filepath.Join(dossier, Fichier))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return theme.EcrireAtomique(dossier, Fichier, s.Texte())
}

// Valeur : la valeur en vigueur de l'option (son défaut si elle n'est pas réglée ou inconnue).
func (s *Style) Valeur(nom string) string {
	nom = NomCle(nom)
	if s != nil {
		if v, ok := s.v[nom]; ok {
			return v
		}
	}
	c, _ := Trouver(nom)
	return c.Defaut
}

// Regle : l'option s'écarte-t-elle du défaut ?
func (s *Style) Regle(nom string) bool {
	if s == nil {
		return false
	}
	_, ok := s.v[NomCle(nom)]
	return ok
}

// Oui : l'option yes|no vaut-elle yes ?
func (s *Style) Oui(nom string) bool { return s.Valeur(nom) == Oui }

// Entier : la valeur d'une option numérique.
func (s *Style) Entier(nom string) int {
	n, _ := strconv.Atoi(s.Valeur(nom))
	return n
}

// Definir : règle l'option, après validation ; la valeur par défaut retire le réglage. Les anciens noms
// français (de l'option, des items, des valeurs) sont acceptés et gardés sous leur nom actuel.
func (s *Style) Definir(nom, v string) error {
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
	if v == c.Defaut {
		delete(s.v, c.Nom)
		return nil
	}
	if s.v == nil {
		s.v = map[string]string{}
	}
	s.v[c.Nom] = v
	return nil
}

// normaliser : une disposition ou une liste d'items écrite toujours de la même façon (« a b | c »), avec
// les noms actuels des items.
func normaliser(c Cle, v string) string {
	if c.Genre == ListeItems {
		var l []string
		for _, n := range strings.Fields(v) {
			i, _ := NomItem(n)
			l = append(l, i)
		}
		return strings.Join(l, " ")
	}
	segs, _ := LireDisposition(v)
	return EcrireDisposition(segs)
}

// Retirer : l'option revient à son défaut.
func (s *Style) Retirer(nom string) { delete(s.v, NomCle(nom)) }

// EstDefaut : aucune option ne s'écarte du défaut.
func (s *Style) EstDefaut() bool { return s == nil || len(s.v) == 0 }

// Copie : un style indépendant, de mêmes valeurs.
func (s *Style) Copie() *Style {
	c := &Style{}
	if s != nil && len(s.v) > 0 {
		c.v = make(map[string]string, len(s.v))
		for k, v := range s.v {
			c.v[k] = v
		}
	}
	return c
}

// Egal : les deux styles ont les mêmes valeurs.
func (s *Style) Egal(o *Style) bool {
	for _, c := range catalogue {
		if s.Valeur(c.Nom) != o.Valeur(c.Nom) {
			return false
		}
	}
	return true
}

// Texte : le contenu du fichier du style (les écarts au défaut, dans l'ordre du catalogue).
func (s *Style) Texte() string {
	var b strings.Builder
	b.WriteString(i18n.T("# Style de la statusline : agentline style keys pour les options, agentline style pour les régler.\n"))
	for _, c := range catalogue {
		if s.Regle(c.Nom) {
			b.WriteString(c.Nom + "=" + s.v[c.Nom] + "\n")
		}
	}
	return b.String()
}

// ToutDefaut : un fichier de style qui écrit chaque option à sa valeur par défaut, en toutes lettres.
func ToutDefaut() string {
	var b strings.Builder
	for _, c := range catalogue {
		b.WriteString(c.Nom + "=" + c.Defaut + "\n")
	}
	return b.String()
}

// Masques : les items de l'option hide.
func (s *Style) Masques() map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields(s.Valeur("hide")) {
		m[n] = true
	}
	return m
}

// Masquer : ajoute (oui) ou retire l'item de l'option hide, dans l'ordre des items.
func (s *Style) Masquer(item string, oui bool) {
	if i, ok := NomItem(item); ok {
		item = i
	}
	m := s.Masques()
	m[item] = oui
	var l []string
	for _, i := range Items {
		if m[i] {
			l = append(l, i)
		}
	}
	s.Definir("hide", strings.Join(l, " "))
}

// Glyphe : l'icône en vigueur (icon.<nom>, sinon celle du jeu icons).
func (s *Style) Glyphe(nom string) string {
	if v := s.Valeur("icon." + nom); v != Auto {
		return v
	}
	return GlypheDuJeu(s.Valeur("icons"), nom)
}

// GlypheDuJeu : l'icône du jeu (nerd, text, none) ; le signal ● reste sauf si on le change.
func GlypheDuJeu(jeu, nom string) string {
	switch jeu {
	case "text":
		return glyphesTexte[nom]
	case "none":
		if nom == "signal" {
			return glyphesNerd[nom]
		}
		return ""
	}
	return glyphesNerd[nom]
}

// CasesBarre : les trois caractères de la barre du contexte (plein, moitié, éteint) ; ok faux : pas de barre.
func (s *Style) CasesBarre() (c [3]string, ok bool) {
	v := s.Valeur("ctx.bar")
	if b, nomme := barres[v]; nomme {
		v = b
	}
	r := []rune(v)
	if len(r) != 3 {
		return c, false
	}
	return [3]string{string(r[0]), string(r[1]), string(r[2])}, true
}
