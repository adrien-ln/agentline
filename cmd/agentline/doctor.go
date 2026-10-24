package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/installation"
	"github.com/adrien-ln/agentline/internal/theme"
)

// commandeDoctor : « agentline doctor » ; 0 si rien n'est à corriger, 1 sinon. Ne modifie rien (sauf la copie
// des anciens réglages, faite au premier accès aux réglages, quel qu'il soit).
func commandeDoctor(a *appel) int {
	o := optionsReelles(a)
	v := installation.Diagnostiquer(o)
	if m, ok := verifierMigration(); ok {
		v = append(v, m)
	}
	v = append(v, reglagesOrphelins(sources())...)
	v = append(v, verifierCouleurs(getenv), verifierFond(sources(), getenv, detecterFond))
	return installation.Afficher(a.sortie, v)
}

// verifierFond : le fond retenu et sa source ; ⚠ si le terminal, interrogé, en a un autre.
func verifierFond(s theme.Sources, getenv func(string) string, detecter func() (ansi.Fond, bool)) installation.Verification {
	f, source := s.Fond(getenv)
	v := installation.Verification{Niveau: installation.Bon, Sujet: i18n.T("Fond"), Detail: nomFond(f) + " (" + sourceFond(s, source) + ")"}
	if d, ok := detecter(); ok && d != f {
		v.Niveau = installation.Avertissement
		v.Detail += i18n.T(" : le terminal a un fond %s", nomFond(d))
		v.Correction = "agentline theme background " + d.String()
		if source == theme.FondVariable {
			v.Correction = "AGENTLINE_BACKGROUND=" + d.String()
		}
	}
	return v
}

// --- Textes visibles des couleurs (doctor) ---

// verifierCouleurs : le niveau de couleurs retenu et sa raison. Doctor tourne dans le terminal, la
// statusline sous Claude Code : elle hérite du même environnement s'il a été lancé depuis ce terminal.
func verifierCouleurs(getenv func(string) string) installation.Verification {
	n, raison, _ := niveauCouleurs(getenv)
	noms := map[ansi.Niveau]string{ansi.Truecolor: i18n.T("24 bits"), ansi.Couleurs256: i18n.T("256 couleurs"),
		ansi.Couleurs16: i18n.T("16 couleurs"), ansi.SansCouleur: i18n.T("sans couleur")}
	detail := noms[n] + " (" + i18n.T(raison) + ")" // raison : un nom de variable, ou « défaut »
	if raison == "SSH" && n != ansi.SansCouleur {
		detail = i18n.T("couleurs ANSI fixes de la ligne courte (SSH)")
	}
	v := installation.Verification{Niveau: installation.Bon, Sujet: i18n.T("Couleurs"), Detail: detail,
		Correction: i18n.T("pour en changer : AGENTLINE_COLORS=truecolor, 256, 16 ou none")}
	if s := getenv("AGENTLINE_COLORS"); s != "" {
		if _, ok := ansi.LireNiveau(s); !ok {
			v.Niveau, v.Detail = installation.Avertissement, i18n.T("AGENTLINE_COLORS=%s inconnue, ignorée : %s", s, v.Detail)
		}
	} else if n != ansi.Truecolor && raison != "NO_COLOR" && raison != "SSH" {
		v.Niveau = installation.Avertissement // deviné : à confirmer par l'utilisateur
		v.Detail += i18n.T(" : si le terminal affiche le 24 bits, AGENTLINE_COLORS=truecolor")
		v.Correction = ""
	}
	return v
}

// reglagesOrphelins : les réglages ($AGENTLINE_CONFIG) qui désignent un thème qui n'existe pas (renommé,
// supprimé) ; la statusline les ignore et passe à l'étape suivante de la chaîne de priorité.
func reglagesOrphelins(s theme.Sources) []installation.Verification {
	existe := func(n string) bool { _, ok := s.Fichier(n); return ok }
	suggestion := func(n string) string {
		if p := s.Proche(n); p != "" {
			return p
		}
		return "tokyo"
	}
	var v []installation.Verification
	probleme := func(detail, correction string) {
		v = append(v, installation.Verification{Niveau: installation.Probleme, Sujet: i18n.T("Réglages"), Detail: detail, Correction: correction})
	}
	if b, err := os.ReadFile(filepath.Join(s.Config, "defaut")); err == nil {
		if n := strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0]); n != "" && !existe(n) {
			probleme(i18n.T("thème par défaut inconnu : %s", n), "agentline theme set "+suggestion(n))
		}
	}
	for _, p := range s.Associations() {
		if !existe(p[1]) {
			c := i18n.T("dans un clone de %s : agentline theme link %s (ou agentline theme unlink)", p[0], suggestion(p[1]))
			if filepath.IsAbs(p[0]) || strings.HasPrefix(p[0], "/") {
				c = `agentline theme unlink "` + p[0] + `"`
			}
			probleme(i18n.T("projet %s : thème inconnu %s", p[0], p[1]), c)
		}
	}
	for _, b := range s.ReglesBranches() {
		if !existe(b[1]) {
			probleme(i18n.T("branche %s : thème inconnu %s", b[0], b[1]), `agentline theme unbranch "`+b[0]+`"`)
		}
	}
	if len(v) == 0 {
		v = append(v, installation.Verification{Niveau: installation.Bon, Sujet: i18n.T("Réglages"), Detail: i18n.T("aucun thème inconnu dans %s", s.Config)})
	}
	return v
}
