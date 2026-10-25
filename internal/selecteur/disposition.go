package selecteur

import (
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Disposition : tout s'adapte à la taille du terminal, en trois agencements :
//
//	large     la liste à gauche, Clawd et la mini-fenêtre à droite
//	empilée   la liste, puis la mini-fenêtre dessous (la liste passe avant : la fenêtre n'a que le reste)
//	compacte  ni logo ni mini-fenêtre : la liste, puis la vraie statusline du thème en aperçu
//
// Le logo n'est dessiné que s'il reste assez de hauteur une fois la liste et la fenêtre servies. La
// disposition est calculée à chaque touche et à chaque redimensionnement (ajuster), jamais par View.

type agencement int

const (
	large agencement = iota
	empilee
	compacte
)

func (a agencement) String() string { return [...]string{"large", "empilée", "compacte"}[a] }

const (
	compacteLargeur  = 60 // en dessous (colonnes) : disposition compacte
	compacteHauteur  = 22 // en dessous (lignes) : disposition compacte
	largeurMin       = 40 // plus petit, l'écran est dessiné à cette taille et déborde
	hauteurMin       = 16
	listeMin         = 5  // thèmes visibles au moins avant de donner de la place à la fenêtre
	listeMaxEmpilee  = 10 // empilée : la liste n'en montre pas plus, le reste va à la fenêtre
	fenetreMin       = 9  // la plus petite mini-fenêtre lisible (deux lignes de conversation)
	fenetreCote      = 46 // largeur minimale de la fenêtre pour qu'elle tienne à côté de la liste
	lignesLogo       = 4  // len(logo)
	lignesInfos      = 2  // la ligne du dépôt (en vigueur, survolé…), puis une ligne vide
	lignesApercu     = 3  // sans fenêtre : une ligne vide, puis les deux lignes de la statusline
	cadreListe       = 4  // autour des thèmes : le filtre, « ↑ de plus », « ↓ de plus », les pastilles
	aideDeuxColonnes = 76 // à partir de cette largeur, l'aide complète passe sur deux colonnes si besoin
)

type disposition struct {
	agencement
	logo     bool     // le titre en grandes lettres
	fenetre  bool     // la mini-fenêtre (sinon l'aperçu de la statusline seule)
	mascotte bool     // la place de Clawd au-dessus de la fenêtre
	lw, ww   int      // largeurs de la colonne de la liste et de la fenêtre
	liste    int      // thèmes visibles
	wh       int      // hauteur de la fenêtre
	corps    int      // lignes entre l'en-tête et le pied
	pied     []string // le pied de page (aide courte, recherche, ou panneau de validation)
}

func (m *Modele) disposer() disposition {
	w, h := max(m.largeur, largeurMin), max(m.hauteur, hauteurMin)
	d := disposition{pied: m.lignesPied(w - 1), mascotte: m.mascotte != 2}
	reste := h - len(d.pied) - 1 // une ligne vide au-dessus du pied
	if w < compacteLargeur || h < compacteHauteur {
		d.agencement, d.lw, d.mascotte = compacte, w-1, false
		d.corps = reste - lignesInfos
		d.liste = max(d.corps-(cadreListe-1)-lignesApercu, 1) // pas de pastilles
		return d
	}
	d.lw = max(min(w/4, 36), 24)
	if w >= d.lw+2+fenetreCote {
		d.agencement, d.fenetre = large, true
		d.ww = w - d.lw - 3                                                                      // une colonne de marge à droite
		d.logo = !m.aide && reste-lignesInfos-lignesLogo >= max(listeMin+cadreListe, fenetreMin) // l'aide prend sa place
		d.corps = reste - lignesInfos - lignesLogo*b2i(d.logo)
		d.liste = max(d.corps-cadreListe, 1)
		d.poserFenetre(d.corps)
		return d
	}
	d.agencement, d.lw, d.ww = empilee, w-1, w-1
	besoin := listeMin + cadreListe + fenetreMin
	d.logo = !m.aide && reste-lignesInfos-lignesLogo >= besoin
	d.corps = reste - lignesInfos - lignesLogo*b2i(d.logo)
	if d.fenetre = d.corps >= besoin; !d.fenetre {
		d.mascotte = false
		d.liste = max(d.corps-cadreListe-lignesApercu, 1)
		return d
	}
	d.liste = min(max((d.corps-cadreListe)/3, listeMin), listeMaxEmpilee, d.corps-cadreListe-fenetreMin)
	d.poserFenetre(d.corps - d.liste - cadreListe)
	return d
}

// poserFenetre : la fenêtre dans la hauteur donnée, avec Clawd au-dessus s'il reste sa place.
func (d *disposition) poserFenetre(place int) {
	if d.mascotte && place-hauteurMascotte >= fenetreMin {
		d.wh = place - hauteurMascotte
	} else {
		d.mascotte, d.wh = false, max(place, fenetreMin)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// lignesPied : le pied de page, en lignes de largeur w au plus : le panneau de validation (replié, jamais
// tronqué), la recherche en cours, ou l'aide courte.
func (m *Modele) lignesPied(w int) []string {
	var avant []string // un avertissement, un message de l'onglet Statusline, au-dessus du pied
	if m.avertiQuitter {
		for _, x := range replier(strings.Fields(i18n.T("⚠ Modifications du style non gardées : encore une fois pour quitter sans les garder, s (onglet Statusline) pour les garder.")), " ", w) {
			avant = append(avant, gras+x+z)
		}
	}
	if m.onglet == 1 && m.os.message != "" {
		for _, x := range replier(strings.Fields(m.os.message), " ", w) {
			avant = append(avant, gras+x+z)
		}
	}
	return append(avant, m.piedDuMode(w)...)
}

func (m *Modele) piedDuMode(w int) []string {
	switch md := m.mode(); md {
	case enStyleQuestion:
		morceaux := []string{gras + i18n.T("Garder ce style :") + z}
		for _, r := range m.raccourcisDuPied(md) {
			morceaux = append(morceaux, "["+r.touche()+"] "+r.maintenant(m))
		}
		l := replier(morceaux, "  ", w)
		if m.os.ignore {
			l = append(l, gras+tronquer(i18n.T("⚠ AGENTLINE_STYLE=default : il ne se verra pas tant qu'elle est définie"), w)+z)
		}
		return l
	case enStyleSaisie:
		debut := gras + i18n.T("%s : %s", m.os.cle().Nom, m.os.texte) + z + "▏   "
		return []string{tronquer(debut+estompe+m.aideCourte(md, w-largeur(debut))+z, w)}
	case enQuestion:
		morceaux := []string{gras + i18n.T("Garder « %s » :", m.titres[m.i]) + z}
		for _, r := range m.raccourcisDuPied(md) {
			morceaux = append(morceaux, "["+r.touche()+"] "+r.maintenant(m))
		}
		l := replier(morceaux, "  ", w)
		if a := m.avertissement(); a != "" {
			for _, x := range replier(strings.Fields(a), " ", w) {
				l = append(l, gras+x+z)
			}
		}
		return l
	case enRecherche:
		debut := gras + i18n.T("Chercher : %s", m.recherche) + z + "▏   "
		return []string{tronquer(debut+estompe+m.aideCourte(md, w-largeur(debut))+z, w)}
	default:
		return []string{tronquer(estompe+m.aideCourte(md, w)+z, w)}
	}
}

// ajuster : la disposition pour la taille et l'état actuels, et le début de la liste pour que le thème
// choisi reste visible.
func (m *Modele) ajuster() {
	m.d = m.disposer()
	m.ajusterStyle()
	m.lignesListe = m.d.liste
	nv, vis := len(m.vue), m.d.liste
	if m.p < m.haut {
		m.haut = m.p
	}
	if m.p >= m.haut+vis {
		m.haut = m.p - vis + 1
	}
	if m.haut > 0 && m.haut+vis > nv {
		m.haut = max(nv-vis, 0)
	}
}
