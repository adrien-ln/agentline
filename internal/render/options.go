package render

import (
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
)

// options : ce que l'utilisateur règle dans la statusline (internal/style), traduit pour le rendu. Les
// valeurs par défaut donnent le rendu de l'ancienne version bash.
type options struct {
	disp   disposition
	masque map[string]bool // items jamais affichés

	modeleCourt, modeleUnie bool   // premier mot du nom ; une seule couleur
	effort                  string // icon+text, icon, text
	gitResume, gitAmont     bool
	protegees               map[string]bool // branches où des changements non commités sont une alerte
	prEtat, prLien          bool
	sessionMax              int // longueur au-delà de laquelle le nom de session est coupé

	barre        [3]string // cases pleine, à moitié, éteinte
	sansBarre    bool
	largeurBarre int // nombre de cases
	ctxAlerte    int // seuil d'alerte du contexte, en %
	ctxLibelle   string

	quotaAlerte               int // seuil d'alerte des quotas, en %
	projection, plafond       bool
	reset                     string // alert, always, never
	lib5h, lib7j              string
	coutToujours, coutAnglais bool

	signal string // devant un segment en alerte
}

func optionsDefaut() options {
	return options{
		disp:         dispositionDefaut,
		effort:       "icon+text",
		gitAmont:     true,
		protegees:    branchesDe("main master"),
		prEtat:       true,
		prLien:       true,
		sessionMax:   40,
		barre:        [3]string{"━", "╸", "─"},
		largeurBarre: largeurBarre,
		ctxAlerte:    85,
		ctxLibelle:   "ctx",
		quotaAlerte:  90,
		projection:   true,
		plafond:      true,
		reset:        "alert",
		lib5h:        "5h",
		lib7j:        style.Libelle7j(),
		coutAnglais:  !i18n.Francais(),
		signal:       "●",
	}
}

// optionsDe : les options du style ; en SSH, seules les options qui valent pour la ligne courte
// (disposition short, hide, seuils, libellés, séparateur) sont prises, les autres gardent leur défaut.
func optionsDe(st *style.Style, ssh bool) options {
	if st.EstDefaut() {
		return optionsDefaut()
	}
	return lireOptions(st, ssh)
}

// lireOptions : optionsDe, option par option (sans le raccourci du style par défaut).
func lireOptions(st *style.Style, ssh bool) options {
	o := optionsDefaut()
	v := func(nom string) string {
		if c, _ := style.Trouver(nom); ssh && !c.SSH {
			return c.Defaut
		}
		return st.Valeur(nom)
	}
	lire := func(nom, sep string) ligne {
		segs, _ := style.LireDisposition(v(nom)) // valide : le style ne garde que des valeurs valides
		return ligne{segments: segs, sep: v(sep)}
	}
	o.disp = disposition{
		lignes: []ligne{lire("line1", "sep.line1"), lire("line2", "sep.line2")},
		court:  lire("short", "sep.short"),
	}
	if m := st.Masques(); len(m) > 0 {
		o.masque = m
	}
	o.modeleCourt, o.modeleUnie = v("model.name") == "short", v("model.color") == "solid"
	o.effort = v("effort.display")
	o.gitResume, o.gitAmont = v("git.status") == "summary", v("git.upstream") == style.Oui
	o.protegees = branchesDe(v("git.protected"))
	o.prEtat, o.prLien = v("pr.status") == style.Oui, v("pr.link") == style.Oui
	o.sessionMax = entierDe(v("session.max"))
	if ssh {
		o.barre, o.sansBarre = [3]string{"━", "╸", "─"}, false
	} else {
		c, ok := st.CasesBarre()
		o.barre, o.sansBarre = c, !ok
	}
	o.largeurBarre = entierDe(v("ctx.width"))
	o.ctxAlerte, o.ctxLibelle = entierDe(v("ctx.alert")), v("ctx.label")
	o.quotaAlerte = entierDe(v("quota.alert"))
	o.projection, o.plafond, o.reset = v("quota.projection") == style.Oui, v("quota.limit") == style.Oui, v("quota.reset")
	o.lib5h, o.lib7j = v("5h.label"), style.LibelleEnVigueur("7d.label", v("7d.label"))
	o.coutToujours, o.coutAnglais = v("cost.when") == "always", style.CoutAnglais(v("cost.format"))
	if !ssh {
		o.signal = st.Glyphe("signal")
	}
	return o
}

// branchesDe : « main master » -> l'ensemble de ces branches.
func branchesDe(s string) map[string]bool {
	b := map[string]bool{}
	for _, n := range strings.Fields(s) {
		b[n] = true
	}
	return b
}

func entierDe(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
