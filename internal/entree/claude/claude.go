// Package claude : l'adaptateur d'entrée de Claude Code, du JSON envoyé à la statusline vers etat.Session.
package claude

import (
	"strings"

	"github.com/adrien-ln/agentline/internal/etat"
)

// Harness : la valeur de Session.Harness pour Claude Code.
const Harness = "claude"

// Lire : le JSON de Claude Code -> la session. Les valeurs sont gardées telles qu'écrites (voir etat) ;
// un JSON mal formé donne ce qui a pu en être lu.
func Lire(j []byte) etat.Session {
	s := string(j)
	ws := obj(s, "workspace")
	e := etat.Session{
		Harness:  Harness,
		Modele:   str(obj(s, "model"), "display_name"),
		Effort:   str(obj(s, "effort"), "level"),
		Rapide:   val(s, "fast_mode") == "true",
		Session:  str(s, "session_name"),
		Agent:    str(obj(s, "agent"), "name"),
		Style:    str(obj(s, "output_style"), "name"),
		Dossier:  strings.ReplaceAll(str(ws, "current_dir"), `\`, "/"),
		Projet:   strings.ReplaceAll(str(ws, "project_dir"), `\`, "/"),
		Worktree: str(ws, "git_worktree"),
		Contexte: val(obj(s, "context_window"), "used_percentage"),
		Cout:     val(obj(s, "cost"), "total_cost_usd"),
	}
	if n := str(obj(s, "worktree"), "name"); n != "" {
		e.Worktree = n
	}
	pr := obj(s, "pr")
	e.PR = etat.PR{Num: val(pr, "number"), URL: str(pr, "url"), Revue: str(pr, "review_state")}
	o := obj(s, "five_hour")
	e.Quotas.H5 = etat.Quota{Utilise: val(o, "used_percentage"), Reset: val(o, "resets_at")}
	o = obj(s, "seven_day")
	e.Quotas.J7 = etat.Quota{Utilise: val(o, "used_percentage"), Reset: val(o, "resets_at")}
	return e
}
