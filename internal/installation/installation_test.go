package installation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const reglagesBash = `{
  "theme": "dark",
  "statusLine": {
    "type": "command",
    "command": "bash ~/.claude/statusline/statusline.sh",
    "padding": 0
  },
  "model": "opus"
}
`

const profilAvecBash = "# Mon profil, accentué\nfunction ll { Get-ChildItem -Force @args }\n\n" +
	"# >>> statusline : ajouté par install.sh (gitlab.com/adrien-ln/statusline), retiré par uninstall.sh >>>\n" +
	"function sl-theme { & 'C:\\Program Files\\Git\\bin\\bash.exe' 'C:\\Users\\x\\.claude\\statusline\\bin\\sl-theme' @args }\n" +
	"# <<< statusline <<<\n"

func options(t *testing.T) (Options, *bytes.Buffer) {
	t.Helper()
	h := t.TempDir()
	src := filepath.Join(t.TempDir(), "agentline-construit")
	os.WriteFile(src, []byte("binaire v1"), 0o755)
	var sortie bytes.Buffer
	return Options{Home: h, Claude: filepath.Join(h, ".claude"), Source: src, Profils: []string{filepath.Join(h, "profil.ps1")},
		OS: runtime.GOOS, Horodatage: "20990101-000000", Sortie: &sortie}, &sortie
}

func lire(t *testing.T, f string) string {
	t.Helper()
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func copies(f string) int { m, _ := filepath.Glob(f + ".*.bak"); return len(m) }

func TestDepuisLaVersionBash(t *testing.T) {
	o, sortie := options(t)
	os.MkdirAll(o.Claude, 0o755)
	reglages := filepath.Join(o.Claude, "settings.json")
	os.WriteFile(reglages, []byte(reglagesBash), 0o644)
	profil := o.Profils[0]
	os.WriteFile(profil, []byte(profilAvecBash), 0o644)
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	r := lire(t, reglages)
	var v map[string]any
	if err := json.Unmarshal([]byte(r), &v); err != nil {
		t.Fatalf("settings.json invalide : %v\n%s", err, r)
	}
	sl := v["statusLine"].(map[string]any)
	if sl["command"] != o.Commande() || v["theme"] != "dark" || v["model"] != "opus" {
		t.Errorf("réglages : %v", v)
	}
	if !strings.Contains(o.Commande(), filepath.ToSlash(o.Binaire())) || !strings.HasSuffix(o.Commande(), `" render`) {
		t.Errorf("commande : %s", o.Commande())
	}
	if lire(t, o.Binaire()) != "binaire v1" {
		t.Error("binaire non copié")
	}
	if copies(reglages) != 1 {
		t.Errorf("%d copies de settings.json, attendu 1", copies(reglages))
	}
	// Le profil PowerShell : le bloc de la version bash est retiré (plus de sl-theme), le reste gardé
	if o.windows() {
		if p := lire(t, profil); p != "# Mon profil, accentué\nfunction ll { Get-ChildItem -Force @args }\n" {
			t.Errorf("profil PowerShell :\n%q", p)
		}
		if copies(profil) != 1 {
			t.Errorf("%d copies du profil, attendu 1", copies(profil))
		}
	}
	cmd := filepath.Join(o.dossierCommandes(), "sl-theme")
	if _, err := os.Stat(cmd); err == nil {
		t.Error("plus de commande sl-theme")
	}

	// Réinstallation : rien ne change, aucune nouvelle copie
	sortie.Reset()
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sortie.String(), "Réglages") || strings.Contains(sortie.String(), "PowerShell ") || copies(reglages) != 1 {
		t.Errorf("réinstallation : quelque chose a changé\n%s", sortie)
	}

	// Mise à jour du binaire
	os.WriteFile(o.Source, []byte("binaire v2"), 0o755)
	if err := Installer(o); err != nil || lire(t, o.Binaire()) != "binaire v2" {
		t.Errorf("mise à jour du binaire : %v, %q", err, lire(t, o.Binaire()))
	}

	// Désinstallation : statusLine retirée, autres clés gardées, profil rendu tel qu'avant le bloc
	if err := Desinstaller(o); err != nil {
		t.Fatal(err)
	}
	r = lire(t, reglages)
	if strings.Contains(r, "statusLine") || !json.Valid([]byte(r)) || !strings.Contains(r, `"model": "opus"`) {
		t.Errorf("désinstallation, réglages :\n%s", r)
	}
	if o.windows() && (lire(t, profil) != "# Mon profil, accentué\nfunction ll { Get-ChildItem -Force @args }\n" || copies(profil) != 1) {
		t.Errorf("désinstallation, profil :\n%q", lire(t, profil))
	}
	if _, err := os.Stat(o.Binaire()); err == nil {
		t.Error("le binaire doit être renommé")
	}
}

// installationSl : ce que « sl install » laissait (avant le renommage) : binaire sl, commandes sl-theme,
// fonction du profil PowerShell, statusLine qui lance sl.
func installationSl(t *testing.T, o Options) (ancien string) {
	t.Helper()
	d := o.dossierCommandes()
	os.MkdirAll(d, 0o755)
	ancien = filepath.Join(d, "sl")
	if o.windows() {
		ancien += ".exe"
	}
	os.WriteFile(ancien, []byte("MZ… gitlab.com/adrien-ln/statusline/internal/render …"), 0o755)
	os.WriteFile(filepath.Join(d, "sl-theme"), []byte("#!/bin/sh\n"+marqueCommande+"\nexec \""+filepath.ToSlash(ancien)+"\" theme \"$@\"\n"), 0o755)
	if o.windows() {
		os.WriteFile(filepath.Join(d, "sl-theme.cmd"), []byte("@echo off\r\n"+marqueCmd+"\r\n\"%~dp0sl.exe\" theme %*\r\n"), 0o755)
		os.WriteFile(o.Profils[0], []byte("# à moi\n\n# >>> statusline : ajouté par « sl install » (gitlab.com/adrien-ln/statusline), retiré par « sl uninstall » >>>\n"+
			"function sl-theme { & '"+ancien+"' theme @args }\n# <<< statusline <<<\n"), 0o644)
	}
	os.MkdirAll(o.Claude, 0o755)
	os.WriteFile(filepath.Join(o.Claude, "settings.json"), []byte("{\n  \"model\": \"opus\",\n  "+BlocStatusLine(`"`+filepath.ToSlash(ancien)+`" render`)+"\n}\n"), 0o644)
	return ancien
}

func TestRestesDeSl(t *testing.T) {
	for _, os_ := range []string{"windows", "linux"} {
		t.Run(os_, func(t *testing.T) {
			s := simuler(t, os_)
			s.env["SHELL"] = "/bin/bash"
			ancien := installationSl(t, s.Options)
			d := s.dossierCommandes()
			etranger := filepath.Join(d, "sl") // un fichier de ce nom qui n'est pas à sl (la locomotive ?)
			if os_ != "windows" {
				etranger = filepath.Join(d, "sl.exe")
			}
			os.WriteFile(etranger, []byte("un autre programme"), 0o755)

			// doctor : statusLine qui lance sl, chaque reste signalé, corrigés par « agentline install »
			m, code, texte := diagnostic(t, s)
			if code != 1 || m["Claude Code"].Niveau != Probleme || !strings.Contains(m["Claude Code"].Detail, "l'ancien sl") {
				t.Errorf("statusLine vers sl : %d %+v\n%s", code, m["Claude Code"], texte)
			}
			attendus := 2 // binaire, sl-theme
			if os_ == "windows" {
				attendus = 4 // sl-theme.cmd, profil
			}
			if n := strings.Count(texte, "Ancien sl"); n != attendus || m["Ancien sl"].Niveau != Avertissement || m["Ancien sl"].Correction != "agentline install" {
				t.Errorf("restes signalés : %d, attendu %d\n%s", n, attendus, texte)
			}

			// install : tout est mis de côté (.bak), le profil nettoyé (copie .bak), statusLine remplacée
			if err := Installer(s.Options); err != nil {
				t.Fatal(err)
			}
			for _, f := range []string{ancien, filepath.Join(d, "sl-theme"), filepath.Join(d, "sl-theme.cmd")} {
				if _, err := os.Stat(f); err == nil {
					t.Errorf("%s doit être mis de côté", f)
				}
			}
			for _, f := range []string{ancien, filepath.Join(d, "sl-theme")} {
				if _, err := os.Stat(f + "." + s.Horodatage + ".bak"); err != nil {
					t.Errorf("copie .bak de %s : %v", f, err)
				}
			}
			if lire(t, etranger) != "un autre programme" {
				t.Error("un fichier qui n'est pas à sl doit rester tel quel")
			}
			if os_ == "windows" && (lire(t, s.Profils[0]) != "# à moi\n" || copies(s.Profils[0]) != 1) {
				t.Errorf("profil : %q, %d copies", lire(t, s.Profils[0]), copies(s.Profils[0]))
			}
			if r := lire(t, filepath.Join(s.Claude, "settings.json")); !strings.Contains(r, filepath.ToSlash(s.Binaire())) || strings.Contains(r, filepath.ToSlash(ancien)+`\" render`) {
				t.Errorf("settings.json :\n%s", r)
			}
			if !strings.Contains(s.sortie.String(), "Ancien sl") {
				t.Errorf("compte rendu :\n%s", s.sortie)
			}

			// doctor : plus aucun reste
			s.Source = s.Binaire()
			if _, _, texte := diagnostic(t, s); strings.Contains(texte, "Ancien sl") || strings.Contains(texte, "l'ancien sl") {
				t.Errorf("après install :\n%s", texte)
			}
		})
	}
}

// TestDesinstallerRetireLesRestes : « agentline uninstall » retire aussi les restes de sl.
func TestDesinstallerRetireLesRestes(t *testing.T) {
	s := simuler(t, "windows")
	ancien := installationSl(t, s.Options)
	if err := Desinstaller(s.Options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ancien); err == nil || copies(ancien) != 1 {
		t.Errorf("ancien binaire : %v, %d copies", err, copies(ancien))
	}
	if lire(t, s.Profils[0]) != "# à moi\n" {
		t.Errorf("profil : %q", lire(t, s.Profils[0]))
	}
	if r := lire(t, filepath.Join(s.Claude, "settings.json")); strings.Contains(r, "statusLine") {
		t.Errorf("settings.json :\n%s", r)
	}
}

func TestReglagesAbsentsVidesSansCle(t *testing.T) {
	for nom, depart := range map[string]string{"absent": "", "vide": "{}\n", "sans-cle": "{\n  \"theme\": \"dark\"\n}\n"} {
		o, _ := options(t)
		reglages := filepath.Join(o.Claude, "settings.json")
		if depart != "" {
			os.MkdirAll(o.Claude, 0o755)
			os.WriteFile(reglages, []byte(depart), 0o644)
		}
		if err := Installer(o); err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		r := lire(t, reglages)
		if !json.Valid([]byte(r)) || !strings.Contains(r, `"statusLine"`) {
			t.Errorf("%s : %s", nom, r)
		}
		if nom == "sans-cle" && !strings.Contains(r, `"theme": "dark"`) {
			t.Errorf("sans-cle : clé perdue\n%s", r)
		}
	}
}

func TestRetirerStatusLine(t *testing.T) {
	bloc := BlocStatusLine(`"x" render`)
	for nom, c := range map[string]string{
		"seule":    "{\n  " + bloc + "\n}\n",
		"premiere": "{\n  " + bloc + ",\n  \"theme\": \"dark\"\n}\n",
		"milieu":   "{\n  \"a\": 1,\n  " + bloc + ",\n  \"b\": 2\n}\n",
		"derniere": "{\n  \"theme\": \"dark\",\n  " + bloc + "\n}\n",
	} {
		r, ok := RetirerStatusLine(c)
		if !ok || strings.Contains(r, "statusLine") || !json.Valid([]byte(r)) {
			t.Errorf("%s : %q", nom, r)
		}
	}
}

func TestJSONInvalideRefuse(t *testing.T) {
	o, _ := options(t)
	os.MkdirAll(o.Claude, 0o755)
	reglages := filepath.Join(o.Claude, "settings.json")
	casse := "{\n  \"theme\": \"dark\",\n  \"statusLine\": {\"type\": \"command\"},\n  \"x\": [1, 2\n}\n" // crochet non fermé
	os.WriteFile(reglages, []byte(casse), 0o644)
	if err := Installer(o); err == nil {
		t.Fatal("un settings.json invalide doit être refusé")
	}
	if lire(t, reglages) != casse || copies(reglages) != 0 {
		t.Error("rien ne doit être changé quand le résultat serait invalide")
	}
}

func TestCommandeEtrangereGardee(t *testing.T) {
	o, sortie := options(t)
	cmd := filepath.Join(o.dossierCommandes(), "sl-theme")
	os.MkdirAll(filepath.Dir(cmd), 0o755)
	os.WriteFile(cmd, []byte("#!/bin/sh\necho autre outil\n"), 0o755)
	for _, f := range []func(Options) error{Installer, Desinstaller} {
		if err := f(o); err != nil {
			t.Fatal(err)
		}
	}
	if lire(t, cmd) != "#!/bin/sh\necho autre outil\n" || copies(cmd) != 0 || strings.Contains(sortie.String(), "Ancien sl") {
		t.Error("une commande sl-theme qui n'est pas à sl doit être laissée telle quelle")
	}
}

// TestAncienBlocDuShell : le bloc PATH posé par « sl install » (balises statusline) est remplacé, ou retiré.
func TestAncienBlocDuShell(t *testing.T) {
	ancien := "# à moi\n\n" + ancienDebut + " : ajouté par « sl install » >>>\nexport PATH=x\n" + ancienFin + "\n"
	nouveau := Bloc("export PATH=y")
	if r, res := PoserBloc(ancien, nouveau); res != "remplacé" || r != "# à moi\n\n"+nouveau+"\n" {
		t.Errorf("PoserBloc : %s %q", res, r)
	}
	if r, ok := RetirerBloc(ancien); !ok || r != "# à moi\n" {
		t.Errorf("RetirerBloc : %q", r)
	}
	if !ABloc(ancien) || !ABloc(nouveau) || ABloc("# à moi\n") {
		t.Error("ABloc")
	}
}

func TestCopieNom(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "f")
	os.WriteFile(f+".20990101-000000.bak", nil, 0o644)
	if n := CopieNom(f, "20990101-000000"); n != f+".20990101-000000-1.bak" {
		t.Errorf("CopieNom : %s", n)
	}
}
