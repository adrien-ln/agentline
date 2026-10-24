package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestLire(t *testing.T) {
	for s, attendu := range map[string]Langue{"fr": FR, "fr_FR.UTF-8": FR, "FR-ca": FR, "fr.UTF-8": FR,
		"en_US.UTF-8": EN, "en-GB": EN, "C": EN, "C.UTF-8": EN, "POSIX": EN, "de_DE": EN, "frisian": EN} {
		if l, ok := Lire(s); !ok || l != attendu {
			t.Errorf("Lire(%q) = %s, %v ; attendu %s", s, l, ok, attendu)
		}
	}
	if _, ok := Lire(" "); ok {
		t.Error("Lire(vide) : ok")
	}
}

func TestDetecter(t *testing.T) {
	cas := []struct {
		env     map[string]string
		systeme string
		attendu Langue
	}{
		{nil, "", EN},
		{nil, "fr-FR", FR},
		{nil, "en-US", EN},
		{map[string]string{"LANG": "fr_FR.UTF-8"}, "en-US", FR},
		{map[string]string{"LANG": "fr_FR.UTF-8", "LC_MESSAGES": "en_US.UTF-8"}, "", EN},
		{map[string]string{"LANG": "en_US.UTF-8", "LC_ALL": "fr_FR.UTF-8"}, "", FR},
		{map[string]string{"LANG": "C"}, "fr-FR", EN}, // la première variable définie décide
		{map[string]string{"LANG": "de_DE.UTF-8"}, "fr-FR", EN},
		{map[string]string{"AGENTLINE_LANG": "fr", "LC_ALL": "en_US.UTF-8"}, "", FR},
		{map[string]string{"AGENTLINE_LANG": "EN", "LANG": "fr_FR.UTF-8"}, "fr-FR", EN},
		{map[string]string{"AGENTLINE_LANG": "de", "LANG": "fr_FR.UTF-8"}, "", FR}, // valeur inconnue : ignorée
	}
	for _, c := range cas {
		if l := detecter(func(k string) string { return c.env[k] }, func() string { return c.systeme }); l != c.attendu {
			t.Errorf("%v, système %q : %s, attendu %s", c.env, c.systeme, l, c.attendu)
		}
	}
}

func TestTraduire(t *testing.T) {
	avant := Choisir(EN)
	defer Choisir(avant)
	anglais["test : %d chat"] = "test: %d cat"
	anglais["test : %d chats"] = "test: %d cats"
	defer delete(anglais, "test : %d chat")
	defer delete(anglais, "test : %d chats")
	if s := T("test : %d chat", 3); s != "test: 3 cat" {
		t.Errorf("T en : %q", s)
	}
	if s := T("pas au catalogue"); s != "pas au catalogue" {
		t.Errorf("T sans traduction : %q", s)
	}
	for n, attendu := range map[int]string{0: "test: 0 cats", 1: "test: 1 cat", 2: "test: 2 cats"} {
		if s := N(n, "test : %d chat", "test : %d chats", n); s != attendu {
			t.Errorf("N(%d) en : %q", n, s)
		}
	}
	Choisir(FR)
	for n, attendu := range map[int]string{0: "test : 0 chat", 1: "test : 1 chat", 2: "test : 2 chats"} {
		if s := N(n, "test : %d chat", "test : %d chats", n); s != attendu {
			t.Errorf("N(%d) fr : %q", n, s)
		}
	}
	if Choisir("de") != FR || Courante() != FR {
		t.Error("une langue inconnue ne doit rien changer")
	}
}

var verbe = regexp.MustCompile(`%(\[[0-9]+\])?[-+# 0]*[0-9*]*(\.[0-9*]+)?[a-zA-Z%]`)

// verbes : les verbes % d'un format, triés (un index explicite permet de les réordonner).
func verbes(s string) []string {
	v := verbe.FindAllString(s, -1)
	slices.Sort(v)
	return v
}

// TestCatalogue : chaque texte a sa traduction anglaise, avec les mêmes verbes %, et une seule.
func TestCatalogue(t *testing.T) {
	vus := map[string]string{}
	for _, table := range tables {
		for fr, en := range table {
			if strings.TrimSpace(en) == "" {
				t.Errorf("%q : pas de traduction anglaise", fr)
			}
			if !slices.Equal(verbes(fr), verbes(en)) {
				t.Errorf("%q / %q : verbes %v et %v", fr, en, verbes(fr), verbes(en))
			}
			if a, ok := vus[fr]; ok && a != en {
				t.Errorf("%q traduit deux fois : %q et %q", fr, a, en)
			}
			vus[fr] = en
		}
	}
}

// textesDuCode : les textes littéraux passés à i18n.T, i18n.N et i18n.M dans le code (hors tests), et où.
func textesDuCode(t *testing.T) map[string]string {
	t.Helper()
	racine := filepath.Join("..", "..")
	textes := map[string]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != racine && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			appel, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := appel.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); !ok || x.Name != "i18n" {
				return true
			}
			var args []ast.Expr
			switch sel.Sel.Name {
			case "T", "M":
				args = appel.Args[:1]
			case "N":
				args = appel.Args[1:3]
			default:
				return true
			}
			for _, a := range args {
				if lit, ok := a.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					s, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Errorf("%s : %v", fset.Position(lit.Pos()), err)
					}
					textes[s] = fset.Position(lit.Pos()).String()
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return textes
}

// TestTextesDuCode : chaque texte marqué du code est au catalogue, et chaque entrée du catalogue sert.
func TestTextesDuCode(t *testing.T) {
	textes := textesDuCode(t)
	for s, ou := range textes {
		if _, ok := anglais[s]; !ok {
			t.Errorf("%s : %q n'est pas au catalogue", ou, s)
		}
	}
	for s := range anglais {
		if _, ok := textes[s]; !ok {
			t.Errorf("%q : au catalogue, mais aucun i18n.T, N ou M ne l'utilise", s)
		}
	}
}
