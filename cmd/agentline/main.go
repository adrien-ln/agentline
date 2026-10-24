// agentline : la statusline de Claude Code et ses thèmes (réécriture en Go de l'ancienne version bash, voir docs/HISTORY.md).
//
//	agentline render < entrée.json   les deux lignes de la statusline (ce que Claude Code appelle à chaque événement)
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/adrien-ln/agentline"
	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/style"
	"github.com/adrien-ln/agentline/internal/theme"
)

// niveauCouleurs : le niveau de couleurs de la statusline, sa raison, et s'il est imposé par l'utilisateur.
// En SSH : un niveau imposé (AGENTLINE_COLORS, NO_COLOR) > AGENTLINE_SSH_RICH (24 bits, comme avant) > les
// couleurs ANSI fixes de la ligne courte (le niveau détecté ne sert alors qu'à TERM=dumb). Hors SSH : la
// détection de ansi.DetecterNiveau.
func niveauCouleurs(getenv func(string) string) (ansi.Niveau, string, bool) {
	n, raison := ansi.DetecterNiveau(getenv)
	impose := ansi.NiveauImpose(raison)
	if getenv("SSH_CONNECTION") == "" || impose {
		return n, raison, impose
	}
	if riche, _ := strconv.Atoi(getenv("AGENTLINE_SSH_RICH")); riche != 0 {
		return ansi.Truecolor, "AGENTLINE_SSH_RICH", false
	}
	return n, "SSH", false
}

func environnement() render.Env {
	now := time.Now().Unix()
	if s := getenv("AGENTLINE_NOW"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			now = n
		}
	}
	colonnes, _ := strconv.Atoi(getenv("COLUMNS"))
	riche, _ := strconv.Atoi(getenv("AGENTLINE_SSH_RICH"))
	delai := time.Second
	if s, err := strconv.ParseFloat(getenv("AGENTLINE_GIT_TIMEOUT"), 64); err == nil && s > 0 {
		delai = time.Duration(s * float64(time.Second))
	}
	niveau, _, impose := niveauCouleurs(getenv)
	themes := theme.Sources{Config: dossierConfig(), Integre: agentline.Themes, Windows: runtime.GOOS == "windows"}
	fond, _ := themes.Fond(getenv)
	return render.Env{
		Now: now, Loc: time.Local, Columns: colonnes,
		SSH: getenv("SSH_CONNECTION") != "", SSHRiche: riche != 0,
		Couleurs: ansi.Profil{Niveau: niveau, Fond: fond}, CouleursImposees: impose,
		TempDir: os.TempDir(), GitDelai: delai,
		Themes: themes,
		Getenv: getenv,
		Style:  style.Charger(dossierConfig(), getenv),
	}
}

// optionsInstallation : les options de install et uninstall (anciens noms acceptés, pas affichés).
func optionsInstallation() []option {
	return []option{
		{nom: "--in-place", anciens: []string{"--sur-place"}, aide: i18n.T("ne copie pas le binaire : utilise celui-ci, là où il est")},
		{nom: "--no-font", anciens: []string{"--sans-police"}, aide: i18n.T("sans Nerd Font : ne propose pas de l'installer, affiche la commande")},
	}
}

// commandesAgentline : les commandes de « agentline », dans l'ordre de l'aide.
func commandesAgentline() []commande {
	return []commande{
		{nom: "render", args: i18n.T("< entrée.json"), aide: i18n.T("les lignes de la statusline (ce que Claude Code appelle)"), brut: true,
			faire: func(a *appel) int { // jamais d'erreur : Claude Code l'appelle à chaque événement
				entree, _ := io.ReadAll(os.Stdin)
				io.WriteString(a.sortie, render.Render(string(entree), environnement()))
				return 0
			}},
		{nom: "theme", args: i18n.T("[commande]"), aide: i18n.T("sans commande : le sélecteur interactif ; voir ci-dessous"), brut: true,
			faire: func(a *appel) int { return commandeTheme(a.args, a.sortie, a.erreurs) }},
		{nom: "style", args: i18n.T("[commande]"), aide: i18n.T("sans commande : les items de la statusline, dans le sélecteur ; voir ci-dessous"), brut: true,
			faire: func(a *appel) int { return commandeStyle(a.args, a.sortie, a.erreurs) }},
		{nom: "install", args: "[options]", aide: i18n.T("installe agentline et déclare la statusline dans Claude Code"), max: 0, options: optionsInstallation(),
			faire: func(a *appel) int { return commandeInstallation(false, a) }},
		{nom: "uninstall", args: "[options]", aide: i18n.T("retire la statusline de Claude Code (tes réglages restent)"), max: 0, options: optionsInstallation(),
			faire: func(a *appel) int { return commandeInstallation(true, a) }},
		{nom: "doctor", aide: i18n.T("vérifie l'installation : chaque problème avec la commande qui le corrige"), max: 0,
			faire: commandeDoctor},
		{nom: "version", alias: []string{"--version"}, aide: i18n.T("la version d'agentline"),
			faire: func(a *appel) int { fmt.Fprintln(a.sortie, "agentline "+versionAffichee()); return 0 }},
		{nom: "help", alias: []string{"aide", "-h", "--help"}, aide: i18n.T("cette aide"),
			faire: func(a *appel) int { io.WriteString(a.sortie, aideGenerale()); return 0 }},
	}
}

// aideGenerale : toutes les commandes, celles de « agentline theme » comprises.
func aideGenerale() string {
	return i18n.T("agentline : la statusline de Claude Code et ses thèmes.\n\nCommandes :\n") +
		ligneAide("agentline", i18n.T("sans commande : le sélecteur interactif (thèmes ; Tab : la statusline)")) +
		aideTable(commandesAgentline(), "agentline") +
		i18n.T("\nThèmes :\n") + aideTheme() + i18n.T("\nStyle :\n") + aideStyle() + i18n.T("\nOptions de install et uninstall : agentline install --help\n") +
		i18n.T("Variables : %s. Leurs anciens noms (SL_THEME…) sont encore lus, en secours : obsolètes.\n", "AGENTLINE_"+strings.Join(variables, ", AGENTLINE_")) +
		i18n.T("Langue : AGENTLINE_LANG=en ou fr ; sinon LC_ALL, LC_MESSAGES, LANG, puis la langue du système.\n")
}

// interactif : vrai si l'entrée et la sortie sont des consoles, où le sélecteur peut s'ouvrir ; les
// tests la remplacent, comme ouvrirSelecteur.
var interactif = func() bool { return terminal(os.Stdin) && terminal(os.Stdout) }

var ouvrirSelecteur = themeChoisir

// principale : « agentline … » ; renvoie le code de sortie. Sans commande : le sélecteur dans une
// console, sinon (tube, fichier, script) l'aide et le code 2.
func principale(args []string, sortie, erreurs io.Writer) int {
	if len(args) == 0 {
		if interactif() {
			return ouvrirSelecteur(&appel{prefixe: "agentline", opts: map[string]bool{}, sortie: sortie, erreurs: erreurs})
		}
		io.WriteString(erreurs, aideGenerale())
		return codeUsage
	}
	c, ok := chercher(commandesAgentline(), args[0])
	if !ok {
		fmt.Fprint(erreurs, i18n.T("agentline : commande inconnue : %s (voir « agentline help »)\n", args[0]))
		return codeUsage
	}
	return lancer(c, "agentline", args[1:], sortie, erreurs)
}

func main() {
	i18n.Detecter(getenv)
	os.Exit(principale(os.Args[1:], os.Stdout, os.Stderr))
}
