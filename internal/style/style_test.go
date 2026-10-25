package style

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCatalogue : chaque défaut est valide, chaque nom unique, chaque valeur nommée acceptée.
func TestCatalogue(t *testing.T) {
	vus := map[string]bool{}
	for _, c := range Cles() {
		if vus[c.Nom] {
			t.Errorf("%s : deux fois", c.Nom)
		}
		vus[c.Nom] = true
		if err := c.Valider(c.Defaut); err != nil {
			t.Errorf("%s : défaut %q refusé : %v", c.Nom, c.Defaut, err)
		}
		for _, v := range c.Valeurs {
			if err := c.Valider(v); err != nil {
				t.Errorf("%s : valeur %q refusée : %v", c.Nom, v, err)
			}
		}
		if c.Aide == "" {
			t.Errorf("%s : sans aide", c.Nom)
		}
		for _, i := range c.Items {
			if !EstItem(i) {
				t.Errorf("%s : item inconnu %s", c.Nom, i)
			}
		}
	}
}

// TestToutDefaut : un fichier qui écrit toutes les valeurs par défaut donne le style par défaut.
func TestToutDefaut(t *testing.T) {
	s := Lire([]byte(ToutDefaut()))
	if !s.EstDefaut() {
		t.Errorf("écarts : %q", s.Texte())
	}
}

// TestLireTolerant : lignes invalides ignorées ; le fichier est écrit avec les anciens noms français, lus
// sous les nouveaux (TestAnciensNoms compare les deux écritures).
func TestLireTolerant(t *testing.T) {
	s := Lire([]byte("# commentaire\r\n  ctx.largeur = 12 \r\nctx.largeur=99\ninconnue=3\nmodele.nom=moyen\nsans egal\n=x\nsep.ligne1=▶\nicone.pr=\nctx.barre=😀😀😀\nmasquer=pr agent\nligne2=ctx 5h | nimporte\n"))
	for cle, attendu := range map[string]string{
		"ctx.width": "12", "model.name": "full", "sep.line1": "▶", "icon.pr": "", "ctx.bar": "lines",
		"hide": "pr agent", "line2": Ligne2Defaut,
	} {
		if v := s.Valeur(cle); v != attendu {
			t.Errorf("%s : %q, attendu %q", cle, v, attendu)
		}
	}
	if !s.Regle("icon.pr") || s.Regle("model.name") {
		t.Error("icon.pr= (vide) est un réglage, une valeur invalide n'en est pas un")
	}
	if g := s.Glyphe("pr"); g != "" {
		t.Errorf("icône pr vide : %q", g)
	}
}

// TestAnciensNoms : un fichier de style à l'ancienne (noms et valeurs en français) donne le même style
// qu'à la nouvelle, et se réécrit avec les nouveaux noms.
func TestAnciensNoms(t *testing.T) {
	ancien := Lire([]byte("ligne1=modele effort | dossier git\nligne2=ctx | 7j | cout\ncourt=modele | cout\nmasquer=rapide 7j\n" +
		"icones=texte\nsep.ligne1=/\nsep.court=|\nicone.dossier=D\nicone.effort.haut=^\nmodele.nom=court\nmodele.couleur=unie\n" +
		"effort.affichage=icone\ngit.etat=resume\ngit.amont=non\npr.etat=non\npr.lien=non\nctx.barre=non\nctx.largeur=10\n" +
		"ctx.alerte=70\nctx.libelle=contexte\nquota.alerte=80\nquota.projection=non\nquota.plafond=non\nquota.reset=toujours\n" +
		"5h.libelle=5 h\n7j.libelle=semaine\ncout.quand=toujours\ncout.format=en\n"))
	nouveau := Lire([]byte("line1=model effort | folder git\nline2=ctx | 7d | cost\nshort=model | cost\nhide=fast 7d\n" +
		"icons=text\nsep.line1=/\nsep.short=|\nicon.folder=D\nicon.effort.high=^\nmodel.name=short\nmodel.color=solid\n" +
		"effort.display=icon\ngit.status=summary\ngit.upstream=no\npr.status=no\npr.link=no\nctx.bar=none\nctx.width=10\n" +
		"ctx.alert=70\nctx.label=contexte\nquota.alert=80\nquota.projection=no\nquota.limit=no\nquota.reset=always\n" +
		"5h.label=5 h\n7d.label=semaine\ncost.when=always\ncost.format=en\n"))
	if !ancien.Egal(nouveau) || ancien.Texte() != nouveau.Texte() {
		t.Errorf("ancien et nouveau diffèrent :\n%s\n%s", ancien.Texte(), nouveau.Texte())
	}
	if n := strings.Count(nouveau.Texte(), "\n") - 1; n != 28 {
		t.Errorf("%d réglages lus, attendu 28 :\n%s", n, nouveau.Texte())
	}
	for _, x := range []string{"ligne", "modele", "dossier", "cout", "7j", "icone", "masquer", "=oui", "=non\n", "toujours"} {
		if strings.Contains(nouveau.Texte(), x) {
			t.Errorf("%q réécrit :\n%s", x, nouveau.Texte())
		}
	}
	if s := Lire([]byte("quota.reset=non\n")); s.Valeur("quota.reset") != "never" {
		t.Errorf("quota.reset=non : %q", s.Valeur("quota.reset"))
	}
	if c, ok := Trouver("ctx.libelle"); !ok || c.Nom != "ctx.label" {
		t.Errorf("Trouver(ctx.libelle) : %v %s", ok, c.Nom)
	}
	if n, ok := NomItem("cout"); !ok || n != "cost" {
		t.Errorf("NomItem(cout) : %v %s", ok, n)
	}
}

func TestValider(t *testing.T) {
	refus := map[string][]string{
		"ctx.width":     {"3", "21", "huit", " 8", ""},
		"icons":         {"Nerd", "emoji"},
		"ctx.libelle":   {"一二", "trop long pour un libellé", "a\tb"},
		"sep.line1":     {"→→→→→", "🚀", "́"},
		"ctx.bar":       {"ab", "abcd", "█▌🚀"},
		"line1":         {"model | inconnu"},
		"hide":          {"model zut"},
		"git.protected": {"main dev:1", "a~b", " main", "main " + strings.Repeat("x", 80)},
	}
	for cle, vals := range refus {
		c, _ := Trouver(cle)
		for _, v := range vals {
			if c.Valider(v) == nil {
				t.Errorf("%s=%q accepté", cle, v)
			}
		}
	}
	accepte := map[string][]string{
		"ctx.bar":       {"#=-", "▰▱▱", "none"},
		"line1":         {"", "ctx|5h", "  model   effort |  | git "},
		"sep.short":     {"", "/", "»»"},
		"git.protected": {"", "main", "main master release/1.0"},
	}
	for cle, vals := range accepte {
		c, _ := Trouver(cle)
		for _, v := range vals {
			if err := c.Valider(v); err != nil {
				t.Errorf("%s=%q refusé : %v", cle, v, err)
			}
		}
	}
	if err := (Cle{Genre: Glyphe}).Valider("🚀"); err == nil || !strings.Contains(err.Error(), "2 colonnes") {
		t.Errorf("message d'un glyphe large : %v", err)
	}
}

func TestDefinirNormalise(t *testing.T) {
	s := Defaut()
	s.Definir("line2", "ctx|5h   7d ||cost")
	if v := s.Valeur("line2"); v != "ctx | 5h 7d | cost" {
		t.Errorf("%q", v)
	}
	s.Definir("line2", " ctx | 5h | 7d | cost")
	if s.Regle("line2") {
		t.Error("une disposition égale au défaut n'est pas un écart")
	}
	s.Masquer("cost", true)
	s.Masquer("model", true)
	if v := s.Valeur("hide"); v != "model cost" {
		t.Errorf("hide : %q", v)
	}
	s.Masquer("model", false)
	s.Masquer("cost", false)
	if !s.EstDefaut() {
		t.Errorf("%q", s.Texte())
	}
}

func TestEcrireRelire(t *testing.T) {
	d := t.TempDir()
	s := Defaut()
	s.Definir("ctx.width", "12")
	s.Definir("icon.pr", "")
	if err := Ecrire(d, s); err != nil {
		t.Fatal(err)
	}
	if r := Charger(d, nil); !r.Egal(s) || !r.Regle("icon.pr") {
		t.Errorf("relu : %q", r.Texte())
	}
	for _, v := range []string{"default", "defaut"} { // defaut : l'ancien nom
		if r := Charger(d, func(k string) string { return map[string]string{"AGENTLINE_STYLE": v}[k] }); !r.EstDefaut() {
			t.Errorf("AGENTLINE_STYLE=%s doit ignorer le fichier", v)
		}
	}
	if err := Ecrire(d, Defaut()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, Fichier)); !os.IsNotExist(err) {
		t.Error("le style par défaut retire le fichier")
	}
	if err := Ecrire(d, Defaut()); err != nil {
		t.Errorf("retirer un fichier absent : %v", err)
	}
}

func TestDesItems(t *testing.T) {
	if l := DesItems("ctx"); len(l) != 4 {
		t.Errorf("ctx : %d options", len(l))
	}
	for _, c := range DesItems("") {
		if len(c.Items) > 0 || c.Genre == Disposition {
			t.Errorf("%s n'est pas générale", c.Nom)
		}
	}
}

// FuzzLire : n'importe quel contenu se lit sans paniquer, ne garde que des valeurs valides, et se
// réécrit à l'identique.
func FuzzLire(f *testing.F) {
	f.Add([]byte(ToutDefaut()))
	f.Add([]byte("ctx.largeur=12\nmasquer=pr\nligne1=ctx|5h\n# x\n=\n"))
	f.Add([]byte("icone.pr=\xff\xfe\nsep.ligne1=\x00"))
	f.Add([]byte("ctx.width=12\nhide=pr\nline1=model|7d cost\nctx.bar=none\n"))
	f.Fuzz(func(t *testing.T, b []byte) {
		s := Lire(b)
		for _, c := range Cles() {
			if err := c.Valider(s.Valeur(c.Nom)); err != nil {
				t.Fatalf("%s=%q gardé : %v", c.Nom, s.Valeur(c.Nom), err)
			}
		}
		if r := Lire([]byte(s.Texte())); !r.Egal(s) || r.Texte() != s.Texte() {
			t.Fatalf("relecture différente :\n%q\n%q", s.Texte(), r.Texte())
		}
	})
}
