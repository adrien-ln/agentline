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
	iconeModele             string // model.icon : none, writing, weight
	effort                  string // icon+text, icon, text
	effort5                 bool   // effort.levels=5 : une icône par niveau
	gitResume, gitAmont     bool
	protegees               map[string]bool // branches où des changements non commités sont une alerte
	prEtat, prLien          bool
	etatPR                  map[string]string // pr.display=icon : l'icône de chaque état (icon.pr.approved…)
	sessionMax              int               // longueur au-delà de laquelle le nom de session est coupé

	barre        [3]string // cases pleine, à moitié, éteinte
	sansBarre    bool
	camembert    bool   // ctx.bar=pie : un glyphe au lieu de la barre
	jaugeQuota   string // quota.gauge : none, battery, pie
	jeu          string // jeu d'icônes (icons), pour les jauges en un glyphe
	largeurBarre int    // nombre de cases
	ctxAlerte    int    // seuil d'alerte du contexte, en %
	ctxCompact   bool   // en alerte, rappeler /compact (Claude Code)
	ctxLibelle   string

	quotaAlerte               int // seuil d'alerte des quotas, en %
	projection, plafond       bool
	reset                     string // alert, always, never
	lib5h, lib7j              string
	iconesLib                 map[string]bool // libellés remplacés par une icône (labels=icons) : ctx, 5h, 7d
	icCout                    string          // icône devant le coût (labels=icons)
	coutToujours, coutAnglais bool

	signal string // devant un segment en alerte
}

func optionsDefaut() options {
	return options{
		disp:         dispositionDefaut,
		effort:       "icon+text",
		iconeModele:  "none",
		gitAmont:     true,
		protegees:    branchesDe("main master"),
		prEtat:       true,
		prLien:       true,
		sessionMax:   40,
		barre:        [3]string{"━", "╸", "─"},
		largeurBarre: largeurBarre,
		ctxAlerte:    85,
		ctxCompact:   true,
		ctxLibelle:   "ctx",
		quotaAlerte:  90,
		projection:   true,
		plafond:      true,
		reset:        "alert",
		lib5h:        "5h",
		lib7j:        style.Libelle7j(),
		coutAnglais:  !i18n.Francais(),
		signal:       "●",
		jaugeQuota:   "none",
		jeu:          "nerd",
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
	o.iconeModele = v("model.icon")
	o.effort = v("effort.display")
	o.effort5 = !ssh && v("effort.levels") == "5"
	o.gitResume, o.gitAmont = v("git.status") == "summary", v("git.upstream") == style.Oui
	o.protegees = branchesDe(v("git.protected"))
	o.prEtat, o.prLien = v("pr.status") == style.Oui, v("pr.link") == style.Oui
	if !ssh && v("pr.display") == "icon" {
		o.etatPR = map[string]string{}
		for _, n := range []string{"pr.approved", "pr.changes", "pr.review", "pr.draft"} {
			o.etatPR[n] = st.Glyphe(n)
		}
	}
	o.sessionMax = entierDe(v("session.max"))
	if ssh {
		o.barre, o.sansBarre = [3]string{"━", "╸", "─"}, false
	} else {
		c, ok := st.CasesBarre()
		o.barre, o.sansBarre = c, !ok
		o.camembert = v("ctx.bar") == "pie"
		o.jaugeQuota, o.jeu = v("quota.gauge"), v("icons")
	}
	o.largeurBarre = entierDe(v("ctx.width"))
	o.ctxAlerte, o.ctxLibelle = entierDe(v("ctx.alert")), v("ctx.label")
	o.ctxCompact = v("ctx.compact") == style.Oui
	o.quotaAlerte = entierDe(v("quota.alert"))
	o.projection, o.plafond, o.reset = v("quota.projection") == style.Oui, v("quota.limit") == style.Oui, v("quota.reset")
	o.lib5h, o.lib7j = v("5h.label"), style.LibelleEnVigueur("7d.label", v("7d.label"))
	if !ssh && v("labels") == "icons" { // une icône vide (icons=text ou none) laisse le libellé
		o.iconesLib = map[string]bool{}
		for nom, lib := range map[string]*string{"ctx": &o.ctxLibelle, "5h": &o.lib5h, "7d": &o.lib7j} {
			if g := st.Glyphe(nom); g != "" {
				*lib, o.iconesLib[nom] = g, true
			}
		}
		o.icCout = st.Glyphe("cost")
	}
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
