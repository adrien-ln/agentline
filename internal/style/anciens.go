package style

import "strings"

// Les anciens noms français des options, des items et des valeurs (avant le passage à l'anglais) : un
// fichier de style écrit avec eux se lit comme avec les nouveaux, et la CLI les accepte ; l'écriture
// produit toujours les nouveaux.

// anciensItems : ancien nom d'item -> nom actuel.
var anciensItems = map[string]string{"modele": "model", "rapide": "fast", "dossier": "folder", "7j": "7d", "cout": "cost"}

// anciennesCles : ancien nom d'option -> nom actuel (les icônes, icone.<nom>, ont leur table).
var anciennesCles = map[string]string{
	"ligne1": "line1", "ligne2": "line2", "court": "short", "masquer": "hide", "icones": "icons",
	"sep.ligne1": "sep.line1", "sep.ligne2": "sep.line2", "sep.court": "sep.short",
	"modele.nom": "model.name", "modele.couleur": "model.color", "effort.affichage": "effort.display",
	"git.etat": "git.status", "git.amont": "git.upstream", "pr.etat": "pr.status", "pr.lien": "pr.link",
	"ctx.barre": "ctx.bar", "ctx.largeur": "ctx.width", "ctx.alerte": "ctx.alert", "ctx.libelle": "ctx.label",
	"quota.alerte": "quota.alert", "quota.plafond": "quota.limit", "5h.libelle": "5h.label", "7j.libelle": "7d.label",
	"cout.quand": "cost.when", "cout.format": "cost.format",
}

// anciensGlyphes : ancien nom d'icône (icone.<nom>) -> nom actuel.
var anciensGlyphes = map[string]string{"dossier": "folder", "branche": "branch", "alerte": "alert", "rapide": "fast",
	"effort.bas": "effort.low", "effort.haut": "effort.high"}

// anciennesValeurs : ancienne valeur nommée -> valeur actuelle ; « non » dépend de l'option (anciennesNon).
var anciennesValeurs = map[string]string{
	"oui": Oui, "texte": "text", "aucune": "none", "complet": "full", "court": "short", "degrade": "gradient", "unie": "solid",
	"icone+texte": "icon+text", "icone": "icon", "resume": "summary", "traits": "lines", "blocs": "blocks", "points": "dots",
	"alerte": "alert", "toujours": "always",
}

// anciennesNon : ce que voulait dire « non », selon les valeurs de l'option.
var anciennesNon = []string{Non, "none", "never"}

// NomCle : le nom actuel de l'option (son ancien nom français est accepté) ; inchangé s'il est inconnu.
func NomCle(nom string) string {
	if n, ok := anciennesCles[nom]; ok {
		return n
	}
	if g, ok := strings.CutPrefix(nom, "icone."); ok {
		if n, ok := anciensGlyphes[g]; ok {
			g = n
		}
		return "icon." + g
	}
	return nom
}

// NomItem : le nom actuel de l'item (son ancien nom est accepté) ; faux s'il est inconnu.
func NomItem(n string) (string, bool) {
	if a, ok := anciensItems[n]; ok {
		n = a
	}
	return n, EstItem(n)
}

// valeurActuelle : une ancienne valeur nommée de l'option remplacée par la nouvelle ; les autres valeurs
// (et les textes libres) sont gardées telles quelles.
func valeurActuelle(c Cle, v string) string {
	if c.Genre != Choix && c.Genre != Barre || contientValeur(c.Valeurs, v) {
		return v
	}
	if v == "non" {
		for _, n := range anciennesNon {
			if contientValeur(c.Valeurs, n) {
				return n
			}
		}
	}
	if n, ok := anciennesValeurs[v]; ok && contientValeur(c.Valeurs, n) {
		return n
	}
	return v
}

func contientValeur(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}
