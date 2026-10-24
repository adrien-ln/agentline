package installation

import (
	"strings"
	"testing"
)

// Le format de Windows Terminal : clé et accolade sur deux lignes, quatre espaces, commentaires.
const reglagesTerminal = `{
    "$help": "https://aka.ms/terminal-documentation",
    // mes réglages
    "profiles":
    {
        "defaults":
        {
            "font":
            {
                "face": "Cascadia Mono"
            }
        },
        "list":
        [
            { "name": "PowerShell", /* le mien */ "font": { "face": "Consolas" } },
            { "name": "Git Bash", },
        ]
    }
}
`

func TestPoserChaineJSONC(t *testing.T) {
	r, res, err := PoserChaineJSONC(reglagesTerminal, "FiraCode Nerd Font", "profiles", "defaults", "font", "face")
	if err != nil || res != "remplacé" || r != strings.Replace(reglagesTerminal, `"Cascadia Mono"`, `"FiraCode Nerd Font"`, 1) {
		t.Fatalf("remplacement : %v %s\n%s", err, res, r)
	}
	if _, res, _ := PoserChaineJSONC(r, "FiraCode Nerd Font", "profiles", "defaults", "font", "face"); res != "identique" {
		t.Errorf("deuxième passage : %s", res)
	}

	cas := map[string]string{
		"defaults vide": "{\n    \"profiles\": {\n        \"defaults\": {},\n        \"list\": []\n    }\n}\n",
		"sans defaults": "{\n    \"profiles\": {\n        \"list\": []\n    }\n}\n",
		"sans profiles": "{\n    \"theme\": \"dark\"\n}\n",
		"vide":          "",
		"crlf":          "{\r\n  \"profiles\": {\r\n    \"list\": [],\r\n  }\r\n}\r\n",
		"sur une ligne": `{"profiles": {"list": []}}`,
	}
	for nom, c := range cas {
		r, res, err := PoserChaineJSONC(c, "FiraCode Nerd Font", "profiles", "defaults", "font", "face")
		if err != nil || res != "ajouté" {
			t.Errorf("%s : %v %s", nom, err, res)
			continue
		}
		if err := jsoncValide(r); err != nil {
			t.Errorf("%s : résultat invalide (%v)\n%s", nom, err, r)
		}
		if v, ok := ChaineJSONC(r, "profiles", "defaults", "font", "face"); !ok || v != "FiraCode Nerd Font" {
			t.Errorf("%s : valeur relue %q\n%s", nom, v, r)
		}
		if nom == "crlf" && strings.Count(r, "\n") != strings.Count(r, "\r\n") {
			t.Errorf("crlf : fins de ligne mélangées\n%q", r)
		}
		if nom == "sans defaults" && !strings.Contains(r, "        \"defaults\": {\n            \"font\": {\n                \"face\": \"FiraCode Nerd Font\"\n            }\n        },\n        \"list\"") {
			t.Errorf("sans defaults : retrait\n%s", r)
		}
	}

	// VS Code : une clé à points, au premier niveau
	vs := "{\n\t// éditeur\n\t\"editor.fontSize\": 14,\n}\n"
	r, res, err = PoserChaineJSONC(vs, "FiraCode Nerd Font", "terminal.integrated.fontFamily")
	if err != nil || res != "ajouté" || r != "{\n\t// éditeur\n\t\"terminal.integrated.fontFamily\": \"FiraCode Nerd Font\",\n\t\"editor.fontSize\": 14,\n}\n" {
		t.Errorf("VS Code : %v %s\n%s", err, res, r)
	}

	// profiles en tableau (ancien format) : erreur, rien n'est changé
	if _, _, err := PoserChaineJSONC(`{"profiles": []}`, "x", "profiles", "defaults", "font", "face"); err == nil {
		t.Error("profiles en tableau : erreur attendue")
	}
}

func TestJSONCInvalide(t *testing.T) {
	for _, c := range []string{`{"a": }`, `{"a": 1`, `{"a": 1} x`, `{"a": /* 1}`, `{"a": tru}`, `{a: 1}`} {
		if jsoncValide(c) == nil {
			t.Errorf("%q devrait être refusé", c)
		}
		if _, _, err := PoserChaineJSONC(c, "x", "a"); err == nil {
			t.Errorf("%q : PoserChaineJSONC devrait échouer", c)
		}
	}
	if err := jsoncValide(bom + "{\"a\": [1, -2.5e3, true, null, \"x\\\"y\",], // fin\n}"); err != nil {
		t.Errorf("JSONC valide refusé : %v", err)
	}
}
