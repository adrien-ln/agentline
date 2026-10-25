package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/installation"
)

// commandesPolice : les sous-commandes de « agentline font ».
func commandesPolice() []commande {
	return []commande{
		{nom: "list", alias: []string{"ls"}, aide: i18n.T("les Nerd Fonts proposées, celles qui sont installées, et la police de chaque terminal"), faire: policeListe},
		{nom: "set", args: i18n.T("<police>"), aide: i18n.T("installe cette Nerd Font si elle manque et la choisit dans Windows Terminal et VS Code"), min: 1, max: 1, faire: policeChoisir},
		{nom: "help", alias: []string{"aide"}, aide: i18n.T("cette aide"), faire: func(a *appel) int { io.WriteString(a.sortie, aidePolice()); return 0 }},
	}
}

func aidePolice() string {
	return aideTable(commandesPolice(), "agentline font") +
		i18n.T("Les icônes de la statusline demandent une Nerd Font ; c'est le terminal qui l'affiche, pas agentline.\n")
}

// commandePolice : « agentline font … » ; sans commande, la liste ; renvoie le code de sortie.
func commandePolice(args []string, sortie, erreurs io.Writer) int {
	if len(args) == 0 {
		args = []string{"list"}
	}
	if estAide(args[0]) {
		io.WriteString(sortie, aidePolice())
		return 0
	}
	c, ok := chercher(commandesPolice(), args[0])
	if !ok {
		fmt.Fprint(erreurs, i18n.T("agentline font : commande inconnue : %s (voir « agentline font help »)\n", args[0]))
		return codeUsage
	}
	return lancer(c, "agentline font", args[1:], sortie, erreurs)
}

// policeListe : les polices proposées (installée, retenue), puis la police réglée dans chaque terminal.
// Rien n'est modifié.
func policeListe(a *appel) int {
	o := optionsReelles(a)
	retenue := o.PoliceRetenue()
	for _, p := range installation.Polices {
		var etat []string
		if o.PoliceInstallee(p) {
			etat = append(etat, i18n.T("installée"))
		}
		if p.Nom == retenue {
			etat = append(etat, i18n.T("retenue"))
		}
		fmt.Fprintln(a.sortie, strings.TrimRight(fmt.Sprintf("  %-14s %-30s %s", p.Nom, p.Titre, strings.Join(etat, ", ")), " "))
	}
	for _, r := range o.PolicesReglees() {
		police := r.Police
		if police == "" {
			police = i18n.T("aucune police réglée")
		}
		fmt.Fprintf(a.sortie, i18n.T("%s : %s (%s)\n"), r.Terminal, police, r.Fichier)
	}
	return 0
}

// policeChoisir : « agentline font set <police> » : installe la police si elle manque, la choisit dans
// Windows Terminal et VS Code (copie .bak avant) et la retient pour les installations suivantes.
func policeChoisir(a *appel) int {
	p, ok := installation.PoliceNommee(a.args[0])
	if !ok {
		return a.usage(i18n.T("police inconnue : %s (au choix : %s)"), a.args[0], strings.Join(installation.NomsPolices(), ", "))
	}
	o := optionsReelles(a)
	o.Police, o.PoliceImposee = p.Nom, true
	if err := o.AppliquerPolice(); err != nil {
		return a.echec(i18n.T("police non installée (%s). À la main : %s"), err.Error(), o.CommandePolice())
	}
	return 0
}
