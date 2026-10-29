package render

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/style"
)

// Les items de la statusline et leur composition en lignes.
//
// Un item donne son texte (vide : absent) et s'il est en alerte. court : la version courte (SSH).
// Dans un segment, les items sont collés tels quels : un item qui en suit un autre porte lui-même son
// espace de tête (double avant la branche, simple ailleurs, et une espace seule quand l'effort est
// inconnu en version courte), comme dans l'ancienne version bash. Dans une disposition personnelle, le
// composeur retire cette espace en tête de segment, et en ajoute une devant un item qui n'en porte pas.
type item func(r *rendu, s *etat.Session, court bool) (texte string, alerte bool)

// defItem : un item du registre ; ouvert : en version longue, il laisse ses couleurs ouvertes pour
// l'item suivant, et le composeur ferme le segment par z ; colle : il porte son espace de tête.
type defItem struct {
	f      item
	ouvert bool
	colle  bool
}

var items = map[string]defItem{
	"model":    {f: itemModele},
	"effort":   {f: itemEffort, colle: true},
	"fast":     {f: itemRapide, colle: true},
	"folder":   {f: itemDossier, ouvert: true},
	"git":      {f: itemGit, ouvert: true, colle: true},
	"worktree": {f: itemWorktree, ouvert: true, colle: true},
	"pr":       {f: itemPR},
	"agent":    {f: itemAgent},
	"style":    {f: itemStyle},
	"session":  {f: itemSession},
	"ctx":      {f: itemCtx},
	"5h":       {f: itemQuota5h},
	"7d":       {f: itemQuota7j},
	"cost":     {f: itemCout},
}

func itemModele(r *rendu, s *etat.Session, court bool) (string, bool) {
	if court { // premier mot seulement, et pas de « Claude » par défaut
		premier, _, _ := strings.Cut(s.Modele, " ")
		return r.nomModele(premier), false
	}
	m := s.Modele
	if r.o.modeleCourt {
		m, _, _ = strings.Cut(m, " ")
	}
	if m == "" {
		m = "Claude"
	}
	return r.nomModele(m), false
}

func itemEffort(r *rendu, s *etat.Session, court bool) (string, bool) {
	switch {
	case s.Effort == "":
		return "", false
	case court:
		return " " + r.puissance(s.Effort), false
	}
	jauge, nom := " "+r.puissance(s.Effort), " "+r.discret+s.Effort+z
	if r.sansGlypheEffort() { // sans icône, la jauge disparaît avec son espace
		jauge = ""
	}
	switch r.o.effort {
	case "icon":
		if jauge != "" && jauge != " " { // niveau inconnu : pas de jauge, on garde le nom
			return jauge, false
		}
	case "text":
		return nom, false
	}
	return jauge + nom, false
}

func itemRapide(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if !s.Rapide {
		return "", false
	}
	return " " + r.jaune + ico(r.iRapide) + "fast" + z, false
}

// itemDossier : en version longue, les couleurs restent ouvertes pour la branche et le worktree (le
// composeur ferme le segment) ; en version courte, le dossier est toujours là, même vide.
func itemDossier(r *rendu, _ *etat.Session, court bool) (string, bool) {
	if court {
		return r.texte + gras + r.nom + z, false
	}
	if r.nom == "" {
		return "", false
	}
	return r.dossier + ico(r.iDossier) + r.texte + gras + r.nom + maigre, false
}

// itemGit : la branche et son état ; en version longue, seulement avec le dossier, en alerte s'il y a
// des conflits.
func itemGit(r *rendu, _ *etat.Session, court bool) (string, bool) {
	g := r.g
	// des changements non commités sur une branche protégée (main…) : la branche passe en rouge, le segment en alerte
	protegee := r.sale && r.o.protegees[g.branche]
	couleur := r.lieu
	if protegee {
		couleur = r.rouge
	}
	if court {
		s := ""
		if g.branche != "" {
			s += " " + couleur + g.branche + z
			if r.sale {
				s += r.jaune + "*" + z
			}
		}
		if g.conflit != 0 {
			s += r.rouge + gras + "!" + z
		}
		return s, false
	}
	if r.nom == "" {
		return "", false
	}
	s := ""
	if g.branche != "" {
		s += "  " + couleur + ico(r.iBranche) + g.branche + r.texte
		compteurs := ""
		if g.indexe != 0 {
			compteurs += " +" + strconv.Itoa(g.indexe)
		}
		if g.modifie != 0 {
			compteurs += " ●" + strconv.Itoa(g.modifie)
		}
		if g.neuf != 0 {
			compteurs += " ?" + strconv.Itoa(g.neuf)
		}
		if g.conflit != 0 {
			compteurs += " " + r.rouge + gras + "×" + strconv.Itoa(g.conflit) + maigre + r.texte
		}
		switch {
		case !g.fiable:
			s += " " + r.discret + "…" // git n'a pas répondu
		case compteurs == "":
			s += " " + r.vert + "✓"
		case r.o.gitResume: // un seul signe : en conflit, sinon modifié
			if g.conflit != 0 {
				s += " " + r.rouge + gras + "×" + maigre + r.texte
			} else {
				s += " " + r.jaune + "*" + r.texte
			}
		default:
			s += compteurs
		}
		if g.avance != 0 && r.o.gitAmont {
			s += " " + r.texte + "↑" + strconv.Itoa(g.avance)
		}
		if g.retard != 0 && r.o.gitAmont {
			s += " " + r.orange + "↓" + strconv.Itoa(g.retard)
		}
	}
	return s, g.conflit != 0 || protegee
}

func itemWorktree(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if r.nom == "" || s.Worktree == "" {
		return "", false
	}
	return " " + r.discret + ico(r.iWorktree) + s.Worktree, false
}

func itemPR(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if s.PR.Num == "" {
		return "", false
	}
	t := "#" + s.PR.Num
	if s.PR.URL != "" && r.o.prLien {
		t = "\x1b]8;;" + s.PR.URL + "\a" + t + "\x1b]8;;\a" // cliquable (Ctrl+clic)
	}
	t = r.lieu + ico(r.iPR) + t
	revue := s.PR.Revue
	if !r.o.prEtat {
		revue = ""
	}
	// l'état, en mot ou en icône (pr.display=icon ; une icône vide laisse le mot)
	etat := func(couleur, mot, icone string) {
		if r.o.etatPR[icone] != "" {
			mot = r.o.etatPR[icone]
		}
		t += " " + couleur + mot
	}
	switch revue {
	case "approved":
		etat(r.vert, i18n.T("approuvée"), "pr.approved")
	case "changes_requested":
		etat(r.rouge, i18n.T("à corriger"), "pr.changes")
	case "pending":
		etat(r.jaune, i18n.T("en revue"), "pr.review")
	case "draft":
		etat(r.discret, i18n.T("brouillon"), "pr.draft")
	}
	return t + z, false
}

func itemAgent(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if s.Agent == "" {
		return "", false
	}
	return r.discret + "agent" + z + " " + r.texte + s.Agent + z, false
}

func itemStyle(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if s.Style == "" || s.Style == "default" {
		return "", false
	}
	return r.discret + "style" + z + " " + r.texte + s.Style + z, false
}

func itemSession(r *rendu, s *etat.Session, _ bool) (string, bool) {
	n := s.Session
	if n == "" {
		return "", false
	}
	if c := []rune(n); len(c) > r.o.sessionMax {
		n = string(c[:r.o.sessionMax-1]) + "…"
	}
	return r.discret + ital + n + z, false
}

// itemCtx : la part du contexte utilisée, avec sa barre en version longue (suivie d'un second z).
func itemCtx(r *rendu, s *etat.Session, court bool) (string, bool) {
	if s.Contexte == "" {
		return "", false
	}
	p := pct(s.Contexte)
	n, _ := strconv.Atoi(p)
	alerte := n >= r.o.ctxAlerte
	t := r.libelle("ctx", r.o.ctxLibelle, r.couleurValeur(n, alerte))
	if !court && !r.o.sansBarre {
		t += r.barre(n, r.o.largeurBarre) + z + " "
	}
	if !court && r.o.camembert {
		t += r.jauge("pie", n, alerte)
	}
	t += r.valeur(p, alerte)
	// le geste qui libère le contexte, avant que l'auto-compact ne le fasse : une commande de Claude Code
	if alerte && r.o.ctxCompact && !court && !r.sansIndice && s.Harness == "claude" {
		r.indice = true
		t += " " + r.discret + "/compact" + z
	}
	return t, alerte
}

func itemQuota5h(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if s.Quotas.H5.Utilise == "" {
		return "", false
	}
	return r.quota("5h", r.o.lib5h, s.Quotas.H5.Utilise, s.Quotas.H5.Reset, 18000)
}

func itemQuota7j(r *rendu, s *etat.Session, _ bool) (string, bool) {
	if s.Quotas.J7.Utilise == "" {
		return "", false
	}
	return r.quota("7d", r.o.lib7j, s.Quotas.J7.Utilise, s.Quotas.J7.Reset, 604800)
}

// itemCout : le coût, par défaut seulement sans quotas (facturation à l'usage).
func itemCout(r *rendu, s *etat.Session, _ bool) (string, bool) {
	c := cout(s.Cout, r.o.coutAnglais)
	if !r.o.coutToujours && (s.Quotas.H5.Utilise != "" || s.Quotas.J7.Utilise != "") || c == "" {
		return "", false
	}
	return r.texte + ico(r.o.icCout) + c + z, false
}

// ligne : les segments d'une ligne, chacun la liste de ses items, et le caractère qui les sépare.
type ligne struct {
	segments [][]string
	sep      string
}

// disposition : les lignes de la version longue, et la ligne de la version courte (SSH).
type disposition struct {
	lignes []ligne
	court  ligne
}

// nouvelleLigne : « a b | c » -> segments {a b} et {c} (« | » sépare les segments, l'espace colle les
// items). Un item inconnu fait paniquer : la disposition par défaut est écrite ici, en Go.
func nouvelleLigne(texte, sep string) ligne {
	l := ligne{sep: sep}
	for _, seg := range strings.Split(texte, "|") {
		noms := strings.Fields(seg)
		for _, n := range noms {
			if items[n].f == nil {
				panic("item inconnu : " + n)
			}
		}
		if len(noms) > 0 {
			l.segments = append(l.segments, noms)
		}
	}
	return l
}

var dispositionDefaut = disposition{
	lignes: []ligne{
		nouvelleLigne(style.Ligne1Defaut, "│"),
		nouvelleLigne(style.Ligne2Defaut, "·"),
	},
	court: nouvelleLigne(style.CourtDefaut, "·"),
}

// composer : les lignes de la disposition (la version courte en SSH). Un segment sans texte disparaît,
// une ligne sans segment aussi ; un segment qui contient un item ouvert est fermé par z, puis tout
// segment est précédé du signal ● si l'un de ses items est en alerte. Séparateurs : couleur SEP,
// DISCRET en version courte.
func (r *rendu) composer(s *etat.Session, d disposition) string {
	court := r.env.SSH
	lignes, couleur := d.lignes, r.sep
	if court {
		lignes, couleur = []ligne{d.court}, r.discret
	}
	var sortie strings.Builder
	k := 0 // lignes écrites
	for _, l := range lignes {
		r.indice = false
		segments, places := r.segmentsDe(l, s, court)
		if len(segments) > 0 {
			sep := " " + couleur + l.sep + z + " "
			texte, debuts := r.afficher(sep, segments)
			if len(debuts) < len(segments) && r.indice { // trop large : les indices cèdent leur place aux segments
				r.sansIndice = true
				segments, places = r.segmentsDe(l, s, court)
				r.sansIndice = false
				texte, debuts = r.afficher(sep, segments)
			}
			sortie.WriteString(texte)
			if r.places != nil {
				for i, d := range debuts {
					for _, p := range places[i] {
						p.Ligne, p.Debut, p.Fin = k, p.Debut+d, p.Fin+d
						*r.places = append(*r.places, p)
					}
				}
			}
			k++
		}
	}
	return sortie.String()
}

// segmentsDe : les segments d'une ligne, chacun avec les colonnes de ses items (relatives au segment).
func (r *rendu) segmentsDe(l ligne, s *etat.Session, court bool) (segments []string, places [][]Place) {
	for _, seg := range l.segments {
		texte, alerte, ouvert := "", false, false
		var ps []Place
		for _, n := range seg {
			if r.o.masque[n] {
				continue
			}
			d := items[n]
			t, a := d.f(r, s, court)
			switch {
			case texte == "": // en tête de segment, pas d'espace (une disposition personnelle peut l'y mettre)
				t = strings.TrimLeft(t, " ")
			case t != "" && !d.colle:
				t = " " + t
			}
			if v := ansi.Visible(t); strings.TrimSpace(v) != "" {
				debut := visibles(ansi.Visible(texte)) + visibles(v) - visibles(strings.TrimLeft(v, " "))
				ps = append(ps, Place{Item: n, Debut: debut, Fin: visibles(ansi.Visible(texte)) + visibles(strings.TrimRight(v, " "))})
			}
			texte += t
			alerte = alerte || a
			ouvert = ouvert || t != "" && d.ouvert && !court
		}
		if texte == "" {
			continue
		}
		if ouvert {
			texte += z
		}
		if alerte {
			avant := visibles(ansi.Visible(texte))
			texte = r.signal(texte)
			decaler(ps, visibles(ansi.Visible(texte))-avant)
		}
		segments = append(segments, texte)
		places = append(places, ps)
	}
	return segments, places
}

// Place : où un item est dessiné : sa ligne (0 : la première écrite), ses colonnes de début et de fin
// (fin exclue), en caractères visibles comme ansi.Visible.
type Place struct {
	Item              string
	Ligne, Debut, Fin int
}

func visibles(s string) int { return utf8.RuneCountInString(s) }

func decaler(ps []Place, d int) {
	for i := range ps {
		ps[i].Debut += d
		ps[i].Fin += d
	}
}
