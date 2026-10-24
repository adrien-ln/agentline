package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/theme"
)

// Le fond du terminal (clair ou sombre) : la statusline ne peut pas le demander au terminal (elle n'en a
// pas), le sélecteur et l'installation le peuvent, et l'enregistrent pour elle.

// delaiFond : le temps laissé au terminal pour répondre ; ceux qui répondent le font en quelques ms, la
// réponse à DA1 arrête l'attente plus tôt.
const delaiFond = 300 * time.Millisecond

// detecterFond : le fond du terminal, demandé par OSC 11 ; faux sans console ou sans réponse. Les tests
// la remplacent (TestMain) : aucun n'écrit dans un vrai terminal.
var detecterFond = func() (ansi.Fond, bool) {
	if !terminal(os.Stdin) || !terminal(os.Stdout) {
		return ansi.Sombre, false
	}
	r, ok := interrogerTerminal(ansi.RequeteFond, delaiFond, ansi.FinReponse)
	if !ok {
		return ansi.Sombre, false
	}
	return ansi.FondDeReponse(r)
}

// --- Textes visibles du fond (commande, sélecteur, installation, doctor) ---

func nomFond(f ansi.Fond) string {
	if f == ansi.Clair {
		return i18n.T("clair")
	}
	return i18n.T("sombre")
}

// sourceFond : d'où vient le fond, en peu de mots.
func sourceFond(s theme.Sources, source string) string {
	switch source {
	case theme.FondVariable:
		return "AGENTLINE_BACKGROUND"
	case theme.FondReglage:
		return i18n.T("réglage %s", s.Config+"/fond")
	case theme.FondCOLORFGBG:
		return "COLORFGBG"
	}
	return i18n.T("par défaut")
}

// fondImpose : vrai si le fond vient de l'utilisateur (variable ou réglage) : la détection ne le change pas.
func fondImpose(source string) bool {
	return source == theme.FondVariable || source == theme.FondReglage
}

// enregistrerFondDetecte : le fond détecté par le terminal, enregistré s'il n'y a ni variable ni réglage et
// s'il diffère de celui que la statusline prendrait sans lui ; renvoie le message à afficher (vide : rien
// de fait).
func enregistrerFondDetecte(s theme.Sources, f ansi.Fond) string {
	actuel, source := s.Fond(getenv)
	if fondImpose(source) || f == actuel {
		return ""
	}
	if err := s.EcrireFond(f, false); err != nil {
		return i18n.T("⚠ fond %s détecté, mais pas enregistré : %s", nomFond(f), err.Error())
	}
	return i18n.T("Fond %s détecté : enregistré pour la statusline (agentline theme background pour changer).", nomFond(f))
}

// themeFond : « agentline theme background [dark|light|auto] ».
func themeFond(a *appel) int {
	s := sources()
	if len(a.args) == 0 {
		f, source := s.Fond(getenv)
		fmt.Fprintf(a.sortie, "%s (%s)\n", f, sourceFond(s, source))
		if d, ok := detecterFond(); ok && d != f {
			fmt.Fprint(a.sortie, i18n.T("⚠ le terminal a un fond %s : agentline theme background %s\n", nomFond(d), d))
		}
		return 0
	}
	f, auto, ok := ansi.LireFond(a.args[0])
	if !ok || a.args[0] == "" {
		return a.usage(i18n.T("fond inconnu : %s (dark, light ou auto)"), a.args[0])
	}
	if err := s.EcrireFond(f, auto); err != nil {
		return a.echec("%v", err)
	}
	if auto {
		g, source := s.Fond(getenv)
		fmt.Fprint(a.sortie, i18n.T("✓ fond : détection automatique (%s, %s)\n", g, sourceFond(s, source)))
	} else {
		fmt.Fprint(a.sortie, i18n.T("✓ fond → %s\n", f))
	}
	if v := getenv("AGENTLINE_BACKGROUND"); v != "" {
		if g, eauto, ok := ansi.LireFond(v); ok && !(eauto && auto || !eauto && !auto && g == f) {
			fmt.Fprintln(a.sortie, i18n.T("⚠ AGENTLINE_BACKGROUND=%s l'emporte tant qu'elle est définie", v))
		}
	}
	return 0
}

// signalerFond : à l'installation, le fond du terminal détecté et enregistré (rien s'il est déjà réglé).
func signalerFond(s theme.Sources, sortie io.Writer) {
	if f, ok := detecterFond(); ok {
		if m := enregistrerFondDetecte(s, f); m != "" {
			fmt.Fprintln(sortie, m)
		}
	}
}
