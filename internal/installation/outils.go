package installation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Les autres outils dont la statusline peut lancer agentline, en plus de Claude Code. Un outil est
// configuré seulement s'il est installé (son dossier de réglages existe) et après un oui : sa clé
// "statusLine" lance « agentline render --from <outil> », copie .bak avant, comme pour Claude Code.

// outil : un outil, son nom affiché, sa valeur pour --from, son fichier de réglages et ce qu'il faut
// encore faire de son côté.
type outil struct {
	nom, from string
	reglages  func(o Options) string
	note      string // à faire dans l'outil (en français, traduit à l'affichage), vide si rien
}

var outils = []outil{
	{nom: "Antigravity CLI", from: "antigravity",
		reglages: func(o Options) string { return filepath.Join(o.Home, ".gemini", "antigravity-cli", "settings.json") }},
	{nom: "Copilot CLI", from: "copilot",
		reglages: func(o Options) string {
			if d := o.env("COPILOT_HOME"); d != "" {
				return filepath.Join(d, "settings.json")
			}
			return filepath.Join(o.Home, ".copilot", "settings.json")
		},
		note: i18n.M("la statusline de Copilot CLI est expérimentale : active-la avec /experimental, puis relance-le")},
}

// present : l'outil est installé sur cette machine (son dossier de réglages existe).
func (t outil) present(o Options) bool {
	_, err := os.Stat(filepath.Dir(t.reglages(o)))
	return err == nil
}

// commande : ce que lance la statusline de l'outil.
func (t outil) commande(o Options) string { return o.Commande() + " --from " + t.from }

// lanceAgentline : la commande de statusLine est-elle celle d'agentline (pour ne retirer que la nôtre) ?
func lanceAgentline(c string) bool {
	return strings.Contains(c, "agentline") && strings.Contains(c, " render")
}

// commandeDe : la commande de statusLine dans un fichier de réglages ; faux si elle manque ou si le
// fichier est illisible.
func commandeDe(f string) (string, bool) {
	b, err := os.ReadFile(f)
	if err != nil {
		return "", false
	}
	var r struct {
		StatusLine *struct {
			Command string `json:"command"`
		} `json:"statusLine"`
	}
	if json.Unmarshal(b, &r) != nil || r.StatusLine == nil {
		return "", false
	}
	return r.StatusLine.Command, true
}

// configurerOutils : pour chaque outil installé, propose de faire lancer agentline par sa statusline ;
// sans terminal pour répondre, dit seulement comment le faire.
func (o Options) configurerOutils() error {
	for _, t := range outils {
		if !t.present(o) {
			continue
		}
		f := t.reglages(o)
		if c, ok := commandeDe(f); ok && c == t.commande(o) {
			o.etape(i18n.T("À jour"), f)
			continue
		}
		if !o.Interactif || o.Demander == nil {
			o.etape(t.nom, i18n.T("détecté : lance « agentline install » dans un terminal pour y mettre la statusline"))
			continue
		}
		if !o.Demander(i18n.T("%s est installé : y mettre la statusline d'agentline ? [O/n] ", t.nom)) {
			continue
		}
		res, copie, err := o.modifier(f, func(c string) (string, string) { return PoserStatusLine(c, BlocStatusLine(t.commande(o))) }, jsonValide)
		if err != nil {
			return err
		}
		o.etape(t.nom, i18n.T("statusLine %s dans %s%s : %s", i18n.T(res), f, avecCopie(copie), t.commande(o)))
		if t.note != "" {
			o.etape(t.nom, i18n.T(t.note))
		}
	}
	return nil
}

// retirerOutils : retire la statusline des autres outils, seulement là où elle lance agentline.
func (o Options) retirerOutils() error {
	for _, t := range outils {
		f := t.reglages(o)
		if c, ok := commandeDe(f); !ok || !lanceAgentline(c) {
			continue
		}
		res, copie, err := o.modifier(f, func(c string) (string, string) {
			n, ok := RetirerStatusLine(c)
			if !ok {
				return c, identique
			}
			return n, "retiré"
		}, jsonValide)
		if err != nil {
			return err
		}
		if res != identique {
			o.etape(t.nom, i18n.T("statusLine retirée de %s%s", f, avecCopie(copie)))
		}
	}
	return nil
}

// verifierOutils : pour chaque outil installé, sa statusline lance-t-elle agentline ?
func (o Options) verifierOutils() []Verification {
	var v []Verification
	for _, t := range outils {
		if !t.present(o) {
			continue
		}
		f := t.reglages(o)
		attendu := `"` + filepath.ToSlash(o.binaireInstalle()) + `" render --from ` + t.from
		switch c, ok := commandeDe(f); {
		case !ok:
			v = append(v, Verification{Avertissement, t.nom, i18n.T("installé, sans la statusline d'agentline"), "agentline install"})
		case c != attendu:
			v = append(v, Verification{Avertissement, t.nom, i18n.T("statusLine lance « %s », pas %s", c, attendu), "agentline install"})
		default:
			v = append(v, Verification{Niveau: Bon, Sujet: t.nom, Detail: i18n.T("statusLine : %s", c)})
		}
	}
	return v
}
