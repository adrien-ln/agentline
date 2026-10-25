// Package entree : choisit l'adaptateur d'entrée de l'outil qui appelle la statusline (Claude Code,
// Antigravity CLI, Copilot CLI), d'après l'option --from de « agentline render », sinon d'après le JSON reçu.
package entree

import (
	"bytes"

	"github.com/adrien-ln/agentline/internal/entree/antigravity"
	"github.com/adrien-ln/agentline/internal/entree/claude"
	"github.com/adrien-ln/agentline/internal/entree/copilot"
	"github.com/adrien-ln/agentline/internal/etat"
)

// Outils : les outils pris en charge, dans l'ordre de l'aide (leur nom pour --from).
var Outils = []string{claude.Harness, antigravity.Harness, copilot.Harness}

// Lire : le JSON de l'outil -> la session ; outil vide ou inconnu : reconnu d'après le JSON.
func Lire(j []byte, outil string, now int64) etat.Session {
	if !connu(outil) {
		outil = Detecter(j)
	}
	switch outil {
	case antigravity.Harness:
		return antigravity.Lire(j, now)
	case copilot.Harness:
		return copilot.Lire(j)
	}
	return claude.Lire(j)
}

func connu(outil string) bool {
	for _, o := range Outils {
		if o == outil {
			return true
		}
	}
	return false
}

// Detecter : l'outil d'après des clés que lui seul envoie ; Claude Code par défaut (c'était le seul).
func Detecter(j []byte) string {
	a := func(cle string) bool { return bytes.Contains(j, []byte(`"`+cle+`"`)) }
	switch {
	case a("agent_state") || a("conversation_id") || a("artifact_count") || a("plan_tier"):
		return antigravity.Harness
	case a("total_premium_requests") || a("current_context_used_percentage") || a("displayed_context_limit"):
		return copilot.Harness
	}
	return claude.Harness
}
