package claude

import (
	"regexp"
	"strings"
	"sync"
)

// Lecture du JSON par expressions régulières, exactement comme l'ancienne version bash : même tolérance
// (un JSON mal formé affiche ce qui a pu être lu), mêmes résultats.

var (
	motifs   = map[string]*regexp.Regexp{}
	motifsMu sync.Mutex
)

func motif(forme, cle string) *regexp.Regexp {
	k := forme + "\x00" + cle
	motifsMu.Lock()
	defer motifsMu.Unlock()
	if re, ok := motifs[k]; ok {
		return re
	}
	q := regexp.QuoteMeta(cle)
	var src string
	switch forme {
	case "val":
		src = `"` + q + `":(-?[0-9.]+([eE][-+]?[0-9]+)?|true|false)`
	case "str":
		src = `"` + q + `":"(([^"\\]|\\.)*)"`
	case "obj":
		src = `"` + q + `":\{(([^{}]|\{[^{}]*\})*)\}`
	}
	re := regexp.MustCompile(src)
	motifs[k] = re
	return re
}

// val : nombre ou booléen, vide si la clé manque.
func val(s, cle string) string {
	if m := motif("val", cle).FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// str : chaîne ; seuls \" et \\ sont décodés, comme en bash.
func str(s, cle string) string {
	m := motif("str", cle).FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	r := strings.ReplaceAll(m[1], `\"`, `"`)
	return strings.ReplaceAll(r, `\\`, `\`)
}

// obj : contenu d'un objet, un niveau d'imbrication toléré.
func obj(s, cle string) string {
	if m := motif("obj", cle).FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}
