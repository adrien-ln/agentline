package theme

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
)

// L'import des jeux de couleurs de terminal (agentline theme import) : Windows Terminal (un jeu, ou tout
// settings.json), iTerm2 (.itermcolors) et base16 (YAML). Leurs 16 couleurs ANSI deviennent les rôles
// d'un thème (DepuisJeu), que l'éditeur ouvre ensuite pour les retoucher.

// JeuANSI : un jeu de couleurs de terminal : son nom, le fond, le texte, les 16 couleurs ANSI (0 noir, 1 rouge,
// 2 vert, 3 jaune, 4 bleu, 5 magenta, 6 cyan, 7 blanc, puis leurs versions vives de 8 à 15), et l'orange s'il
// en a un (base16).
type JeuANSI struct {
	Nom         string
	Fond, Texte string
	ANSI        [16]string
	Orange      string
}

// Campbell : le jeu par défaut de Windows Terminal (il n'est pas écrit dans settings.json).
var Campbell = JeuANSI{Nom: "Campbell", Fond: "0c0c0c", Texte: "cccccc", ANSI: [16]string{
	"0c0c0c", "c50f1f", "13a10e", "c19c00", "0037da", "881798", "3a96dd", "cccccc",
	"767676", "e74856", "16c60c", "f9f1a5", "3b78ff", "b4009e", "61d6d6", "f2f2f2"}}

// cleWT : les noms des 16 couleurs dans un jeu de Windows Terminal, dans l'ordre ANSI.
var cleWT = []string{"black", "red", "green", "yellow", "blue", "purple", "cyan", "white",
	"brightBlack", "brightRed", "brightGreen", "brightYellow", "brightBlue", "brightPurple", "brightCyan", "brightWhite"}

// LireJeu : le jeu du fichier, d'après son contenu : plist (iTerm2), base16, sinon JSON de Windows Terminal ;
// nom : le jeu à prendre dans un settings.json qui en a plusieurs.
func LireJeu(contenu []byte, nom string) (JeuANSI, error) {
	s := strings.TrimSpace(string(contenu))
	switch {
	case strings.HasPrefix(s, "<?xml") || strings.HasPrefix(s, "<plist"):
		return LireITerm(contenu)
	case strings.Contains(s, "base00"):
		return LireBase16(contenu)
	}
	return LireWindowsTerminal(contenu, nom)
}

// LireWindowsTerminal : un jeu de Windows Terminal, seul ou dans settings.json (tableau schemes, choisi par
// nom ; un seul jeu se passe de nom).
func LireWindowsTerminal(contenu []byte, nom string) (JeuANSI, error) {
	var v any
	if err := json.Unmarshal([]byte(sansCommentaires(string(contenu))), &v); err != nil {
		return JeuANSI{}, fmt.Errorf(i18n.T("format inconnu : ni Windows Terminal, ni iTerm2, ni base16 (%v)"), err)
	}
	obj, _ := v.(map[string]any)
	if schemes, ok := obj["schemes"].([]any); ok {
		var noms []string
		for _, x := range schemes {
			s, _ := x.(map[string]any)
			n, _ := s["name"].(string)
			noms = append(noms, n)
			if strings.EqualFold(n, nom) || nom == "" && len(schemes) == 1 {
				return jeuWT(s)
			}
		}
		if nom == "Campbell" || nom == "" && len(schemes) == 0 {
			return Campbell, nil
		}
		return JeuANSI{}, fmt.Errorf(i18n.T("jeu de couleurs introuvable : %q (dans le fichier : %s)"), nom, strings.Join(noms, ", "))
	}
	return jeuWT(obj)
}

func jeuWT(s map[string]any) (JeuANSI, error) {
	c := func(k string) string { v, _ := s[k].(string); return hexa(v) }
	j := JeuANSI{Fond: c("background"), Texte: c("foreground")}
	j.Nom, _ = s["name"].(string)
	for k, n := range cleWT {
		j.ANSI[k] = c(n)
	}
	return j, j.complet()
}

// LireITerm : un fichier .itermcolors (plist XML : « Ansi 0 Color »… et leurs composantes de 0 à 1).
func LireITerm(contenu []byte) (JeuANSI, error) {
	d := xml.NewDecoder(bytes.NewReader(contenu))
	var racine any
	for {
		tok, err := d.Token()
		if err != nil {
			return JeuANSI{}, fmt.Errorf(i18n.T("fichier iTerm2 illisible : %v"), err)
		}
		if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "dict" {
			racine = valeurPlist(d, el)
			break
		}
	}
	dict, _ := racine.(map[string]any)
	couleur := func(k string) string {
		c, _ := dict[k].(map[string]any)
		comp := func(n string) int {
			f, _ := strconv.ParseFloat(fmt.Sprint(c[n+" Component"]), 64)
			return int(math.Round(math.Max(0, math.Min(1, f)) * 255))
		}
		if c == nil {
			return ""
		}
		return fmt.Sprintf("%02x%02x%02x", comp("Red"), comp("Green"), comp("Blue"))
	}
	j := JeuANSI{Fond: couleur("Background Color"), Texte: couleur("Foreground Color")}
	for k := range j.ANSI {
		j.ANSI[k] = couleur(fmt.Sprintf("Ansi %d Color", k))
	}
	return j, j.complet()
}

// valeurPlist : la valeur qui commence à el (dict -> map, sinon son texte).
func valeurPlist(d *xml.Decoder, el xml.StartElement) any {
	if el.Name.Local != "dict" {
		var t string
		d.DecodeElement(&t, &el)
		return strings.TrimSpace(t)
	}
	m := map[string]any{}
	cle := ""
	for {
		tok, err := d.Token()
		if err != nil {
			return m
		}
		switch x := tok.(type) {
		case xml.StartElement:
			if x.Name.Local == "key" {
				d.DecodeElement(&cle, &x)
				continue
			}
			m[cle] = valeurPlist(d, x)
		case xml.EndElement:
			if x.Name.Local == "dict" {
				return m
			}
		}
	}
}

var (
	ligneBase16 = regexp.MustCompile(`(?m)^\s*base0([0-9A-Fa-f])\s*:\s*["']?#?([0-9A-Fa-f]{6})`)
	nomBase16   = regexp.MustCompile(`(?m)^\s*(?:scheme|name)\s*:\s*["']?([^"'\n]+?)["']?\s*$`)
)

// LireBase16 : un jeu base16 (base00 à base0F) ; base09 est son orange.
func LireBase16(contenu []byte) (JeuANSI, error) {
	var b [16]string
	for _, m := range ligneBase16.FindAllStringSubmatch(string(contenu), -1) {
		k, _ := strconv.ParseInt(m[1], 16, 0)
		b[k] = strings.ToLower(m[2])
	}
	j := JeuANSI{Fond: b[0], Texte: b[5], Orange: b[9], ANSI: [16]string{
		b[0], b[8], b[11], b[10], b[13], b[14], b[12], b[5],
		b[3], b[8], b[11], b[10], b[13], b[14], b[12], b[7]}}
	if m := nomBase16.FindStringSubmatch(string(contenu)); m != nil {
		j.Nom = strings.TrimSpace(m[1])
	}
	return j, j.complet()
}

func (j JeuANSI) complet() error {
	if j.Fond == "" || j.Texte == "" {
		return errors.New(i18n.T("jeu de couleurs incomplet : il manque le fond ou le texte"))
	}
	for _, c := range j.ANSI[:8] {
		if c == "" {
			return errors.New(i18n.T("jeu de couleurs incomplet : il manque des couleurs ANSI"))
		}
	}
	return nil
}

// hexa : « #RRGGBB » -> « rrggbb » (vide si ce n'en est pas un).
func hexa(v string) string {
	v = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "#"))
	if len(v) != 6 {
		return ""
	}
	if _, err := strconv.ParseUint(v, 16, 32); err != nil {
		return ""
	}
	return v
}

// sansCommentaires : du JSON avec commentaires (// et /* */) et virgules finales, rendu lisible par
// encoding/json ; les chaînes ne sont pas touchées.
func sansCommentaires(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			j := i + 1
			for j < len(s) && s[j] != '"' {
				if s[j] == '\\' {
					j++
				}
				j++
			}
			b.WriteString(s[i:min(j+1, len(s))])
			i = j
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			b.WriteByte('\n')
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			fin := strings.Index(s[i+2:], "*/")
			if fin < 0 {
				return b.String()
			}
			i += fin + 3
		case c == ',':
			k := i + 1
			for k < len(s) && strings.IndexByte(" \t\r\n", s[k]) >= 0 {
				k++
			}
			if k < len(s) && (s[k] == '}' || s[k] == ']') {
				continue // virgule finale
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// DepuisJeu : le thème d'un jeu de couleurs : le texte, des gris tirés du fond et du texte, le nom du modèle
// en magenta, le dossier en bleu, la branche en cyan, les seuils en vert, jaune, orange (celui du jeu, sinon
// entre jaune et rouge) et rouge. Les versions vives servent sur fond sombre ; chaque couleur est ajustée
// pour rester lisible sur le fond du jeu.
func DepuisJeu(j JeuANSI) Palette {
	clair := ansi.EstClaire(j.Fond)
	vif := func(k int) string {
		if !clair && j.ANSI[k+8] != "" {
			return j.ANSI[k+8]
		}
		return j.ANSI[k]
	}
	lisible := func(h string, mini float64) string { return ansi.Ajuster(h, j.Fond, mini, 0) }
	orange := j.Orange
	if orange == "" {
		l1, c1, t1 := ansi.LCh(vif(3))
		l2, c2, t2 := ansi.LCh(vif(1))
		d := math.Mod(t1-t2+540, 360) - 180 // le plus court chemin de teinte, du rouge vers le jaune
		orange = ansi.DepuisLCh((l1+l2)/2, (c1+c2)/2, math.Mod(t2+d/2+360, 360))
	}
	muet := j.ANSI[8]
	if muet == "" {
		muet = melange(j.Fond, j.Texte, 0.55)
	}
	p := Palette{Clair: clair}
	p.Texte = lisible(j.Texte, 4.5)
	p.Discret = lisible(muet, 3)
	p.Sep, p.Eteint = melange(j.Fond, j.Texte, 0.25), melange(j.Fond, j.Texte, 0.4)
	p.Modele, p.Dossier, p.Lieu = lisible(vif(5), 3), lisible(vif(4), 3), lisible(vif(6), 3)
	p.D0, p.D50, p.D75, p.D100 = lisible(vif(2), 3), lisible(vif(3), 3), lisible(orange, 3), lisible(vif(1), 3)
	rougeVif := p.D100 // l'effort très haut : le rouge vif du jeu, s'il en a un
	if j.ANSI[9] != "" {
		rougeVif = lisible(j.ANSI[9], 3)
	}
	p.Intensite = [5]string{p.Lieu, p.Dossier, p.Modele, rougeVif, p.D75}
	return p
}

// melange : la couleur à t (0 : a, 1 : b) entre deux couleurs « rrggbb ».
func melange(a, b string, t float64) string {
	r1, g1, b1 := ansi.RGB(a)
	r2, g2, b2 := ansi.RGB(b)
	m := func(x, y int) int { return int(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return fmt.Sprintf("%02x%02x%02x", m(r1, r2), m(g1, g2), m(b1, b2))
}
