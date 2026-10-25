// Package antigravity : l'adaptateur d'entrée d'Antigravity CLI, du JSON envoyé à sa statusline
// (statusLine.command dans ~/.gemini/antigravity-cli/settings.json) vers etat.Session.
//
// Format d'après la documentation d'Antigravity CLI (antigravity.google/docs/cli/statusline) ; les noms des
// quotas (quota.<id>) n'y sont pas donnés : une fenêtre de 6 heures au plus compte pour « 5h », la plus
// longue des autres pour « 7d » (à vérifier sur une vraie session).
package antigravity

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/adrien-ln/agentline/internal/etat"
)

// Harness : la valeur de Session.Harness pour Antigravity CLI.
const Harness = "antigravity"

type entree struct {
	Cwd               string `json:"cwd"`
	ConversationTitle string `json:"conversation_title"`
	Model             *struct {
		DisplayName string `json:"display_name"`
		ID          string `json:"id"`
		Effort      string `json:"effort"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	ContextWindow struct {
		UsedPercentage json.Number `json:"used_percentage"`
	} `json:"context_window"`
	Agent *struct {
		Name string `json:"name"`
	} `json:"agent"`
	Quota map[string]struct {
		RemainingFraction *float64 `json:"remaining_fraction"`
		ResetTime         any      `json:"reset_time"`
		ResetInSeconds    *float64 `json:"reset_in_seconds"`
		Disabled          bool     `json:"disabled"`
	} `json:"quota"`
	Cost *struct {
		TotalUSD json.Number `json:"total_usd"`
	} `json:"cost"`
	TerminalWidth int `json:"terminal_width"`
}

// Lire : le JSON d'Antigravity CLI -> la session, à l'instant now (secondes Unix) pour les quotas donnés en
// secondes restantes. Un JSON illisible donne une session vide.
func Lire(j []byte, now int64) etat.Session {
	var e entree
	d := json.NewDecoder(bytes.NewReader(j))
	d.UseNumber()
	d.Decode(&e)
	s := etat.Session{
		Harness:  Harness,
		Session:  e.ConversationTitle,
		Dossier:  barres(e.Workspace.CurrentDir),
		Projet:   barres(e.Workspace.ProjectDir),
		Contexte: e.ContextWindow.UsedPercentage.String(),
		Colonnes: e.TerminalWidth,
	}
	if s.Dossier == "" {
		s.Dossier = barres(e.Cwd)
	}
	if e.Model != nil {
		s.Modele, s.Effort = e.Model.DisplayName, e.Model.Effort
		if s.Modele == "" {
			s.Modele = e.Model.ID
		}
	}
	if e.Agent != nil {
		s.Agent = e.Agent.Name
	}
	if e.Cost != nil {
		s.Cout = e.Cost.TotalUSD.String()
	}
	var courte, longue float64 // la plus courte des fenêtres de 6 h au plus, la plus longue des autres
	for _, q := range e.Quota {
		if q.Disabled || q.RemainingFraction == nil {
			continue
		}
		reset, dans := instant(q.ResetTime, q.ResetInSeconds, now)
		quota := etat.Quota{Utilise: strconv.FormatFloat((1-*q.RemainingFraction)*100, 'f', -1, 64), Reset: reset}
		switch {
		case dans > 0 && dans <= 6*3600:
			if courte == 0 || dans < courte {
				courte, s.Quotas.H5 = dans, quota
			}
		case dans > longue:
			longue, s.Quotas.J7 = dans, quota
		}
	}
	return s
}

// instant : la remise à zéro en secondes Unix (texte), et dans combien de secondes ; reset_time peut être un
// nombre de secondes Unix ou une date RFC 3339.
func instant(t any, dans *float64, now int64) (string, float64) {
	var u int64
	switch v := t.(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			u = int64(f)
		}
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			u = n
		} else if d, err := time.Parse(time.RFC3339, v); err == nil {
			u = d.Unix()
		}
	}
	if u == 0 && dans != nil {
		u = now + int64(*dans)
	}
	if u == 0 {
		return "", 0
	}
	return strconv.FormatInt(u, 10), float64(u - now)
}

func barres(s string) string { return strings.ReplaceAll(s, `\`, "/") }
