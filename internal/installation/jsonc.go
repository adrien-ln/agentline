package installation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// --- JSON avec commentaires (Windows Terminal, VS Code) : lu sans être reformaté ---
//
// Ces fichiers acceptent les commentaires // et /* */ et les virgules finales. On n'en change qu'une
// valeur : le reste (espaces, commentaires, ordre des clés) est gardé à l'octet près.

// noeud : une valeur du fichier, sa place [debut, fin) et, pour un objet ou un tableau, son contenu.
type noeud struct {
	debut, fin int
	genre      byte // '{', '[', '"' ou 0 (nombre, true, false, null)
	cles       []cle
	elements   []*noeud
}

type cle struct {
	nom   string
	debut int // le guillemet ouvrant de la clé
	val   *noeud
}

func (n *noeud) cle(nom string) *noeud {
	for _, c := range n.cles {
		if c.nom == nom {
			return c.val
		}
	}
	return nil
}

type lecteur struct {
	s string
	i int
}

func (l *lecteur) erreur(quoi string) error {
	ligne := strings.Count(l.s[:l.i], "\n") + 1
	return fmt.Errorf("%s, ligne %d", quoi, ligne)
}

// blancs : espaces et commentaires.
func (l *lecteur) blancs() error {
	for l.i < len(l.s) {
		switch {
		case strings.ContainsRune(" \t\r\n", rune(l.s[l.i])):
			l.i++
		case strings.HasPrefix(l.s[l.i:], "//"):
			if j := strings.IndexByte(l.s[l.i:], '\n'); j >= 0 {
				l.i += j + 1
			} else {
				l.i = len(l.s)
			}
		case strings.HasPrefix(l.s[l.i:], "/*"):
			j := strings.Index(l.s[l.i+2:], "*/")
			if j < 0 {
				return l.erreur("commentaire non fermé")
			}
			l.i += j + 4
		default:
			return nil
		}
	}
	return nil
}

func (l *lecteur) chaine() (string, error) {
	debut := l.i
	l.i++
	for l.i < len(l.s) {
		switch l.s[l.i] {
		case '\\':
			l.i += 2
		case '"':
			l.i++
			var s string
			if err := json.Unmarshal([]byte(l.s[debut:l.i]), &s); err != nil {
				return "", l.erreur("chaîne invalide")
			}
			return s, nil
		case '\n':
			return "", l.erreur("chaîne non fermée")
		default:
			l.i++
		}
	}
	return "", l.erreur("chaîne non fermée")
}

func (l *lecteur) valeur() (*noeud, error) {
	if err := l.blancs(); err != nil {
		return nil, err
	}
	if l.i >= len(l.s) {
		return nil, l.erreur("valeur attendue")
	}
	n := &noeud{debut: l.i, genre: l.s[l.i]}
	switch l.s[l.i] {
	case '"':
		if _, err := l.chaine(); err != nil {
			return nil, err
		}
	case '{', '[':
		ferme := byte('}')
		if n.genre == '[' {
			ferme = ']'
		}
		l.i++
		for {
			if err := l.blancs(); err != nil {
				return nil, err
			}
			if l.i < len(l.s) && l.s[l.i] == ferme { // objet vide, ou virgule finale
				l.i++
				break
			}
			if n.genre == '{' {
				if l.i >= len(l.s) || l.s[l.i] != '"' {
					return nil, l.erreur("clé attendue")
				}
				c := cle{debut: l.i}
				nom, err := l.chaine()
				if err != nil {
					return nil, err
				}
				c.nom = nom
				if err := l.blancs(); err != nil {
					return nil, err
				}
				if l.i >= len(l.s) || l.s[l.i] != ':' {
					return nil, l.erreur("« : » attendu")
				}
				l.i++
				if c.val, err = l.valeur(); err != nil {
					return nil, err
				}
				n.cles = append(n.cles, c)
			} else {
				e, err := l.valeur()
				if err != nil {
					return nil, err
				}
				n.elements = append(n.elements, e)
			}
			if err := l.blancs(); err != nil {
				return nil, err
			}
			if l.i < len(l.s) && l.s[l.i] == ',' {
				l.i++
				continue
			}
			if l.i < len(l.s) && l.s[l.i] == ferme {
				l.i++
				break
			}
			return nil, l.erreur("« , » ou « " + string(ferme) + " » attendu")
		}
	default:
		n.genre = 0
		j := l.i
		for j < len(l.s) && strings.ContainsRune("+-.0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", rune(l.s[j])) {
			j++
		}
		mot := l.s[l.i:j]
		if _, err := strconv.ParseFloat(mot, 64); mot != "true" && mot != "false" && mot != "null" && err != nil {
			return nil, l.erreur("valeur invalide")
		}
		l.i = j
	}
	n.fin = l.i
	return n, nil
}

// lireJSONC : la valeur racine ; erreur si le fichier n'est pas un JSON (avec commentaires) complet.
func lireJSONC(s string) (*noeud, error) {
	l := &lecteur{s: s}
	if strings.HasPrefix(s, bom) {
		l.i = len(bom)
	}
	n, err := l.valeur()
	if err != nil {
		return nil, err
	}
	if err := l.blancs(); err != nil {
		return nil, err
	}
	if l.i != len(s) {
		return nil, l.erreur("texte après la fin")
	}
	return n, nil
}

func jsoncValide(s string) error { _, err := lireJSONC(s); return err }

const bom = "\xef\xbb\xbf" // marque d'ordre des octets UTF-8, que certains éditeurs mettent en tête

func guillemets(s string) string { b, _ := json.Marshal(s); return string(b) }

// ChaineJSONC : la chaîne au chemin (clés d'objets imbriqués) ; faux si elle est absente.
func ChaineJSONC(contenu string, chemin ...string) (string, bool) {
	n, err := lireJSONC(contenu)
	if err != nil {
		return "", false
	}
	for _, k := range chemin {
		if n.genre != '{' {
			return "", false
		}
		if n = n.cle(k); n == nil {
			return "", false
		}
	}
	var s string
	if n.genre != '"' || json.Unmarshal([]byte(contenu[n.debut:n.fin]), &s) != nil {
		return "", false
	}
	return s, true
}

// PoserChaineJSONC : le contenu avec la chaîne valeur au chemin, les objets manquants créés ; résultat
// « identique », « remplacé » ou « ajouté ». Le reste n'est pas reformaté (indentation et fins de ligne
// du fichier reprises pour ce qui est ajouté).
func PoserChaineJSONC(contenu, valeur string, chemin ...string) (string, string, error) {
	if strings.TrimSpace(contenu) == "" {
		contenu = "{}\n"
	}
	racine, err := lireJSONC(contenu)
	if err != nil {
		return "", "", err
	}
	val := guillemets(valeur)
	fl := "\n"
	if strings.Contains(contenu, "\r\n") {
		fl = "\r\n"
	}
	unite := uniteIndentation(contenu)
	ind := func(p int) string { return strings.Repeat(unite, p) }

	n := racine
	for i, k := range chemin {
		if n.genre != '{' {
			return "", "", fmt.Errorf("%s n'est pas un objet", strings.Join(chemin[:i], "."))
		}
		suivant := n.cle(k)
		if suivant != nil && i < len(chemin)-1 {
			n = suivant
			continue
		}
		if suivant != nil { // la valeur elle-même
			if contenu[suivant.debut:suivant.fin] == val {
				return contenu, "identique", nil
			}
			return contenu[:suivant.debut] + val + contenu[suivant.fin:], "remplacé", nil
		}
		// k manque dans n (profondeur i) : la clé et les objets qui suivent, imbriqués
		texte := val
		for j := len(chemin) - 1; j > i; j-- {
			texte = "{" + fl + ind(j+1) + guillemets(chemin[j]) + ": " + texte + fl + ind(j) + "}"
		}
		paire := guillemets(k) + ": " + texte
		if len(n.cles) > 0 {
			premiere := n.cles[0].debut
			debutLigne := strings.LastIndexByte(contenu[:premiere], '\n') + 1
			retrait := contenu[debutLigne:premiere]
			if strings.TrimSpace(retrait) != "" { // la clé n'est pas en début de ligne
				return contenu[:premiere] + paire + ", " + contenu[premiere:], "ajouté", nil
			}
			return contenu[:premiere] + paire + "," + fl + retrait + contenu[premiere:], "ajouté", nil
		}
		return contenu[:n.debut] + "{" + fl + ind(i+1) + paire + fl + ind(i) + "}" + contenu[n.fin:], "ajouté", nil
	}
	return contenu, "identique", nil
}

// uniteIndentation : le retrait de la première clé indentée du fichier (quatre espaces à défaut).
func uniteIndentation(s string) string {
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimLeft(l, " \t")
		if strings.HasPrefix(t, `"`) && len(t) < len(l) {
			return l[:len(l)-len(t)]
		}
	}
	return "    "
}
