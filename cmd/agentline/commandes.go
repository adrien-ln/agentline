package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Codes de sortie : 0 réussi, 1 échec (écriture impossible…), 2 erreur d'usage (commande, option ou
// argument invalide), comme la plupart des outils en ligne de commande.
const (
	codeEchec = 1
	codeUsage = 2
)

// commande : une commande ou une sous-commande, et de quoi générer son aide.
type commande struct {
	nom      string
	alias    []string // anciens noms ou raccourcis, acceptés mais pas affichés
	args     string   // ce qui suit le nom, dans l'aide (« <thème> [dossier] »)
	aide     string
	cachee   bool     // pas dans l'aide (alias d'une autre)
	min, max int      // nombre d'arguments, options non comprises
	brut     bool     // arguments passés tels quels (ni option ni contrôle du nombre)
	options  []option // options acceptées ; -h et --help le sont toujours
	faire    func(a *appel) int
}

// option : une option, son aide, et ses anciens noms (acceptés, pas affichés).
type option struct {
	nom, aide string
	anciens   []string
	valeur    string // le nom de sa valeur dans l'aide (« name ») ; vide : une option sans valeur
}

// appel : une commande lancée, ses arguments et options, où écrire.
type appel struct {
	cmd             commande
	prefixe         string // « agentline » ou « agentline theme »
	args            []string
	opts            map[string]bool
	vals            map[string]string // la valeur des options qui en ont une (--font firacode)
	sortie, erreurs io.Writer
}

// usage : une erreur d'usage, sur la sortie d'erreur, avec la ligne d'usage de la commande.
func (a *appel) usage(format string, x ...any) int {
	fmt.Fprintf(a.erreurs, i18n.T("%s : %s\n"), a.prefixe, fmt.Sprintf(format, x...))
	if a.cmd.nom != "" {
		fmt.Fprintf(a.erreurs, i18n.T("usage : %s\n"), strings.TrimSpace(a.prefixe+" "+a.cmd.nom+" "+a.cmd.args))
	}
	return codeUsage
}

// echec : une erreur d'exécution (fichier illisible, écriture impossible).
func (a *appel) echec(format string, x ...any) int {
	fmt.Fprintf(a.erreurs, i18n.T("%s : %s\n"), a.prefixe, fmt.Sprintf(format, x...))
	return codeEchec
}

// chercher : la commande de ce nom (ou alias) dans la table.
func chercher(table []commande, nom string) (commande, bool) {
	for _, c := range table {
		if c.nom == nom {
			return c, true
		}
		for _, a := range c.alias {
			if a == nom {
				return c, true
			}
		}
	}
	return commande{}, false
}

func estAide(a string) bool { return a == "-h" || a == "--help" }

// lancer : sépare options et arguments, vérifie leur nombre, puis exécute la commande ; -h ou --help
// affiche son aide.
func lancer(c commande, prefixe string, args []string, sortie, erreurs io.Writer) int {
	a := &appel{cmd: c, prefixe: prefixe, opts: map[string]bool{}, vals: map[string]string{}, sortie: sortie, erreurs: erreurs}
	if c.brut {
		a.args = args
		return c.faire(a)
	}
	for k := 0; k < len(args); k++ {
		x := args[k]
		switch {
		case estAide(x):
			io.WriteString(sortie, usageDe(c, prefixe))
			return 0
		case strings.HasPrefix(x, "-") && x != "-":
			nom, val, avecEgal := strings.Cut(x, "=")
			o, ok := optionDe(c, nom)
			switch {
			case !ok:
				return a.usage(i18n.T("option inconnue : %s%s"), x, listeOptions(c))
			case o.valeur == "" && avecEgal:
				return a.usage(i18n.T("%s ne prend pas de valeur"), o.nom)
			case o.valeur != "" && !avecEgal:
				if k+1 >= len(args) {
					return a.usage(i18n.T("il manque la valeur de %s"), o.nom)
				}
				k++
				val = args[k]
			}
			a.opts[o.nom] = true
			if o.valeur != "" {
				a.vals[o.nom] = val
			}
		default:
			a.args = append(a.args, x)
		}
	}
	switch {
	case len(a.args) < c.min:
		return a.usage("%s", i18n.T("il manque un argument"))
	case len(a.args) > c.max:
		return a.usage(i18n.T("argument en trop : %s"), a.args[c.max])
	}
	return c.faire(a)
}

// optionDe : l'option x de la commande (par son nom actuel ou un ancien) ; faux si elle ne l'a pas.
func optionDe(c commande, x string) (option, bool) {
	for _, o := range c.options {
		if o.nom == x || slices.Contains(o.anciens, x) {
			return o, true
		}
	}
	return option{}, false
}

func listeOptions(c commande) string {
	if len(c.options) == 0 {
		return i18n.T(" (aucune option)")
	}
	var n []string
	for _, o := range c.options {
		n = append(n, o.nom)
	}
	return " (" + strings.Join(n, ", ") + ")"
}

const colonneAide = 41 // où commence l'explication, dans les aides (« agentline theme current [--why] [dossier] » tient)

func ligneAide(gauche, droite string) string {
	return fmt.Sprintf("  %-*s %s\n", colonneAide, gauche, droite)
}

// aideTable : une ligne par commande visible de la table.
func aideTable(table []commande, prefixe string) string {
	var b strings.Builder
	for _, c := range table {
		if !c.cachee {
			b.WriteString(ligneAide(strings.TrimSpace(prefixe+" "+c.nom+" "+c.args), c.aide))
		}
	}
	return b.String()
}

// usageDe : l'aide d'une commande (agentline theme set --help).
func usageDe(c commande, prefixe string) string {
	var b strings.Builder
	fmt.Fprintf(&b, i18n.T("usage : %s\n%s\n"), strings.TrimSpace(prefixe+" "+c.nom+" "+c.args), c.aide)
	if len(c.options) > 0 {
		b.WriteString(i18n.T("\nOptions :\n"))
		for _, o := range c.options {
			nom := o.nom
			if o.valeur != "" {
				nom += " <" + o.valeur + ">"
			}
			b.WriteString(ligneAide(nom, o.aide))
		}
	}
	return b.String()
}
