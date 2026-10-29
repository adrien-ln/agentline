package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/installation"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/selecteur"
	"github.com/adrien-ln/agentline/internal/style"

	tea "github.com/charmbracelet/bubbletea"
)

// commandesPolice : les sous-commandes de « agentline font ».
func commandesPolice() []commande {
	return []commande{
		{nom: "list", alias: []string{"ls"}, aide: i18n.T("les Nerd Fonts proposées, celles qui sont installées, et la police de chaque terminal"), faire: policeListe},
		{nom: "preview", aide: i18n.T("essayer les Nerd Fonts installées dans ce terminal, pour de vrai, puis garder l'une ou remettre la tienne"), faire: policeApercu},
		{nom: "test", aide: i18n.T("les glyphes d'agentline dessinés dans la police de ce terminal : ce qui manque ou déborde se voit"), faire: policeTest},
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

// glyphesDivers : les caractères des barres, cadres, flèches et de Pip, par groupe (un caractère par case).
var glyphesDivers = [][2]string{
	{i18n.M("Pip (demi-blocs)"), "▘▝▀▖▌▞▛▗▚▐▜▄▙▟█▂─"},
	{i18n.M("barres du contexte"), "━╸─█▌░●◐○"},
	{i18n.M("jauges en un glyphe"), style.GlyphesJauges()},
	{i18n.M("icônes des modèles"), style.GlyphesModeles()},
	{i18n.M("cadres et flèches"), "╭╮╰╯│↑↓←→↗↻✓×·"},
}

// policeTest : « agentline font test » : chaque glyphe qu'utilise agentline entre deux barres, dans la police
// du terminal, puis la statusline d'exemple et la police réglée dans chaque terminal. Rien n'est modifié.
func policeTest(a *appel) int {
	w := a.sortie
	fmt.Fprint(w, i18n.T("Chaque glyphe doit tenir entre ses deux barres, toutes alignées : un carré ou un vide, la police n'a pas ce glyphe ; une barre décalée, il est trop large (police non « Mono »).\n"))
	noms := func(titre string, l [][2]string) {
		fmt.Fprintf(w, "\n%s\n", titre)
		for _, g := range l {
			fmt.Fprintf(w, "  |%s|  %s\n", g[1], g[0])
		}
	}
	var icones [][2]string
	for _, c := range style.Cles() {
		if n, ok := strings.CutPrefix(c.Nom, "icon."); ok {
			icones = append(icones, [2]string{n, style.GlypheDuJeu("nerd", n)})
		}
	}
	noms(i18n.T("Icônes de la statusline (icons=nerd)"), icones)
	noms(i18n.T("Touches du sélecteur"), selecteur.GlyphesTouches())
	for _, g := range glyphesDivers {
		cases := []rune(g[1])
		regle := strings.Repeat("|-", len(cases)) + "|"
		fmt.Fprintf(w, "\n%s\n  %s\n  |%s|\n", i18n.T(g[0]), regle, strings.Join(strings.Split(g[1], ""), "|"))
	}
	e := environnement()
	if e.Columns == 0 {
		e.Columns = 120
	}
	d, _ := os.Getwd()
	fmt.Fprintf(w, "\n%s\n%s", i18n.T("La statusline avec ce thème et ce style :"), render.RenderSession(etat.Exemple(e.Now, filepath.ToSlash(d), 42), e))
	o := optionsReelles(a)
	for _, r := range o.PolicesReglees() {
		police := r.Police
		if police == "" {
			police = i18n.T("aucune police réglée")
		}
		fmt.Fprintf(w, i18n.T("%s : %s (%s)\n"), r.Terminal, police, r.Fichier)
	}
	return 0
}

// policeApercu : « agentline font preview » : la police du terminal change pour de vrai à chaque choix ;
// Entrée la garde (comme agentline font set, dans les deux terminaux), Échap remet l'ancienne. Un aperçu
// interrompu auparavant est d'abord défait.
func policeApercu(a *appel) int {
	o := optionsReelles(a)
	if msg := o.RemettreApercuInterrompu(); msg != "" {
		fmt.Fprintln(a.sortie, msg)
	}
	ap, err := o.PreparerApercu()
	if errors.Is(err, installation.ErrSansApercu) {
		return a.echec("%s", i18n.T("l'aperçu en direct marche dans Windows Terminal et VS Code ; ici, « agentline font test » montre les glyphes dans ta police"))
	}
	if err != nil {
		return a.echec("%v", err)
	}
	var liste []selecteur.ChoixPolice
	connue := false
	for _, p := range installation.Polices {
		liste = append(liste, selecteur.ChoixPolice{Nom: p.Nom, Titre: p.Titre, Famille: p.Famille, Installee: o.PoliceInstallee(p)})
		connue = connue || p.Famille == ap.Avant
	}
	if ap.Avant != "" && !connue {
		liste = append([]selecteur.ChoixPolice{{Titre: ap.Avant, Famille: ap.Avant, Installee: true}}, liste...)
	}
	m := selecteur.NouvellesPolices(environnement(), liste, ap.Terminal, ap.Avant, ap.Essayer)
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil || !m.Garder {
		if e := ap.Remettre(); e != nil {
			return a.echec("%v", e)
		}
		if err != nil {
			return a.echec("%v", err)
		}
		fmt.Fprint(a.sortie, i18n.T("Police remise : %s (%s).\n", montrePolice(ap.Avant), ap.Terminal))
		return 0
	}
	ap.Garder()
	if m.Choix.Nom == "" { // la police d'avant, hors de la liste d'agentline : rien d'autre à régler
		fmt.Fprint(a.sortie, i18n.T("Police gardée : %s (%s).\n", m.Choix.Famille, ap.Terminal))
		return 0
	}
	o.Police, o.PoliceImposee = m.Choix.Nom, true
	if err := o.AppliquerPolice(); err != nil {
		return a.echec("%v", err)
	}
	return 0
}

func montrePolice(f string) string {
	if f == "" {
		return i18n.T("celle par défaut du terminal")
	}
	return f
}
