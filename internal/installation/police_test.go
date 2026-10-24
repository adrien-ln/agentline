package installation

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveFausse : un FiraCode.zip minuscule, avec ce qu'il faut ignorer (Mono, licence) et un chemin piégé.
func archiveFausse(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, n := range []string{"FiraCodeNerdFont-Regular.ttf", "FiraCodeNerdFont-Bold.ttf", "FiraCodeNerdFontMono-Regular.ttf", "LICENSE", "../../piege/FiraCodeNerdFont-Light.ttf"} {
		w, _ := z.Create(n)
		w.Write([]byte("police " + n))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// avecPolice : la simulation interactive, qui répond reponse, avec ces programmes « installés ».
func (s *simule) avecPolice(t *testing.T, reponse bool, programmes ...string) {
	s.Interactif = true
	s.Demander = func(q string) bool {
		fmt.Fprint(s.sortie, q+"\n")
		return reponse
	}
	s.Chercher = func(p string) (string, error) {
		for _, x := range programmes {
			if x == p {
				return "/simule/" + p, nil
			}
		}
		return "", fmt.Errorf("%s introuvable", p)
	}
	zip := archiveFausse(t)
	s.Telecharger = func(url string) ([]byte, error) {
		s.lancees = append(s.lancees, "télécharger "+url)
		return zip, nil
	}
}

func (s *simule) lance(c string) bool {
	for _, l := range s.lancees {
		if l == c {
			return true
		}
	}
	return false
}

func TestPoliceDejaInstallee(t *testing.T) {
	for _, os_ := range []string{"windows", "darwin", "linux"} {
		s := simuler(t, os_)
		s.avecPolice(t, true)
		d := s.dossierPolicesUtilisateur()
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "MesloLGMNerdFont-Regular.ttf"), nil, 0o644)
		if err := Installer(s.Options); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(s.sortie.String(), "[O/n]") || strings.Contains(s.sortie.String(), "Nerd Font trouvée") {
			t.Errorf("%s : rien à proposer\n%s", os_, s.sortie)
		}
	}
}

func TestPoliceSansReponse(t *testing.T) {
	for _, c := range []struct {
		nom        string
		interactif bool
		sans       bool
		reponse    bool
	}{{"non interactif", false, false, true}, {"--sans-police", true, true, true}, {"réponse non", true, false, false}} {
		for _, os_ := range []string{"windows", "darwin", "linux"} {
			s := simuler(t, os_)
			s.avecPolice(t, c.reponse, "scoop", "brew")
			s.Interactif, s.SansPolice = c.interactif, c.sans
			if err := Installer(s.Options); err != nil {
				t.Fatal(err)
			}
			out := s.sortie.String()
			if strings.Contains(out, "[O/n]") != (c.interactif && !c.sans) {
				t.Errorf("%s, %s : question posée ?\n%s", c.nom, os_, out)
			}
			if !strings.Contains(out, s.CommandePolice()) {
				t.Errorf("%s, %s : la commande manque\n%s", c.nom, os_, out)
			}
			for _, l := range s.lancees {
				if strings.Contains(l, "install") || strings.HasPrefix(l, "télécharger") {
					t.Errorf("%s, %s : rien ne doit être installé (%s)", c.nom, os_, l)
				}
			}
		}
	}
}

func TestPoliceCommandes(t *testing.T) {
	for os_, attendu := range map[string]string{
		"darwin": "brew install --cask font-fira-code-nerd-font",
		"linux":  "fc-cache -f",
	} {
		s := simuler(t, os_)
		s.avecPolice(t, true, "brew")
		s.reponses["brew install --cask font-fira-code-nerd-font"] = ""
		if os_ == "darwin" {
			s.Getenv = func(k string) string { return map[string]string{"TERM_PROGRAM": "iTerm.app"}[k] }
		}
		if err := Installer(s.Options); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(s.CommandePolice(), attendu) && os_ == "darwin" {
			t.Errorf("CommandePolice %s : %s", os_, s.CommandePolice())
		}
		if os_ == "darwin" {
			if !s.lance("brew install --cask font-fira-code-nerd-font") || s.lance("télécharger "+ArchivePolice) {
				t.Errorf("darwin : %v", s.lancees)
			}
			if !strings.Contains(s.sortie.String(), "iTerm2 : Settings > Profiles > Text > Font : FiraCode Nerd Font") {
				t.Errorf("darwin : instructions iTerm2\n%s", s.sortie)
			}
		}
	}
}

func TestPoliceLinuxArchive(t *testing.T) {
	s := simuler(t, "linux")
	s.avecPolice(t, true)
	s.env["VTE_VERSION"] = "7600"
	s.reponses["fc-cache -f"] = ""
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(s.Home, ".local", "share", "fonts", "FiraCodeNerdFont")
	m, _ := filepath.Glob(filepath.Join(d, "*"))
	if len(m) != 3 { // Regular, Bold, Light (sorti du chemin piégé) ; ni Mono ni LICENSE
		t.Errorf("polices copiées : %v", m)
	}
	if _, err := os.Stat(filepath.Join(s.Home, ".local", "share", "piege")); err == nil {
		t.Error("un chemin de l'archive ne doit jamais être suivi")
	}
	if !s.lance("fc-cache -f") || !strings.Contains(s.sortie.String(), "GNOME Terminal") {
		t.Errorf("fc-cache et instructions : %v\n%s", s.lancees, s.sortie)
	}
	// La police est maintenant trouvée : plus de question
	s.sortie.Reset()
	if err := Installer(s.Options); err != nil || strings.Contains(s.sortie.String(), "[O/n]") {
		t.Errorf("deuxième installation : %v\n%s", err, s.sortie)
	}
}

func TestPoliceWindows(t *testing.T) {
	const terminalAvant = "{\n    // à moi\n    \"profiles\": \n    {\n        \"defaults\": {},\n        \"list\": [ { \"name\": \"Ubuntu\", \"font\": { \"face\": \"Ubuntu Mono\" } } ]\n    }\n}\n"
	for _, scoop := range []bool{true, false} {
		s := simuler(t, "windows")
		s.env["WT_SESSION"] = "x"
		var programmes []string
		if scoop {
			programmes = []string{"scoop"}
			s.reponses["scoop install nerd-fonts/FiraCode-NF"] = "installé"
		}
		s.avecPolice(t, true, programmes...)
		wt := filepath.Join(s.localAppData(), "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
		vs := filepath.Join(s.Home, "AppData", "Roaming", "Code", "User", "settings.json")
		for f, c := range map[string]string{wt: terminalAvant, vs: "{\n  \"editor.fontSize\": 14\n}\n"} {
			os.MkdirAll(filepath.Dir(f), 0o755)
			os.WriteFile(f, []byte(c), 0o644)
		}
		if err := Installer(s.Options); err != nil {
			t.Fatal(err)
		}
		if scoop != s.lance("scoop install nerd-fonts/FiraCode-NF") || scoop == s.lance("télécharger "+ArchivePolice) {
			t.Errorf("scoop=%v : %v", scoop, s.lancees)
		}
		if !scoop {
			d := filepath.Join(s.localAppData(), "Microsoft", "Windows", "Fonts")
			v, ok := s.reg[clePolices+`\FiraCode Nerd Font Regular (TrueType)`]
			if !ok || v.Texte != filepath.Join(d, "FiraCodeNerdFont-Regular.ttf") || v.Etendu {
				t.Errorf("police déclarée dans HKCU : %+v %v", v, ok)
			}
			if len(s.diffuse) != 2 || s.diffuse[1] != "polices" {
				t.Errorf("WM_FONTCHANGE : %v", s.diffuse)
			}
		}
		if v, _ := ChaineJSONC(lire(t, wt), "profiles", "defaults", "font", "face"); v != NomPolice || copies(wt) != 1 || !strings.Contains(lire(t, wt), "// à moi") {
			t.Errorf("Windows Terminal (%d copies) :\n%s", copies(wt), lire(t, wt))
		}
		if v, _ := ChaineJSONC(lire(t, vs), "terminal.integrated.fontFamily"); v != NomPolice || copies(vs) != 1 {
			t.Errorf("VS Code :\n%s", lire(t, vs))
		}
		out := s.sortie.String()
		if !strings.Contains(out, "le profil « Ubuntu » garde sa police (Ubuntu Mono)") {
			t.Errorf("le profil qui a sa propre police doit être signalé\n%s", out)
		}
		if strings.Contains(out, "Paramètres > Profils") {
			t.Errorf("Windows Terminal est réglé : pas d'instructions\n%s", out)
		}
	}
}

func TestTerminalIllisibleLaisse(t *testing.T) {
	s := simuler(t, "windows")
	wt := filepath.Join(s.localAppData(), "Microsoft", "Windows Terminal", "settings.json")
	os.MkdirAll(filepath.Dir(wt), 0o755)
	casse := "{ \"profiles\": { \"defaults\": { , } }\n"
	os.WriteFile(wt, []byte(casse), 0o644)
	s.reglerTerminaux()
	if lire(t, wt) != casse || copies(wt) != 0 || !strings.Contains(s.sortie.String(), "illisible") {
		t.Errorf("un fichier illisible doit être laissé tel quel\n%s", s.sortie)
	}
}
