// Package i18n : les textes visibles d'agentline, en anglais (par défaut) et en français.
//
// Le texte français sert de clé, comme le msgid de gettext : le code reste lisible, et la traduction
// anglaise est cherchée dans le catalogue (catalogue_*.go). T traduit (et formate s'il y a des
// arguments), N choisit entre singulier et pluriel, M marque un texte traduit plus tard (par T sur une
// variable). Les tests vérifient que chaque texte marqué du code est au catalogue, avec les mêmes
// verbes % dans les deux langues, et qu'aucune entrée du catalogue n'est orpheline.
package i18n

import (
	"fmt"
	"strings"
)

// Langue : « en » ou « fr ».
type Langue string

const (
	EN Langue = "en"
	FR Langue = "fr"
)

var courante = EN

// Courante : la langue des textes.
func Courante() Langue { return courante }

// Francais : la langue est-elle le français ?
func Francais() bool { return courante == FR }

// Choisir : la langue des textes ; renvoie la précédente (pour la rétablir dans un test).
func Choisir(l Langue) Langue {
	avant := courante
	if l == FR || l == EN {
		courante = l
	}
	return avant
}

// Lire : la langue d'un réglage (« fr », « fr_FR.UTF-8 », « en-US », « C ») ; ok faux si le texte est
// vide. Une langue que l'outil ne parle pas donne l'anglais.
func Lire(s string) (l Langue, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return EN, false
	}
	if s == "fr" || strings.HasPrefix(s, "fr_") || strings.HasPrefix(s, "fr-") || strings.HasPrefix(s, "fr.") {
		return FR, true
	}
	return EN, true
}

// Detecter : AGENTLINE_LANG (en ou fr) > LC_ALL > LC_MESSAGES > LANG (la première définie décide,
// comme sous POSIX) > la langue d'interface de Windows > en. La langue trouvée devient la courante.
func Detecter(getenv func(string) string) Langue {
	l := detecter(getenv, langueSysteme)
	courante = l
	return l
}

func detecter(getenv func(string) string, systeme func() string) Langue {
	switch strings.ToLower(strings.TrimSpace(getenv("AGENTLINE_LANG"))) {
	case "fr":
		return FR
	case "en":
		return EN
	}
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := Lire(getenv(v)); ok {
			return l
		}
	}
	if systeme != nil {
		if l, ok := Lire(systeme()); ok {
			return l
		}
	}
	return EN
}

// T : le texte dans la langue courante (le français est la clé) ; avec des arguments, formaté comme
// par fmt.Sprintf.
func T(fr string, args ...any) string {
	s := fr
	if courante == EN {
		if e, ok := anglais[fr]; ok {
			s = e
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// N : T du singulier ou du pluriel selon n (en français, 0 et 1 sont au singulier ; en anglais, 1
// seulement).
func N(n int, un, plusieurs string, args ...any) string {
	if n == 1 || n == 0 && courante == FR {
		return T(un, args...)
	}
	return T(plusieurs, args...)
}

// M : marque un texte à traduire plus tard (T sur la variable qui le porte) ; le renvoie tel quel.
func M(fr string) string { return fr }

// anglais : le catalogue, texte français -> texte anglais, réuni à partir des tables de chaque zone.
var anglais = func() map[string]string {
	m := map[string]string{}
	for _, t := range tables {
		for k, v := range t {
			m[k] = v
		}
	}
	return m
}()
