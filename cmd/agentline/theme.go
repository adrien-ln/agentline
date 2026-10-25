package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/render"
	"github.com/adrien-ln/agentline/internal/selecteur"
	"github.com/adrien-ln/agentline/internal/theme"
)

// commandesTheme : les sous-commandes de « agentline theme », dans l'ordre de l'aide. Les verbes sont ceux des
// versions précédentes (compatibilité).
func commandesTheme() []commande {
	return []commande{
		{nom: "choisir", aide: i18n.T("le sélecteur interactif (ce que fait « agentline theme » seul)"), cachee: true, faire: themeChoisir},
		{nom: "list", alias: []string{"ls"}, aide: i18n.T("thèmes disponibles (● : celui en vigueur ici)"), faire: themeListe},
		{nom: "current", args: i18n.T("[--why] [dossier]"), aide: i18n.T("thème en vigueur et d'où il vient (--why : toute la chaîne)"), max: 1,
			options: []option{{nom: "--why", aide: i18n.T("toute la chaîne de priorité : ce qui est réglé à chaque niveau, ce qui l'emporte")}}, faire: themeCourant},
		{nom: "set", args: i18n.T("<thème>"), aide: i18n.T("thème par défaut (nom ou titre)"), min: 1, max: 1, faire: themeDefaut},
		{nom: "link", args: i18n.T("<thème> [dossier]"), aide: i18n.T("associer un thème au projet (dépôt Git) du dossier"), min: 1, max: 2, faire: themeLier},
		{nom: "unlink", args: i18n.T("[dossier]"), aide: i18n.T("retirer l'association du projet"), max: 1, faire: themeDelier},
		{nom: "links", aide: i18n.T("projets associés"), faire: themeLiens},
		{nom: "branch", args: i18n.T("<motif> <thème>"), aide: i18n.T("thème des branches du motif (main, release/*), dans tous les projets"), min: 2, max: 2, faire: themeBranche},
		{nom: "unbranch", args: i18n.T("<motif>"), aide: i18n.T("retirer la règle"), min: 1, max: 1, faire: themeRetirerBranche},
		{nom: "branches", aide: i18n.T("règles de branche"), faire: themeBranches},
		{nom: "new", args: i18n.T("<nom> [--from <thème>]"), aide: i18n.T("créer un thème dans l'éditeur, à partir de celui en vigueur ici (ou de --from)"), min: 1, max: 1,
			options: []option{{nom: "--from", valeur: "theme", aide: i18n.T("le thème de départ")}}, faire: themeNouveau},
		{nom: "edit", args: i18n.T("<thème>"), aide: i18n.T("modifier un thème dans l'éditeur (un thème intégré : une copie)"), min: 1, max: 1, faire: themeEditer},
		{nom: "import", args: i18n.T("<fichier> | --terminal"), aide: i18n.T("un jeu de couleurs de terminal (Windows Terminal, iTerm2, base16) ouvert dans l'éditeur"), max: 1,
			options: []option{{nom: "--scheme", valeur: "name", aide: i18n.T("le jeu à prendre dans un settings.json qui en a plusieurs")},
				{nom: "--terminal", aide: i18n.T("le jeu de couleurs du profil Windows Terminal en cours")}}, faire: themeImporter},
		{nom: "export", args: i18n.T("<thème>"), aide: i18n.T("le fichier du thème, avec les noms anglais documentés, à partager"), min: 1, max: 1, faire: themeExporter},
		{nom: "preview", args: i18n.T("[thème]"), aide: i18n.T("aperçu réel de la statusline ici"), max: 1, faire: themeApercu},
		{nom: "background", args: "[dark|light|auto]", aide: i18n.T("fond du terminal, clair ou sombre (sans argument : celui retenu)"), max: 1, faire: themeFond},
		{nom: "help", alias: []string{"aide"}, aide: i18n.T("cette aide"), faire: func(a *appel) int { io.WriteString(a.sortie, aideTheme()); return 0 }},
	}
}

// aideTheme : les sous-commandes, la priorité et l'emplacement des réglages.
func aideTheme() string {
	return aideTable(commandesTheme(), "agentline theme") +
		i18n.T("Priorité : AGENTLINE_THEME > branche > projet > défaut > AGENTLINE_PALETTE > tokyo (agentline theme current --why).\n") +
		i18n.T("Réglages : $AGENTLINE_CONFIG (par défaut ~/.config/agentline). Thèmes perso : $AGENTLINE_CONFIG/themes/<nom>.theme\n")
}

// commandeTheme : « agentline theme … » ; renvoie le code de sortie.
func commandeTheme(args []string, sortie, erreurs io.Writer) int {
	if len(args) == 0 {
		args = []string{"choisir"}
	}
	if estAide(args[0]) {
		io.WriteString(sortie, aideTheme())
		return 0
	}
	c, ok := chercher(commandesTheme(), args[0])
	if !ok {
		fmt.Fprint(erreurs, i18n.T("agentline theme : commande inconnue : %s (voir « agentline theme help »)\n", args[0]))
		return codeUsage
	}
	return lancer(c, "agentline theme", args[1:], sortie, erreurs)
}

func sources() theme.Sources { return environnement().Themes }

// cheminArg : le dossier en argument (absolu), ou le dossier courant, et s'il existe.
func (a *appel) cheminArg(i int) (string, bool) {
	if len(a.args) <= i {
		d, _ := os.Getwd()
		return d, true
	}
	d, err := filepath.Abs(a.args[i])
	if err != nil {
		d = a.args[i]
	}
	fi, err := os.Stat(d)
	return d, err == nil && fi.IsDir()
}

// dossierArg : comme cheminArg, avec une erreur d'usage si le dossier n'existe pas.
func (a *appel) dossierArg(i int) (string, bool) {
	d, ok := a.cheminArg(i)
	if !ok {
		a.usage(i18n.T("dossier introuvable : %s"), a.args[i])
	}
	return d, ok
}

// themeArg : le thème en argument, par son nom ou son titre ; sinon une erreur d'usage, avec une suggestion.
func (a *appel) themeArg(s theme.Sources, saisie string) (string, bool) {
	if n, ok := s.Trouver(saisie); ok {
		return n, true
	}
	if p := s.Proche(saisie); p != "" {
		t, _ := s.Entete(p)
		a.usage(i18n.T("thème inconnu : %s (tu voulais dire « %s », %s ? voir « agentline theme list »)"), saisie, p, t)
	} else {
		a.usage(i18n.T("thème inconnu : %s (voir « agentline theme list »)"), saisie)
	}
	return "", false
}

// titre : « Tokyo Night (tokyo) ».
func titre(s theme.Sources, nom string) string {
	t, _ := s.Entete(nom)
	if t == "" || t == nom {
		return nom
	}
	return t + " (" + nom + ")"
}

// reussi : « ✓ cible → thème », puis l'avertissement si ce réglage reste masqué dans le dossier.
func (a *appel) reussi(s theme.Sources, cible, nom, source, dossier string) int {
	fmt.Fprintf(a.sortie, "✓ %s → %s\n", cible, titre(s, nom))
	if m := avertissement(s, nom, source, dossier, getenv); m != "" {
		fmt.Fprintln(a.sortie, m)
	}
	return 0
}

// avertissement : « ⚠ … » si un thème réglé à cette source ne sera pas celui en vigueur dans le dossier.
func avertissement(s theme.Sources, nom, source, dossier string, getenv func(string) string) string {
	ch := s.Resoudre(dossier, theme.BrancheDe(dossier), getenv)
	m, ok := theme.MasquePar(ch, source)
	if !ok {
		if act := theme.Active(ch); act.Source == source && act.Nom != nom && source == theme.SourceBranche {
			// une règle plus précise s'applique à la branche d'ici
			return i18n.T("⚠ ici, la règle %s (plus précise) l'emporte : %s reste en vigueur", act.Motif, titre(s, act.Nom))
		}
		return ""
	}
	qui := m.Libelle()
	switch m.Source {
	case theme.SourceForce:
		qui = "AGENTLINE_THEME=" + m.Nom
	case theme.SourceBranche:
		qui = i18n.T("la %s", qui)
	case theme.SourceProjet:
		qui = i18n.T("le %s", qui)
	}
	return i18n.T("⚠ masqué ici par %s : %s reste en vigueur (agentline theme current --why)", qui, titre(s, m.Nom))
}

func themeChoisir(a *appel) int { return lancerSelecteur(a, nil) }

// lancerSelecteur : le sélecteur plein écran (préparé par avant, s'il y a lieu), puis son message de sortie.
func lancerSelecteur(a *appel, avant func(*selecteur.Modele)) int {
	env := environnement()
	d, _ := os.Getwd()
	// Avant Bubble Tea (qui prend l'entrée) : le fond du terminal, s'il n'est pas déjà réglé
	detecte, aDetecte := ansi.Sombre, false
	if _, source := env.Themes.Fond(getenv); !fondImpose(source) {
		if detecte, aDetecte = detecterFond(); aDetecte {
			env.Couleurs.Fond = detecte
		}
	}
	m := selecteur.Nouveau(env.Themes, env, d, getenv)
	if avant != nil {
		avant(m)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return a.echec("%v", err)
	}
	fmt.Fprintln(a.sortie, m.Message())
	if aDetecte {
		if msg := enregistrerFondDetecte(env.Themes, detecte); msg != "" {
			fmt.Fprintln(a.sortie, msg)
		}
	}
	return 0
}

// themeNouveau : « agentline theme new <nom> [--from <thème>] » : l'éditeur sur une copie du thème de départ,
// sous ce nom ; s la crée.
func themeNouveau(a *appel) int {
	s := sources()
	nom := a.args[0]
	if !theme.NomValide(nom) {
		return a.usage(i18n.T("nom de thème invalide : %s (minuscules, chiffres et tirets)"), nom)
	}
	if _, pris := s.Fichier(nom); pris {
		return a.usage(i18n.T("un thème s'appelle déjà %s (agentline theme edit %s)"), nom, nom)
	}
	d, _ := os.Getwd()
	base := s.Choisir(d, getenv).Nom
	if v, ok := a.vals["--from"]; ok {
		if base, ok = a.themeArg(s, v); !ok {
			return codeUsage
		}
	}
	return lancerSelecteur(a, func(m *selecteur.Modele) { m.OuvrirEditeur(base, nom) })
}

// themeEditer : « agentline theme edit <thème> » : l'éditeur sur ce thème (un thème intégré : une copie).
func themeEditer(a *appel) int {
	nom, ok := a.themeArg(sources(), a.args[0])
	if !ok {
		return codeUsage
	}
	return lancerSelecteur(a, func(m *selecteur.Modele) { m.OuvrirEditeur(nom, "") })
}

// themeImporter : « agentline theme import <fichier> [--scheme <nom>] » ou « --terminal » : le jeu de couleurs
// devient un thème, ouvert dans l'éditeur pour le retoucher ; s le crée.
func themeImporter(a *appel) int {
	var j theme.JeuANSI
	var err error
	titre := ""
	switch {
	case a.opts["--terminal"]:
		if getenv("WT_SESSION") == "" {
			return a.echec("%s", i18n.T("--terminal lit le jeu de couleurs de Windows Terminal : lance-le dans Windows Terminal, ou donne le fichier du jeu"))
		}
		f := ""
		for _, r := range optionsReelles(a).PolicesReglees() {
			if r.Terminal == "Windows Terminal" && f == "" {
				f = r.Fichier
			}
		}
		b, e := os.ReadFile(f)
		if e != nil {
			return a.echec("%v", e)
		}
		j, err = theme.JeuDuProfil(b, getenv("WT_PROFILE_ID"))
	case len(a.args) == 1:
		b, e := os.ReadFile(a.args[0])
		if e != nil {
			return a.echec("%v", e)
		}
		j, err = theme.LireJeu(b, a.vals["--scheme"])
		titre = strings.TrimSuffix(filepath.Base(a.args[0]), filepath.Ext(a.args[0]))
	default:
		return a.usage("%s", i18n.T("il faut un fichier de jeu de couleurs, ou --terminal"))
	}
	if err != nil {
		return a.echec("%v", err)
	}
	if j.Nom != "" {
		titre = j.Nom
	}
	if titre == "" {
		titre = i18n.T("Terminal")
	}
	return lancerSelecteur(a, func(m *selecteur.Modele) { m.OuvrirImport(theme.DepuisJeu(j), titre) })
}

// themeExporter : « agentline theme export <thème> » : son fichier, avec les noms anglais documentés.
func themeExporter(a *appel) int {
	s := sources()
	nom, ok := a.themeArg(s, a.args[0])
	if !ok {
		return codeUsage
	}
	p, _ := s.Charger(nom)
	titre, tags := s.Entete(nom)
	a.sortie.Write(theme.Format(p, titre, tags))
	return 0
}

func themeListe(a *appel) int {
	s := sources()
	d, _ := os.Getwd()
	c := s.Choisir(d, getenv)
	for _, n := range s.Noms() {
		t, _ := s.Entete(n)
		marque := " "
		if n == c.Nom {
			marque = "●"
		}
		fmt.Fprintf(a.sortie, "%s %-22s %s\n", marque, n, t)
	}
	return 0
}

// court : la source de l'étape, comme l'affiche « current » (« projet <id> », « branche <nom> », « défaut »).
func court(e theme.Etape) string {
	switch e.Source {
	case theme.SourceProjet, theme.SourceBranche:
		return theme.NomSource(e.Source) + " " + e.Cle
	}
	return theme.NomSource(e.Source)
}

func themeCourant(a *appel) int {
	d, ok := a.dossierArg(0)
	if !ok {
		return codeUsage
	}
	s := sources()
	ch := s.Resoudre(d, theme.BrancheDe(d), getenv)
	act := theme.Active(ch)
	fmt.Fprintf(a.sortie, "%s (%s)\n", act.Nom, court(act))
	if !a.opts["--why"] {
		return 0
	}
	gauche := 0
	for _, e := range ch {
		gauche = max(gauche, len([]rune(e.Libelle())))
	}
	for k, e := range ch {
		marque, etat := " ", ""
		switch {
		case e.Active:
			marque, etat = "●", titre(s, e.Nom)
		case e.Nom == "":
			etat = "—"
		case !e.Existe:
			etat = i18n.T("%s : thème introuvable, ignoré", e.Nom)
		default:
			etat = i18n.T("%s : masqué", titre(s, e.Nom))
		}
		l := e.Libelle()
		fmt.Fprintf(a.sortie, "%s %d. %s%s  %s\n", marque, k+1, l, strings.Repeat(" ", gauche-len([]rune(l))), etat)
	}
	return 0
}

func themeDefaut(a *appel) int {
	s := sources()
	nom, ok := a.themeArg(s, a.args[0])
	if !ok {
		return codeUsage
	}
	if err := s.EcrireDefaut(nom); err != nil {
		return a.echec("%v", err)
	}
	d, _ := os.Getwd()
	return a.reussi(s, i18n.T("thème par défaut"), nom, theme.SourceDefaut, d)
}

func themeLier(a *appel) int {
	s := sources()
	nom, ok := a.themeArg(s, a.args[0])
	if !ok {
		return codeUsage
	}
	d, ok := a.dossierArg(1)
	if !ok {
		return codeUsage
	}
	id := s.ProjetID(d)
	if err := s.Associer(id, nom); err != nil {
		return a.echec("%v", err)
	}
	return a.reussi(s, i18n.T("projet %s", id), nom, theme.SourceProjet, d)
}

func themeDelier(a *appel) int {
	s := sources()
	d, existe := a.cheminArg(0) // un dossier disparu : son association (par chemin) peut encore être retirée
	id := s.ProjetID(d)
	ancien := s.ThemeDuProjet(id)
	if ancien == "" {
		if !existe {
			return a.usage(i18n.T("dossier introuvable : %s"), a.args[0])
		}
		fmt.Fprint(a.sortie, i18n.T("Aucun thème associé au projet %s\n", id))
		return 0
	}
	if err := s.Associer(id, ""); err != nil {
		return a.echec("%v", err)
	}
	fmt.Fprint(a.sortie, i18n.T("✓ projet %s : association à %s retirée\n", id, titre(s, ancien)))
	return 0
}

func themeLiens(a *appel) int {
	l := sources().Associations()
	if len(l) == 0 {
		fmt.Fprintln(a.sortie, i18n.T("Aucun projet associé."))
	}
	for _, x := range l {
		fmt.Fprintf(a.sortie, "%-50s %s\n", x[0], x[1])
	}
	return 0
}

func themeBranche(a *appel) int {
	s := sources()
	motif := a.args[0]
	if !theme.MotifValide(motif) {
		return a.usage(i18n.T("motif invalide : %q"), motif)
	}
	nom, ok := a.themeArg(s, a.args[1])
	if !ok {
		return codeUsage
	}
	if err := s.RegleBranche(motif, nom); err != nil {
		return a.echec("%v", err)
	}
	fmt.Fprint(a.sortie, i18n.T("✓ branche %s → %s\n", motif, titre(s, nom)))
	d, _ := os.Getwd()
	if theme.Correspond(motif, theme.BrancheDe(d)) { // la règle s'applique ici : est-elle visible ?
		if m := avertissement(s, nom, theme.SourceBranche, d, getenv); m != "" {
			fmt.Fprintln(a.sortie, m)
		}
	} else if f := getenv("AGENTLINE_THEME"); f != "" {
		if _, ok := s.Fichier(f); ok {
			fmt.Fprintln(a.sortie, i18n.T("⚠ AGENTLINE_THEME=%s l'emporte partout tant qu'elle est définie", f))
		}
	}
	return 0
}

func themeRetirerBranche(a *appel) int {
	s := sources()
	motif := a.args[0]
	for _, r := range s.ReglesBranches() {
		if r[0] == motif {
			if err := s.RegleBranche(motif, ""); err != nil {
				return a.echec("%v", err)
			}
			fmt.Fprint(a.sortie, i18n.T("✓ branche %s : règle %s retirée\n", motif, titre(s, r[1])))
			return 0
		}
	}
	fmt.Fprint(a.sortie, i18n.T("Aucune règle pour %s\n", motif))
	return 0
}

func themeBranches(a *appel) int {
	r := sources().ReglesBranches()
	if len(r) == 0 {
		fmt.Fprintln(a.sortie, i18n.T("Aucune règle de branche."))
	}
	for _, x := range r {
		fmt.Fprintf(a.sortie, "%-30s %s\n", x[0], x[1])
	}
	return 0
}

func themeApercu(a *appel) int {
	env := environnement()
	s := env.Themes
	d, _ := os.Getwd()
	nom := s.Choisir(d, getenv).Nom
	if len(a.args) > 0 {
		var ok bool
		if nom, ok = a.themeArg(s, a.args[0]); !ok {
			return codeUsage
		}
	}
	e := env
	e.Getenv = func(k string) string {
		if k == "AGENTLINE_THEME" {
			return nom
		}
		return ""
	}
	if e.Columns == 0 {
		e.Columns = 120
	}
	io.WriteString(a.sortie, render.RenderSession(etat.Exemple(e.Now, filepath.ToSlash(d), 42), e))
	return 0
}
