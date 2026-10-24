package installation

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registreFaux : HKCU en mémoire.
type registreFaux map[string]ValeurRegistre

func (r registreFaux) Lire(cle, nom string) (ValeurRegistre, bool, error) {
	v, ok := r[cle+`\`+nom]
	return v, ok, nil
}

func (r registreFaux) Ecrire(cle, nom string, v ValeurRegistre) error {
	r[cle+`\`+nom] = v
	return nil
}

// simule : une installation sur l'OS donné, dans des dossiers temporaires, sans rien de réel : un
// environnement en table, aucune commande (lancees les note), un registre en mémoire.
type simule struct {
	Options
	env      map[string]string
	reg      registreFaux
	lancees  []string
	reponses map[string]string // commande lancée -> sa sortie
	sortie   *bytes.Buffer
	diffuse  []string
}

func simuler(t *testing.T, os_ string) *simule {
	t.Helper()
	o, sortie := options(t)
	s := &simule{Options: o, env: map[string]string{}, reg: registreFaux{}, reponses: map[string]string{}, sortie: sortie}
	s.OS = os_
	s.Getenv = func(k string) string { return s.env[k] }
	s.Lancer = func(nom string, args ...string) (string, error) {
		l := strings.TrimSpace(nom + " " + strings.Join(args, " "))
		s.lancees = append(s.lancees, l)
		if r, ok := s.reponses[l]; ok {
			return r, nil
		}
		return "", fmt.Errorf("%s : introuvable", nom)
	}
	if os_ == "windows" {
		s.Registre = s.reg
		s.Diffuser = func(q string) bool { s.diffuse = append(s.diffuse, q); return true }
	}
	return s
}

func (s *simule) path() ValeurRegistre { return s.reg[cleEnvironnement+`\Path`] }

func TestPathWindows(t *testing.T) {
	s := simuler(t, "windows")
	s.env["USERPROFILE"] = s.Home
	avant := `%USERPROFILE%\AppData\Local\Microsoft\WindowsApps;C:\Outils`
	s.reg[cleEnvironnement+`\Path`] = ValeurRegistre{Texte: avant, Etendu: true}
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	d := s.dossierCommandes()
	if p := s.path(); p.Texte != avant+";"+d || !p.Etendu {
		t.Errorf("PATH après installation : %+v", p)
	}
	if len(s.diffuse) != 1 || s.diffuse[0] != "Environment" {
		t.Errorf("WM_SETTINGCHANGE : %v", s.diffuse)
	}
	if m, _ := filepath.Glob(filepath.Join(d, "path-utilisateur.*.bak")); len(m) != 1 || lire(t, m[0]) != avant+"\n" {
		t.Errorf("copie de l'ancien PATH : %v", m)
	}
	if !strings.Contains(s.sortie.String(), "ajouté au PATH utilisateur") {
		t.Errorf("sortie :\n%s", s.sortie)
	}
	if _, err := os.Stat(filepath.Join(d, ".agentline-path")); err != nil {
		t.Errorf("marque du PATH : %v", err)
	}

	// Réinstallation : rien ne change
	s.sortie.Reset()
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if s.path().Texte != avant+";"+d || strings.Contains(s.sortie.String(), "PATH utilisateur (") {
		t.Errorf("réinstallation : %+v\n%s", s.path(), s.sortie)
	}

	// Désinstallation : PATH rendu tel qu'avant, marque retirée
	if err := Desinstaller(s.Options); err != nil {
		t.Fatal(err)
	}
	if p := s.path(); p.Texte != avant || !p.Etendu {
		t.Errorf("PATH après désinstallation : %+v", p)
	}
	if _, err := os.Stat(filepath.Join(d, ".agentline-path")); err == nil {
		t.Error("la marque du PATH doit être retirée")
	}
}

// TestPathWindowsAjouteParSl : le dossier ajouté au PATH par « sl install » (marque .sl-path) est repris :
// « agentline uninstall » l'en retire.
func TestPathWindowsAjouteParSl(t *testing.T) {
	s := simuler(t, "windows")
	d := s.dossierCommandes()
	s.reg[cleEnvironnement+`\Path`] = ValeurRegistre{Texte: `C:\Outils;` + d, Etendu: true}
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, ".sl-path"), []byte(d+"\n"), 0o644)
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, ".agentline-path")); err != nil {
		t.Errorf("marque reprise : %v", err)
	}
	if err := Desinstaller(s.Options); err != nil {
		t.Fatal(err)
	}
	if p := s.path(); p.Texte != `C:\Outils` {
		t.Errorf("PATH après désinstallation : %+v", p)
	}
	for _, f := range []string{".sl-path", ".agentline-path"} {
		if _, err := os.Stat(filepath.Join(d, f)); err == nil {
			t.Errorf("%s doit être retiré", f)
		}
	}
}

func TestPathWindowsDejaPresent(t *testing.T) {
	s := simuler(t, "windows")
	s.env["USERPROFILE"] = s.Home
	avant := `C:\Outils;%USERPROFILE%\BIN\` // déjà là, écrit autrement, en REG_SZ
	s.reg[cleEnvironnement+`\Path`] = ValeurRegistre{Texte: avant}
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if p := s.path(); p.Texte != avant || p.Etendu {
		t.Errorf("le PATH ne doit pas changer : %+v", p)
	}
	if err := Desinstaller(s.Options); err != nil {
		t.Fatal(err)
	}
	if s.path().Texte != avant {
		t.Errorf("la désinstallation ne retire pas une entrée qu'elle n'a pas ajoutée : %+v", s.path())
	}
}

func TestPathWindowsAbsent(t *testing.T) {
	s := simuler(t, "windows")
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if p := s.path(); p.Texte != s.dossierCommandes() || !p.Etendu {
		t.Errorf("PATH créé : %+v", p)
	}
	if m, _ := filepath.Glob(filepath.Join(s.dossierCommandes(), "path-utilisateur.*.bak")); len(m) != 0 {
		t.Errorf("pas de copie quand la valeur n'existait pas : %v", m)
	}
}

func TestPathShell(t *testing.T) {
	for _, c := range []struct{ os_, shell, fichier, ligne string }{
		{"linux", "/bin/bash", ".bashrc", `export PATH="$HOME/.local/bin":"$PATH"`},
		{"linux", "/usr/bin/zsh", ".zshrc", `export PATH="$HOME/.local/bin":"$PATH"`},
		{"linux", "/usr/bin/fish", ".config/fish/config.fish", `or set -gx PATH "$HOME/.local/bin" $PATH`},
		{"darwin", "", ".zshrc", `export PATH="$HOME/.local/bin":"$PATH"`}, // zsh : le shell par défaut de macOS
		{"darwin", "/bin/bash", ".bash_profile", `export PATH="$HOME/.local/bin":"$PATH"`},
		{"linux", "/bin/dash", ".profile", `export PATH="$HOME/.local/bin":"$PATH"`},
	} {
		t.Run(c.os_+c.shell, func(t *testing.T) {
			s := simuler(t, c.os_)
			s.env["SHELL"], s.env["PATH"] = c.shell, "/usr/bin:/bin"
			f := filepath.Join(s.Home, filepath.FromSlash(c.fichier))
			os.MkdirAll(filepath.Dir(f), 0o755)
			os.WriteFile(f, []byte("# à moi\nalias ll='ls -l'\n"), 0o644)
			if err := Installer(s.Options); err != nil {
				t.Fatal(err)
			}
			r := lire(t, f)
			if !strings.HasPrefix(r, "# à moi\nalias ll='ls -l'\n\n"+blocDebut) || !strings.Contains(r, c.ligne) || copies(f) != 1 {
				t.Errorf("%s (%d copies) :\n%s", f, copies(f), r)
			}
			if !strings.Contains(s.sortie.String(), "ajouté au PATH dans "+f) {
				t.Errorf("sortie :\n%s", s.sortie)
			}
			// Réinstallation : rien ne change, aucune copie de plus
			if err := Installer(s.Options); err != nil || lire(t, f) != r || copies(f) != 1 {
				t.Errorf("réinstallation : %v, %d copies", err, copies(f))
			}
			// Même quand le nouveau terminal a le dossier dans son PATH (grâce au bloc)
			s.env["PATH"] = s.dossierCommandes() + ":/usr/bin"
			if err := Installer(s.Options); err != nil || lire(t, f) != r {
				t.Errorf("réinstallation avec le PATH à jour : %v", err)
			}
			if err := Desinstaller(s.Options); err != nil {
				t.Fatal(err)
			}
			if lire(t, f) != "# à moi\nalias ll='ls -l'\n" {
				t.Errorf("désinstallation : %q", lire(t, f))
			}
		})
	}
}

func TestPathShellDejaDansLePath(t *testing.T) {
	s := simuler(t, "linux")
	s.env["SHELL"] = "/bin/bash"
	s.env["PATH"] = "/usr/bin:" + s.dossierCommandes() + "/" // ajouté par ~/.profile, par exemple
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Home, ".bashrc")); err == nil {
		t.Error("le dossier est déjà dans le PATH : ~/.bashrc ne doit pas être créé")
	}
}

func TestPathShellInconnu(t *testing.T) {
	s := simuler(t, "linux")
	s.env["SHELL"] = "/bin/tcsh"
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.sortie.String(), "À faire") || !strings.Contains(s.sortie.String(), "tcsh") {
		t.Errorf("sortie :\n%s", s.sortie)
	}
}

// TestProfilPowerShellIntouche : sans bloc de sl, le profil PowerShell n'est ni créé ni modifié.
func TestProfilPowerShellIntouche(t *testing.T) {
	s := simuler(t, "windows")
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Profils[0]); err == nil {
		t.Error("le profil ne doit pas être créé")
	}
	os.WriteFile(s.Profils[0], []byte("# à moi\n"), 0o644)
	for _, f := range []func(Options) error{Installer, Desinstaller} {
		if err := f(s.Options); err != nil {
			t.Fatal(err)
		}
	}
	if lire(t, s.Profils[0]) != "# à moi\n" || copies(s.Profils[0]) != 0 {
		t.Errorf("profil modifié : %q", lire(t, s.Profils[0]))
	}
}

func TestRetirerDuPathGardeLesAutres(t *testing.T) {
	o := Options{OS: "windows", Getenv: func(string) string { return "" }}
	r, ok := o.RetirerDuPath(`C:\a;;C:\Users\x\bin;C:\b;c:\users\X\Bin\`, `C:\Users\x\bin`)
	if !ok || r != `C:\a;;C:\b` {
		t.Errorf("RetirerDuPath : %q", r)
	}
	u := Options{OS: "linux"}
	if r, ok := u.AjouterAuPath("/usr/bin:/bin:", "/h/.local/bin"); !ok || r != "/usr/bin:/bin:/h/.local/bin" {
		t.Errorf("AjouterAuPath : %q", r)
	}
}
