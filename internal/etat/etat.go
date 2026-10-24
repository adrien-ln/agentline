// Package etat : la session telle que la statusline l'affiche, indépendante de l'outil (harness) qui
// l'envoie. Chaque adaptateur d'entrée (internal/entree/…) remplit une Session ; le rendu ne lit qu'elle.
//
// Parité : les nombres restent les chaînes brutes du JSON (vide : absent). Le rendu les convertit comme
// l'ancienne version bash (printf '%.0f', ${s%.*}, donc « 1.8e9 » donne 1 comme instant) ; un typage
// float ou int ici changerait ces cas limites.
package etat

import "strconv"

// Session : ce qu'une session d'agent expose à la statusline.
type Session struct {
	Harness string // outil d'origine : « claude », …

	Modele  string // nom affiché du modèle
	Effort  string // niveau d'effort : low, medium, high, xhigh, max (autre : affiché sans jauge)
	Rapide  bool   // mode rapide
	Session string // nom de la session
	Agent   string // nom de l'agent
	Style   string // style de sortie (« default » : pas affiché)

	Dossier  string // dossier courant, séparateurs « / »
	Projet   string // dossier du projet, séparateurs « / » (vide : inconnu)
	Worktree string // nom du worktree Git

	PR PR

	Contexte string // part de la fenêtre de contexte utilisée, en %
	Quotas   Quotas
	Cout     string // coût de la session en dollars

	Colonnes int // largeur du terminal si l'outil la donne (0 : inconnue) ; pas encore lue par le rendu
}

// PR : la demande de fusion de la branche.
type PR struct {
	Num   string // numéro (vide : aucune)
	URL   string
	Revue string // approved, changes_requested, pending, draft
}

// Quotas : les fenêtres d'utilisation de l'abonnement.
type Quotas struct {
	H5, J7 Quota // 5 heures, 7 jours
}

// Quota : une fenêtre d'utilisation.
type Quota struct {
	Utilise string // part utilisée, en % (vide : quota absent)
	Reset   string // instant de remise à zéro, en secondes Unix (vide : inconnu)
}

// Exemple : la session fictive des aperçus (sélecteur, « agentline theme preview ») à l'instant now, dans le
// dossier donné (séparateurs « / »), avec ctx % du contexte utilisé.
func Exemple(now int64, dossier string, ctx int) Session {
	return Session{
		Harness: "claude", Modele: "Opus 5.5", Effort: "high",
		Dossier: dossier, Projet: dossier,
		Contexte: strconv.Itoa(ctx),
		Quotas: Quotas{
			H5: Quota{Utilise: "61", Reset: strconv.FormatInt(now+3600, 10)},
			J7: Quota{Utilise: "23", Reset: strconv.FormatInt(now+300000, 10)},
		},
	}
}
