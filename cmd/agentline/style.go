package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/selecteur"
	"github.com/adrien-ln/agentline/internal/style"
)

// commandesStyle : les sous-commandes de « agentline style », dans l'ordre de l'aide.
func commandesStyle() []commande {
	return []commande{
		{nom: "show", aide: i18n.T("chaque option, sa valeur en vigueur et d'où elle vient"), faire: styleMontrer},
		{nom: "keys", aide: i18n.T("les options, leurs valeurs permises et leur défaut"), faire: styleCles},
		{nom: "set", args: i18n.T("[--project] <option> <valeur>"), aide: i18n.T("régler une option (valeur vide : \"\") ; --project : pour le projet d'ici"), brut: true, faire: styleRegler},
		{nom: "unset", args: "<option>", aide: i18n.T("remettre une option à son défaut (--project : à la valeur générale)"), min: 1, max: 1, options: []option{optionProjet()}, faire: styleRetirer},
		{nom: "reset", aide: i18n.T("tout remettre par défaut (--project : retirer le style du projet)"), options: []option{optionProjet()}, faire: styleDefaut},
		{nom: "hide", args: "<item>", aide: i18n.T("masquer un item partout (--project : dans ce projet)"), min: 1, max: 1, options: []option{optionProjet()}, faire: func(a *appel) int { return styleMasquer(a, true) }},
		{nom: "unhide", args: "<item>", aide: i18n.T("afficher de nouveau un item masqué"), min: 1, max: 1, options: []option{optionProjet()}, faire: func(a *appel) int { return styleMasquer(a, false) }},
		{nom: "preview", args: "[--ssh]", aide: i18n.T("aperçu réel de la statusline ici (--ssh : la ligne courte)"),
			options: []option{{nom: "--ssh", aide: i18n.T("la ligne courte, comme en SSH")}}, faire: styleApercu},
		{nom: "help", alias: []string{"aide"}, aide: i18n.T("cette aide"), faire: func(a *appel) int { io.WriteString(a.sortie, aideStyle()); return 0 }},
	}
}

// aideStyle : les sous-commandes et l'emplacement du fichier.
func aideStyle() string {
	return ligneAide("agentline style", i18n.T("le sélecteur, sur l'onglet Statusline")) + aideTable(commandesStyle(), "agentline style") +
		i18n.T("Items : %s.\n", strings.Join(style.Items, ", ")) +
		i18n.T("Réglages : $AGENTLINE_CONFIG/style (seuls les écarts au défaut), et $AGENTLINE_CONFIG/styles/ pour les projets ; AGENTLINE_STYLE=default les ignore.\n")
}

// commandeStyle : « agentline style … » ; renvoie le code de sortie.
func commandeStyle(args []string, sortie, erreurs io.Writer) int {
	if len(args) == 0 {
		return styleChoisir(sortie, erreurs)
	}
	if estAide(args[0]) {
		io.WriteString(sortie, aideStyle())
		return 0
	}
	c, ok := chercher(commandesStyle(), args[0])
	if !ok {
		fmt.Fprint(erreurs, i18n.T("agentline style : commande inconnue : %s (voir « agentline style help »)\n", args[0]))
		return codeUsage
	}
	return lancer(c, "agentline style", args[1:], sortie, erreurs)
}

// styleFichier : le style écrit dans les réglages (même si AGENTLINE_STYLE=default).
func styleFichier() *style.Style { return style.LireFichier(dossierConfig()) }

// optionProjet : --project, le style du projet du dossier courant.
func optionProjet() option {
	return option{nom: "--project", aide: i18n.T("pour le projet du dossier courant (ses réglages par-dessus le style général)")}
}

// projetIci : l'identifiant du projet du dossier courant si --project, sinon vide (le style général).
func (a *appel) projetIci() string {
	if !a.opts["--project"] {
		return ""
	}
	d, _ := os.Getwd()
	return sources().ProjetID(d)
}

// stylePortee : le style à modifier : le général, ou pour un projet le général et le sien par-dessus.
func stylePortee(projet string) *style.Style {
	if projet == "" {
		return styleFichier()
	}
	return style.Superposer(styleFichier(), style.ChargerProjet(dossierConfig(), projet))
}

// ecrirePortee : le style écrit à sa portée (pour un projet : seulement ce qui diffère du général).
func (a *appel) ecrirePortee(projet string, s *style.Style) bool {
	if projet == "" {
		return a.ecrireStyle(s)
	}
	if err := style.EcrireProjet(dossierConfig(), projet, style.Ecart(s, styleFichier())); err != nil {
		a.echec("%v", err)
		return false
	}
	return true
}

// ici : la portée dans un message (« (ce projet) »), vide pour le style général.
func ici(projet string) string {
	if projet == "" {
		return ""
	}
	return i18n.T(" (ce projet)")
}

// avertir : les « ⚠ » qui disent pourquoi un réglage ne se verra pas ici.
func (a *appel) avertir(cle string) {
	if style.Force(getenv) {
		fmt.Fprintln(a.sortie, i18n.T("⚠ AGENTLINE_STYLE=default : le style est ignoré tant qu'elle est définie"))
	}
	if c, ok := style.Trouver(cle); ok && !c.SSH && courtIci() {
		fmt.Fprintln(a.sortie, i18n.T("⚠ en SSH, la ligne courte ne tient pas compte de %s (agentline style preview pour la version longue)", cle))
	}
}

func (a *appel) ecrireStyle(s *style.Style) bool {
	if err := style.Ecrire(dossierConfig(), s); err != nil {
		a.echec("%v", err)
		return false
	}
	return true
}

// cleArg : l'option de ce nom, sinon une erreur d'usage avec des suggestions.
func (a *appel) cleArg(nom string) (style.Cle, bool) {
	if c, ok := style.Trouver(nom); ok {
		return c, true
	}
	var proches []string
	for _, c := range style.Cles() {
		if strings.Contains(c.Nom, nom) || strings.HasPrefix(nom, strings.SplitN(c.Nom, ".", 2)[0]+".") {
			proches = append(proches, c.Nom)
		}
	}
	if len(proches) > 0 && len(proches) <= 6 {
		a.usage(i18n.T("option inconnue : %s (proches : %s ; voir « agentline style keys »)"), nom, strings.Join(proches, ", "))
	} else {
		a.usage(i18n.T("option inconnue : %s (voir « agentline style keys »)"), nom)
	}
	return style.Cle{}, false
}

// montre : une valeur affichable (vide : « (vide) »).
func montre(v string) string {
	if v == "" {
		return i18n.T("(vide)")
	}
	return v
}

func styleMontrer(a *appel) int {
	d, _ := os.Getwd()
	projet := sources().ProjetID(d)
	sp := style.ChargerProjet(dossierConfig(), projet)
	s := style.ChargerPour(dossierConfig(), projet, getenv)
	if style.Force(getenv) {
		fmt.Fprintln(a.sortie, i18n.T("⚠ AGENTLINE_STYLE=default : le fichier du style est ignoré, tout est au défaut"))
	}
	ssh := ligneCourte(s)
	for _, c := range style.Cles() {
		source := i18n.T("défaut")
		switch {
		case sp.Regle(c.Nom) && !style.Force(getenv):
			source = i18n.T("ce projet")
		case s.Regle(c.Nom):
			source = i18n.T("réglé")
		}
		if ssh && !c.SSH {
			source += i18n.T(", ignorée en SSH")
		}
		v := montre(s.Valeur(c.Nom))
		if strings.HasPrefix(c.Nom, "icon.") && s.Valeur(c.Nom) == style.Auto {
			v = "auto → " + montre(s.Glyphe(strings.TrimPrefix(c.Nom, "icon.")))
		} else if l := style.LibelleEnVigueur(c.Nom, s.Valeur(c.Nom)); l != s.Valeur(c.Nom) {
			v = "auto → " + montre(l)
		}
		fmt.Fprintf(a.sortie, "%-17s %s  (%s)\n", c.Nom, v, source)
	}
	return 0
}

// permis : les valeurs permises d'une option, en peu de mots.
func permis(c style.Cle) string {
	switch c.Genre {
	case style.Choix:
		return strings.Join(c.Valeurs, "|")
	case style.Nombre:
		return strconv.Itoa(c.Min) + "–" + strconv.Itoa(c.Max)
	case style.Libelle:
		return i18n.T("texte")
	case style.Glyphe:
		return i18n.T("glyphe")
	case style.Barre:
		return strings.Join(c.Valeurs, "|") + i18n.T("|3 car.")
	case style.Disposition:
		return i18n.T("items, « | »")
	case style.Branches:
		return i18n.T("branches")
	}
	return "items"
}

func styleCles(a *appel) int {
	for _, c := range style.Cles() {
		ssh := ""
		if c.SSH {
			ssh = i18n.T(" (aussi en SSH)")
		}
		fmt.Fprintf(a.sortie, i18n.T("%-17s %-26s défaut : %s\n%17s %s%s\n"), c.Nom, permis(c), montre(c.Defaut), "", c.Explication(), ssh)
	}
	return 0
}

// styleRegler : « set <option> <valeur…> », arguments pris tels quels (une valeur peut commencer par « - »).
func styleRegler(a *appel) int {
	if len(a.args) == 1 && estAide(a.args[0]) {
		io.WriteString(a.sortie, usageDe(a.cmd, a.prefixe))
		return 0
	}
	if len(a.args) > 0 && a.args[0] == "--project" { // set prend ses arguments tels quels : --project se lit à la main
		a.opts["--project"], a.args = true, a.args[1:]
	}
	if len(a.args) < 2 {
		return a.usage("%s", i18n.T("il manque l'option ou la valeur"))
	}
	c, ok := a.cleArg(a.args[0])
	if !ok {
		return codeUsage
	}
	v := strings.Join(a.args[1:], " ")
	projet := a.projetIci()
	s := stylePortee(projet)
	if err := s.Definir(c.Nom, v); err != nil {
		return a.usage(i18n.T("valeur refusée pour %s : %v"), c.Nom, err)
	}
	if !a.ecrirePortee(projet, s) {
		return codeEchec
	}
	defaut := ""
	if !s.Regle(c.Nom) {
		defaut = i18n.T(" (défaut)")
	}
	fmt.Fprintf(a.sortie, "✓ %s = %s%s%s\n", c.Nom, montre(s.Valeur(c.Nom)), defaut, ici(projet))
	a.avertir(c.Nom)
	return 0
}

func styleRetirer(a *appel) int {
	c, ok := a.cleArg(a.args[0])
	if !ok {
		return codeUsage
	}
	if projet := a.projetIci(); projet != "" {
		if !style.ChargerProjet(dossierConfig(), projet).Regle(c.Nom) {
			fmt.Fprint(a.sortie, i18n.T("%s n'est pas réglé pour ce projet\n", c.Nom))
			return 0
		}
		s := stylePortee(projet)
		s.Definir(c.Nom, styleFichier().Valeur(c.Nom))
		if !a.ecrirePortee(projet, s) {
			return codeEchec
		}
		fmt.Fprint(a.sortie, i18n.T("✓ %s : la valeur générale pour ce projet (%s)\n", c.Nom, montre(s.Valeur(c.Nom))))
		return 0
	}
	s := styleFichier()
	if !s.Regle(c.Nom) {
		fmt.Fprint(a.sortie, i18n.T("%s est déjà au défaut (%s)\n", c.Nom, montre(c.Defaut)))
		return 0
	}
	s.Retirer(c.Nom)
	if !a.ecrireStyle(s) {
		return codeEchec
	}
	fmt.Fprint(a.sortie, i18n.T("✓ %s : retour au défaut (%s)\n", c.Nom, montre(c.Defaut)))
	return 0
}

func styleDefaut(a *appel) int {
	if projet := a.projetIci(); projet != "" {
		n := style.ChargerProjet(dossierConfig(), projet).Nombre()
		if err := style.EcrireProjet(dossierConfig(), projet, &style.Surcharge{}); err != nil {
			return a.echec("%v", err)
		}
		fmt.Fprint(a.sortie, i18n.N(n, "✓ style du projet retiré (%d réglage) : le style général s'applique ici\n", "✓ style du projet retiré (%d réglages) : le style général s'applique ici\n", n))
		return 0
	}
	s := styleFichier()
	n := 0
	for _, c := range style.Cles() {
		if s.Regle(c.Nom) {
			n++
		}
	}
	if n == 0 {
		fmt.Fprintln(a.sortie, i18n.T("Le style est déjà celui par défaut."))
		return 0
	}
	if !a.ecrireStyle(style.Defaut()) {
		return codeEchec
	}
	fmt.Fprint(a.sortie, i18n.N(n, "✓ style par défaut rétabli (%d réglage retiré)\n", "✓ style par défaut rétabli (%d réglages retirés)\n", n))
	return 0
}

func styleMasquer(a *appel, oui bool) int {
	item, ok := style.NomItem(a.args[0]) // un ancien nom (dossier, cout…) est accepté
	if !ok {
		return a.usage(i18n.T("item inconnu : %s (items : %s)"), item, strings.Join(style.Items, ", "))
	}
	projet := a.projetIci()
	s := stylePortee(projet)
	if s.Masques()[item] == oui {
		if oui {
			fmt.Fprint(a.sortie, i18n.T("%s est déjà masqué\n", item))
		} else {
			fmt.Fprint(a.sortie, i18n.T("%s est affiché\n", item))
		}
		return 0
	}
	s.Masquer(item, oui)
	if !a.ecrirePortee(projet, s) {
		return codeEchec
	}
	if oui {
		fmt.Fprint(a.sortie, i18n.T("✓ %s masqué (agentline style unhide %s pour le retrouver)\n", item, item))
	} else {
		fmt.Fprint(a.sortie, i18n.T("✓ %s affiché\n", item))
		if !place(s, item) {
			fmt.Fprint(a.sortie, i18n.T("⚠ %s n'est dans aucune ligne (line1, line2, short) : agentline style set line2 \"… %s\"\n", item, item))
		}
	}
	a.avertir("hide")
	return 0
}

// place : l'item est-il dans une des dispositions ?
func place(s *style.Style, item string) bool {
	for _, l := range []string{"line1", "line2", "short"} {
		segs, _ := style.LireDisposition(s.Valeur(l))
		for _, seg := range segs {
			for _, n := range seg {
				if n == item {
					return true
				}
			}
		}
	}
	return false
}

func styleApercu(a *appel) int {
	e := environnement()
	d, _ := os.Getwd()
	e.SSH = a.opts["--ssh"]
	if e.Columns == 0 {
		e.Columns = 120
	}
	io.WriteString(a.sortie, render.RenderSession(etat.Exemple(e.Now, filepath.ToSlash(d), 42), e))
	return 0
}

// styleChoisir : « agentline style » seul, le sélecteur ouvert sur l'onglet Statusline.
func styleChoisir(sortie, erreurs io.Writer) int {
	env := environnement()
	d, _ := os.Getwd()
	m := selecteur.Nouveau(env.Themes, env, d, getenv)
	m.OuvrirStyle()
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(erreurs, "agentline style : %v\n", err)
		return codeEchec
	}
	fmt.Fprintln(sortie, m.Message())
	return 0
}
