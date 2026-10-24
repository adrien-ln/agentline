// Package installation : « agentline install », « agentline uninstall » et « agentline doctor ». Il ne touche qu'à des
// endroits précis (le binaire, la clé "statusLine" de settings.json, le PATH utilisateur sous Windows
// ou un bloc balisé du fichier du shell ailleurs, les restes de « sl », le nom d'avant ; avec l'accord
// de l'utilisateur, la police et la clé de police de Windows Terminal et de VS Code), fait une copie
// .bak datée de chaque fichier de configuration avant de le modifier, et ne
// change rien quand tout est déjà en place. Tout ce qui touche au système (environnement, commandes,
// registre, téléchargement, question posée) est injecté par Options : les tests n'ont aucun effet réel.
package installation

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var blocStatusLine = regexp.MustCompile(`"statusLine"\s*:\s*\{[^{}]*\}`)

// BlocStatusLine : la clé "statusLine" qui lance la commande, au format du reste de settings.json.
func BlocStatusLine(commande string) string {
	c, _ := json.Marshal(commande)
	return "\"statusLine\": {\n    \"type\": \"command\",\n    \"command\": " + string(c) + ",\n    \"padding\": 0\n  }"
}

func compact(s string) string { return strings.Join(strings.Fields(s), "") }

// PoserStatusLine : le contenu de settings.json avec la clé posée ; résultat « identique », « remplacé »,
// « ajouté » ou « créé ». Le reste du fichier n'est pas reformaté.
func PoserStatusLine(contenu, bloc string) (string, string) {
	t := strings.TrimSpace(contenu)
	if t == "" || compact(t) == "{}" {
		return "{\n  " + bloc + "\n}\n", "créé"
	}
	if ancien := blocStatusLine.FindString(contenu); ancien != "" {
		if compact(ancien) == compact(bloc) {
			return contenu, "identique"
		}
		return strings.Replace(contenu, ancien, bloc, 1), "remplacé"
	}
	i := strings.Index(contenu, "{")
	return contenu[:i+1] + "\n  " + bloc + "," + contenu[i+1:], "ajouté"
}

// RetirerStatusLine : le contenu sans la clé (et sans sa virgule) ; faux si elle est absente.
func RetirerStatusLine(contenu string) (string, bool) {
	loc := blocStatusLine.FindStringIndex(contenu)
	if loc == nil {
		return contenu, false
	}
	avant, apres := contenu[:loc[0]], contenu[loc[1]:]
	if a := strings.TrimLeft(apres, " \t\r\n"); strings.HasPrefix(a, ",") { // suivie d'une autre clé
		apres = strings.TrimLeft(a[1:], " \t\r\n")
		return avant + apres, true
	}
	if a := strings.TrimRight(avant, " \t\r\n"); strings.HasSuffix(a, ",") { // dernière clé
		avant = a[:len(a)-1]
	} else { // seule clé
		avant = strings.TrimRight(avant, " \t\r\n")
	}
	return avant + "\n" + strings.TrimLeft(apres, " \t\r\n"), true
}

// --- Blocs balisés (fichiers du shell) ---
//
// Les balises « statusline » sont celles de « sl install » et de l'ancienne version bash (fichiers du
// shell, profil PowerShell) : toujours reconnues, pour remplacer ou retirer un bloc posé avant le
// renommage.

const (
	blocDebut   = "# >>> agentline"
	blocFin     = "# <<< agentline <<<"
	ancienDebut = "# >>> statusline"
	ancienFin   = "# <<< statusline <<<"
)

// Bloc : des lignes entre les balises de « agentline install » (le « # » des commentaires vaut pour
// bash, zsh et fish).
func Bloc(corps string) string {
	return blocDebut + " : ajouté par « agentline install » (github.com/adrien-ln/agentline), retiré par « agentline uninstall » >>>\n" +
		strings.TrimRight(corps, "\n") + "\n" + blocFin
}

// trouverBloc : le début et la fin (exclue) du premier bloc balisé, nouvelles balises ou anciennes ;
// faux s'il n'y en a pas.
func trouverBloc(c string) (int, int, bool) {
	for _, b := range [][2]string{{blocDebut, blocFin}, {ancienDebut, ancienFin}} {
		if i := strings.Index(c, b[0]); i >= 0 {
			if j := strings.Index(c[i:], b[1]); j >= 0 {
				return i, i + j + len(b[1]), true
			}
		}
	}
	return 0, 0, false
}

// PoserBloc : le fichier avec le bloc posé (remplacé s'il existe, ancien bloc compris) ; même résultat
// que PoserStatusLine.
func PoserBloc(contenu, bloc string) (string, string) {
	if strings.TrimSpace(contenu) == "" {
		return bloc + "\n", "créé"
	}
	c := strings.TrimRight(contenu, "\n")
	if i, j, ok := trouverBloc(c); ok {
		if c[i:j] == bloc {
			return contenu, "identique"
		}
		return c[:i] + bloc + c[j:] + "\n", "remplacé"
	}
	return c + "\n\n" + bloc + "\n", "ajouté"
}

// ABloc : vrai si le fichier contient un bloc d'« agentline install » (ou de « sl install »).
func ABloc(contenu string) bool {
	_, _, ok := trouverBloc(contenu)
	return ok
}

// RetirerBloc : le fichier sans le bloc ni la ligne vide ajoutée avant lui ; faux s'il est absent.
func RetirerBloc(contenu string) (string, bool) {
	c := strings.TrimRight(contenu, "\n")
	i, j, ok := trouverBloc(c)
	if !ok {
		return contenu, false
	}
	avant, apres := c[:i], c[j:]
	avant = strings.TrimSuffix(strings.TrimSuffix(avant, "\n\n"), "\n")
	if avant == "" && apres == "" {
		return "", true
	}
	return avant + apres + "\n", true
}

// CopieNom : un nom de copie libre (deux passages dans la même seconde ne s'écrasent pas).
func CopieNom(chemin, horodatage string) string {
	n := chemin + "." + horodatage + ".bak"
	for k := 1; ; k++ {
		if _, err := os.Lstat(n); os.IsNotExist(err) {
			return n
		}
		n = chemin + "." + horodatage + "-" + strconv.Itoa(k) + ".bak"
	}
}
