package render

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/ansi"
	"github.com/adrien-ln/agentline/internal/etat"
	"github.com/adrien-ln/agentline/internal/style"
)

// TestIndiceCompact : /compact suit l'alerte du contexte, pour Claude Code seulement, et cède sa place quand
// la ligne serait trop large (le quota en alerte reste).
func TestIndiceCompact(t *testing.T) {
	rendre := func(s etat.Session, colonnes int, regles string) string {
		e := env(t, mode{colonnes: colonnes}, t.TempDir())
		e.Style = style.Lire([]byte(regles))
		e.GitFictif = &GitFictif{Branche: "dev"}
		return ansi.Visible(RenderSession(s, e))
	}
	s := etat.Session{Harness: "claude", Modele: "Opus", Contexte: "91", Quotas: etat.Quotas{H5: etat.Quota{Utilise: "95"}}}
	if v := rendre(s, 200, ""); !strings.Contains(v, "91% /compact") {
		t.Errorf("contexte en alerte : /compact attendu :\n%s", v)
	}
	for _, regles := range []string{"ctx.compact=no", "ctx.alert=95"} {
		if v := rendre(s, 200, regles); strings.Contains(v, "/compact") {
			t.Errorf("%s : pas de /compact :\n%s", regles, v)
		}
	}
	c := s
	c.Harness = "copilot"
	if v := rendre(c, 200, ""); strings.Contains(v, "/compact") {
		t.Errorf("Copilot CLI : pas de /compact (commande de Claude Code) :\n%s", v)
	}
	// juste la place de la ligne sans indice : l'indice part, le quota reste
	sans := strings.Split(rendre(s, 200, "ctx.compact=no"), "\n")[1]
	if v := rendre(s, len([]rune(sans))+4, ""); strings.Contains(v, "/compact") || !strings.Contains(v, "5h 95%") {
		t.Errorf("ligne trop étroite : l'indice doit céder sa place au quota :\n%s", v)
	}
}

// TestItemsDuStyle : le registre du rendu et la liste des items du style ont les mêmes noms.
func TestItemsDuStyle(t *testing.T) {
	var reg []string
	for n := range items {
		reg = append(reg, n)
	}
	l := append([]string(nil), style.Items...)
	sort.Strings(reg)
	sort.Strings(l)
	if !reflect.DeepEqual(reg, l) {
		t.Errorf("registre %v, style %v", reg, l)
	}
	var d []string
	for _, x := range dispositionDefaut.lignes {
		d = append(d, x.sep)
	}
	if d[0] != style.Defaut().Valeur("sep.line1") || d[1] != style.Defaut().Valeur("sep.line2") {
		t.Errorf("séparateurs par défaut : %v", d)
	}
}

// TestOptionsDuStyleDefaut : le style par défaut lu option par option (sans le raccourci du rendu)
// donne exactement les options par défaut, en version longue comme en SSH.
func TestOptionsDuStyleDefaut(t *testing.T) {
	for _, ssh := range []bool{false, true} {
		if o := lireOptions(style.Defaut(), ssh); !reflect.DeepEqual(o, optionsDefaut()) {
			t.Errorf("ssh %v :\n%+v\n%+v", ssh, o, optionsDefaut())
		}
	}
}

// TestReferencesStyleToutDefaut : un fichier de style qui écrit toutes les valeurs par défaut en toutes
// lettres redonne les 168 références, octet pour octet.
func TestReferencesStyleToutDefaut(t *testing.T) {
	st := style.Lire([]byte(style.ToutDefaut()))
	dossier := fixtures(t)
	config := t.TempDir()
	cas, _ := filepath.Glob(filepath.Join(racineDepot, "tests", "cas", "*.json"))
	n := 0
	for _, c := range cas {
		nom := strings.TrimSuffix(filepath.Base(c), ".json")
		j := casJSON(t, nom, dossier)
		for nm, m := range modes {
			e := env(t, m, config)
			e.Style = st
			comparerSans(t, filepath.Join(racineDepot, "tests", "golden", nom+"."+nm+".ans"), Render(j, e))
			n++
		}
	}
	themes, _ := filepath.Glob(filepath.Join(racineDepot, "themes", "*.theme"))
	for _, f := range themes {
		nom := strings.TrimSuffix(filepath.Base(f), ".theme")
		for _, c := range []string{"complet", "tendu", "api"} {
			e := env(t, mode{colonnes: 200, vars: map[string]string{"AGENTLINE_THEME": nom}}, config)
			e.Style = st
			comparerSans(t, filepath.Join(racineDepot, "tests", "golden", "themes", nom+"."+c+".ans"), Render(casJSON(t, c, dossier), e))
			n++
		}
	}
	if n < 150 {
		t.Errorf("%d références seulement", n)
	}
}

// comparerSans : comme comparer, sans jamais réécrire la référence (même avec -maj).
func comparerSans(t *testing.T, fichier, obtenu string) {
	t.Helper()
	if *maj {
		return
	}
	comparer(t, fichier, obtenu)
}

// rendre : le cas rendu avec le style décrit (lignes « clé=valeur »), état git imposé, texte visible.
func rendre(t *testing.T, cas, regles string, ssh bool, g *GitFictif) string {
	t.Helper()
	st := style.Lire([]byte(regles))
	for _, l := range strings.Split(strings.TrimSpace(regles), "\n") {
		if cle, v, _ := strings.Cut(l, "="); l != "" && !strings.HasPrefix(regles, "#!") && st.Valeur(cle) != v && !(cle == "line1" || cle == "line2" || cle == "short") {
			t.Fatalf("%s refusé par le style", l)
		}
	}
	m := modes["riche"]
	if ssh {
		m = modes["ssh"]
	}
	e := env(t, m, t.TempDir())
	e.Style = st
	if g == nil {
		g = &GitFictif{Branche: "main", Indexe: 1, Modifie: 2, Neuf: 3, Avance: 2, Retard: 1}
	}
	e.GitFictif = g
	return ansi.Visible(Render(casJSON(t, cas, "/x"), e))
}

func TestOptionsDuStyle(t *testing.T) {
	type essai struct {
		cas, regles string
		ssh         bool
		voulu, pas  []string
	}
	for _, c := range []essai{
		{cas: "complet", regles: "line1=folder git | model\nline2=ctx", voulu: []string{"projet-propre", "main", "│ Opus 5.5\nctx"}, pas: []string{"5h", "#42", "refonte"}},
		{cas: "complet", regles: "line1=git effort | model", voulu: []string{"\n", "main +1 ●2 ?3 ↑2 ↓1 󱐋 high │ Opus"}},
		{cas: "complet", regles: "line2=ctx 5h 7d", voulu: []string{"42% 5h 61% ↗76% 7j 23%"}},
		{cas: "complet", regles: "hide=effort pr session", voulu: []string{"Opus 5.5 │"}, pas: []string{"high", "#42", "refonte"}},
		{cas: "complet", regles: "hide=model", voulu: []string{"󱐋 high │"}, pas: []string{" 󱐋 high"}},
		{cas: "complet", regles: "icons=none", voulu: []string{"Opus 5.5 high │ ● projet-propre  main +1", "│ #42 en revue"}},
		{cas: "complet", regles: "icons=text", voulu: []string{"▸ projet-propre  ⎇ main", "PR #42", "▴ high"}},
		{cas: "complet", regles: "icon.folder=D\nicon.branch=", voulu: []string{"D projet-propre  main"}},
		{cas: "complet", regles: "sep.line1=/\nsep.line2=|", voulu: []string{"high / ", "42% | 5h"}},
		{cas: "complet", regles: "model.name=short\neffort.display=text", voulu: []string{"Opus high │"}, pas: []string{"Opus 5.5", "󱐋"}},
		{cas: "complet", regles: "effort.display=icon", voulu: []string{"Opus 5.5 󱐋 │"}},
		{cas: "complet", regles: "git.status=summary\ngit.upstream=no", voulu: []string{"main * │"}, pas: []string{"↑2", "+1"}},
		{cas: "api", regles: "icons=text", voulu: []string{"low │ ● ▸ projet-sale  ⎇ main +1"}},
		{cas: "api", regles: "icons=text\ngit.protected=develop", voulu: []string{"low │ ▸ projet-sale  ⎇ main +1"}},
		{cas: "api", regles: "icons=text\ngit.protected=", voulu: []string{"low │ ▸ projet-sale  ⎇ main +1"}},
		{cas: "complet", regles: "pr.status=no", voulu: []string{"#42 │"}, pas: []string{"en revue"}},
		{cas: "complet", regles: "session.max=8", voulu: []string{"refonte…"}, pas: []string{"refonte de"}},
		{cas: "complet", regles: "ctx.bar=none\nctx.label=contexte", voulu: []string{"contexte 42%"}, pas: []string{"━"}},
		{cas: "complet", regles: "ctx.bar=#=-\nctx.width=4", voulu: []string{"ctx #=-- 42%"}},
		{cas: "complet", regles: "ctx.bar=blocks", voulu: []string{"ctx ███▌░░░░ 42%"}},
		{cas: "complet", regles: "ctx.alert=40\nquota.alert=60", voulu: []string{"● ctx", "● 5h 61% ↗76%", "09:00 · 7j"}},
		{cas: "complet", regles: "quota.reset=always\nquota.projection=no\n5h.label=jour\n7d.label=", voulu: []string{"· jour 61%", "09:00 · 23%", "lun 19:20"}, pas: []string{"↗"}},
		{cas: "tendu", regles: "quota.limit=no\nquota.reset=never", voulu: []string{"● 5h 95% ↗105% · 7j 80% ↗93%"}, pas: []string{"plafond", "08:30"}},
		{cas: "tendu", regles: "icon.signal=!", voulu: []string{"! ctx", "! 5h"}},
		{cas: "tendu", regles: "icon.signal=", voulu: []string{"\nctx ━"}, pas: []string{"● "}},
		{cas: "complet", regles: "cost.when=always", pas: []string{"$"}}, // pas de coût dans la session
		{cas: "api", regles: "cost.format=en", voulu: []string{"· $4.98"}},
		{cas: "tendu", regles: "cost.when=always\ncost.format=en", pas: []string{"$"}},
		{cas: "complet", regles: "model.color=solid", voulu: []string{"Opus 5.5"}},
		// labels=icons : une icône à la place de chaque libellé ; sans Nerd Font (icons=text) ou vide, le libellé reste
		{cas: "complet", regles: "labels=icons", voulu: []string{"\U000f035b ━", "· \U000f051b 61%", "· \U000f0a33 23%"}, pas: []string{"ctx", "5h", "7j"}},
		{cas: "complet", regles: "labels=icons\nicon.ctx=C\nicon.7d=", voulu: []string{"C ━", "\U000f051b 61%", "· 7j 23%"}},
		{cas: "complet", regles: "labels=icons\nicons=text", voulu: []string{"ctx ━", "· 5h 61%", "· 7j 23%"}},
		{cas: "api", regles: "labels=icons\ncost.format=en", voulu: []string{"· \U000f0114 $4.98"}},
		// jauges en un glyphe, dans le jeu d'icônes : camembert du contexte, batterie (ce qui reste) ou camembert des quotas
		{cas: "complet", regles: "ctx.bar=pie", voulu: []string{"ctx \U000f0aa0 42%"}, pas: []string{"━"}},
		{cas: "complet", regles: "ctx.bar=pie\nicons=text", voulu: []string{"ctx ◑ 42%"}},
		{cas: "complet", regles: "ctx.bar=pie\nicons=none", voulu: []string{"\nctx 42%"}},
		{cas: "complet", regles: "quota.gauge=battery", voulu: []string{"5h \U000f007d 61%", "7j \U000f0081 23%"}},
		{cas: "complet", regles: "quota.gauge=battery\nicons=text", voulu: []string{"5h 61%", "7j 23%"}},
		{cas: "complet", regles: "quota.gauge=pie\nicons=text", voulu: []string{"5h ◑ 61%", "7j ◔ 23%"}},
		{cas: "tendu", ssh: true, regles: "quota.gauge=battery\nctx.bar=pie", pas: []string{"\U000f007a", "\U000f007b", "\U000f0aa4"}},
		{cas: "tendu", ssh: true, regles: "labels=icons", voulu: []string{"5h"}, pas: []string{"\U000f051b"}},
		// SSH : disposition court, seuils, libellés, séparateur ; le reste garde la variante compacte
		{cas: "tendu", ssh: true, regles: "short=5h | model\nsep.short=/\n5h.label=jour\nquota.alert=99\nicons=none\nmodel.name=full\nquota.limit=no",
			voulu: []string{"● jour 95% ↻ 08:30 plafond ~08:14 / Opus"}},
		{cas: "tendu", ssh: true, regles: "hide=ctx cost\nctx.alert=95", voulu: []string{"Opus max · projet-sale main* · ● 5h"}},
	} {
		obtenu := rendre(t, c.cas, c.regles, c.ssh, nil)
		for _, v := range c.voulu {
			if !strings.Contains(obtenu, v) {
				t.Errorf("%s %q : %q absent de\n%s", c.cas, c.regles, v, obtenu)
			}
		}
		for _, p := range c.pas {
			if strings.Contains(obtenu, p) {
				t.Errorf("%s %q : %q présent dans\n%s", c.cas, c.regles, p, obtenu)
			}
		}
	}
}

// TestModeleUni : une seule couleur pour tout le nom.
func TestModeleUni(t *testing.T) {
	e := env(t, modes["riche"], t.TempDir())
	e.GitFictif = &GitFictif{Branche: "main"}
	e.Style = style.Lire([]byte("model.color=solid"))
	l := Render(casJSON(t, "complet", "/x"), e)
	debut := l[:strings.Index(l, "Opus 5.5")]
	if strings.Count(l[:strings.Index(l, "5.5")+3], "38;2;") != 1 || !strings.HasSuffix(debut, "m") {
		t.Errorf("%q", l[:60])
	}
}

// TestIconesLibellesCouleur : avec labels=icons, chaque icône a la couleur de sa valeur (le libellé en
// texte reste discret).
func TestIconesLibellesCouleur(t *testing.T) {
	e := env(t, modes["riche"], t.TempDir())
	e.GitFictif = &GitFictif{Branche: "main"}
	e.Style = style.Lire([]byte("labels=icons"))
	l := Render(casJSON(t, "complet", "/x"), e)
	for _, c := range [][2]string{{"\U000f035b", "42%"}, {"\U000f051b", "61%"}, {"\U000f0a33", "23%"}} {
		i := strings.Index(l, gras+c[1])
		if i < 0 {
			t.Fatalf("%s absent : %q", c[1], l)
		}
		couleur := l[strings.LastIndex(l[:i], "\x1b["):i] // la couleur juste avant la valeur
		if !strings.Contains(l, couleur+c[0]+z) {
			t.Errorf("icône %q : couleur %q de %s attendue\n%q", c[0], couleur, c[1], l)
		}
	}
}

// TestStyleInvalideIgnore : un fichier de style truffé d'erreurs ne casse pas le rendu.
func TestStyleInvalideIgnore(t *testing.T) {
	if a, b := rendre(t, "complet", "", false, nil), rendre(t, "complet", "#!\nzut\nctx.width=99\nline1=model | ???\nicons=emoji\n=\n", false, nil); a != b {
		t.Errorf("\n%s\n%s", a, b)
	}
}

// TestAncienStyleMemeRendu : un fichier de style à l'ancienne (noms et valeurs en français) donne le
// même rendu que le même fichier écrit avec les nouveaux noms, sur chaque cas, en long et en SSH.
func TestAncienStyleMemeRendu(t *testing.T) {
	ancien := "ligne1=modele effort rapide | dossier git | pr\nligne2=ctx | 7j | 5h | cout\ncourt=modele | dossier git | cout\n" +
		"masquer=session\nicones=texte\nsep.ligne1=/\nsep.court=|\nicone.dossier=D\nicone.effort.haut=^\nmodele.nom=court\n" +
		"modele.couleur=unie\neffort.affichage=icone+texte\ngit.etat=resume\ngit.amont=non\npr.etat=non\npr.lien=non\n" +
		"ctx.barre=blocs\nctx.largeur=10\nctx.alerte=70\nctx.libelle=contexte\nquota.alerte=60\nquota.projection=non\n" +
		"quota.plafond=non\nquota.reset=toujours\n5h.libelle=5 h\n7j.libelle=semaine\ncout.quand=toujours\ncout.format=en\n"
	nouveau := "line1=model effort fast | folder git | pr\nline2=ctx | 7d | 5h | cost\nshort=model | folder git | cost\n" +
		"hide=session\nicons=text\nsep.line1=/\nsep.short=|\nicon.folder=D\nicon.effort.high=^\nmodel.name=short\n" +
		"model.color=solid\neffort.display=icon+text\ngit.status=summary\ngit.upstream=no\npr.status=no\npr.link=no\n" +
		"ctx.bar=blocks\nctx.width=10\nctx.alert=70\nctx.label=contexte\nquota.alert=60\nquota.projection=no\n" +
		"quota.limit=no\nquota.reset=always\n5h.label=5 h\n7d.label=semaine\ncost.when=always\ncost.format=en\n"
	for _, c := range []string{"complet", "tendu", "api", "plafond", "minimal"} {
		for _, ssh := range []bool{false, true} {
			a, n := rendre(t, c, "#!\n"+ancien, ssh, nil), rendre(t, c, nouveau, ssh, nil)
			if a != n {
				t.Errorf("%s (ssh %v) :\nancien  : %q\nnouveau : %q", c, ssh, a, n)
			}
			if d := rendre(t, c, "", ssh, nil); d == n && !ssh {
				t.Errorf("%s : le style n'a rien changé", c)
			}
		}
	}
}

// TestLigneVide : une disposition vide ne donne aucune ligne, sans erreur.
func TestLigneVide(t *testing.T) {
	if o := rendre(t, "complet", "line1=\nline2=ctx", false, nil); !strings.HasPrefix(o, "ctx") || strings.Count(o, "\n") != 1 {
		t.Errorf("%q", o)
	}
}

// BenchmarkStyle : la lecture d'un fichier de style complet (à comparer au rendu : < 0,1 ms attendu).
func BenchmarkStyle(b *testing.B) {
	t := []byte(style.ToutDefaut() + "ctx.width=12\nhide=pr\nline1=model | folder git\n")
	for b.Loop() {
		lireOptions(style.Lire(t), false)
	}
}
