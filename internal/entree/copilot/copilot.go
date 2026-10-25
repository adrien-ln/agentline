// Package copilot : l'adaptateur d'entrée de GitHub Copilot CLI, du JSON envoyé à sa statusline
// (statusLine.command dans ~/.copilot/settings.json, fonction expérimentale STATUS_LINE) vers etat.Session.
//
// Format d'après les guides publiés autour de la fonction (pas encore de documentation officielle stable) :
// il peut changer. Copilot ne donne ni quotas en pourcentage ni coût en dollars : ces items restent vides.
package copilot

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/adrien-ln/agentline/internal/etat"
)

// Harness : la valeur de Session.Harness pour Copilot CLI.
const Harness = "copilot"

type entree struct {
	Cwd         string `json:"cwd"`
	SessionName string `json:"session_name"`
	Model       *struct {
		DisplayName string `json:"display_name"`
		ID          string `json:"id"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	ContextWindow struct {
		CurrentContextUsedPercentage json.Number `json:"current_context_used_percentage"`
		UsedPercentage               json.Number `json:"used_percentage"`
	} `json:"context_window"`
	Remote *struct {
		PullRequestNumber json.Number `json:"pull_request_number"`
		TaskURL           string      `json:"task_url"`
	} `json:"remote"`
}

// Lire : le JSON de Copilot CLI -> la session. Un JSON illisible donne une session vide.
func Lire(j []byte) etat.Session {
	var e entree
	d := json.NewDecoder(bytes.NewReader(j))
	d.UseNumber()
	d.Decode(&e)
	s := etat.Session{
		Harness: Harness,
		Session: e.SessionName,
		Dossier: strings.ReplaceAll(e.Workspace.CurrentDir, `\`, "/"),
		// la part affichée par /context dans Copilot, sinon celle du modèle
		Contexte: e.ContextWindow.CurrentContextUsedPercentage.String(),
	}
	if s.Dossier == "" {
		s.Dossier = strings.ReplaceAll(e.Cwd, `\`, "/")
	}
	if s.Contexte == "" {
		s.Contexte = e.ContextWindow.UsedPercentage.String()
	}
	if e.Model != nil {
		s.Modele = e.Model.DisplayName
		if s.Modele == "" {
			s.Modele = e.Model.ID
		}
	}
	if e.Remote != nil {
		s.PR.Num = e.Remote.PullRequestNumber.String()
	}
	return s
}
