package installation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reglagesWT = "{\n    // à moi\n    \"profiles\": \n    {\n        \"defaults\": { \"font\": { \"face\": \"Cascadia Mono\" } },\n        \"list\": [ { \"guid\": \"{1111}\", \"name\": \"Ubuntu\", \"font\": { \"face\": \"Ubuntu Mono\" } }, { \"guid\": \"{2222}\", \"name\": \"PowerShell\" } ]\n    }\n}\n"

// windowsTerminal : une machine Windows dans Windows Terminal (profil guid), avec ses réglages.
func windowsTerminal(t *testing.T, guid string) (*simule, string) {
	t.Helper()
	s := simuler(t, "windows")
	s.Reglages = t.TempDir()
	s.env["WT_SESSION"], s.env["WT_PROFILE_ID"] = "x", guid
	f := filepath.Join(s.localAppData(), "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
	os.MkdirAll(filepath.Dir(f), 0o755)
	os.WriteFile(f, []byte(reglagesWT), 0o644)
	return s, f
}

// TestApercuPolice : Essayer change la police de defaults (une copie .bak au premier changement seulement, le
// repère posé) ; Remettre rend le fichier octet pour octet et efface le repère.
func TestApercuPolice(t *testing.T) {
	s, f := windowsTerminal(t, "{2222}") // PowerShell n'a pas sa propre police : defaults
	a, err := s.PreparerApercu()
	if err != nil || a.Terminal != "Windows Terminal" || a.Avant != "Cascadia Mono" {
		t.Fatalf("préparer : %+v %v", a, err)
	}
	for _, p := range []string{"Hack Nerd Font", "FiraCode Nerd Font"} {
		if err := a.Essayer(p); err != nil {
			t.Fatal(err)
		}
	}
	if v, _ := ChaineJSONC(lire(t, f), "profiles", "defaults", "font", "face"); v != "FiraCode Nerd Font" || copies(f) != 1 || !strings.Contains(lire(t, f), "// à moi") {
		t.Errorf("après deux essais (%d copies) :\n%s", copies(f), lire(t, f))
	}
	if _, err := os.Stat(filepath.Join(s.Reglages, fichierRepere)); err != nil {
		t.Errorf("le repère doit être posé : %v", err)
	}
	if err := a.Remettre(); err != nil || lire(t, f) != reglagesWT {
		t.Errorf("remettre : %v\n%s", err, lire(t, f))
	}
	if _, err := os.Stat(filepath.Join(s.Reglages, fichierRepere)); !os.IsNotExist(err) {
		t.Error("le repère doit être effacé")
	}
}

// TestApercuPoliceProfil : un profil qui a sa propre police : c'est elle que l'aperçu change ; un fichier
// modifié pendant l'aperçu ne perd que la police d'essai.
func TestApercuPoliceProfil(t *testing.T) {
	s, f := windowsTerminal(t, "{1111}")
	a, _ := s.PreparerApercu()
	if a.Avant != "Ubuntu Mono" {
		t.Fatalf("la police du profil : %q", a.Avant)
	}
	a.Essayer("Hack Nerd Font")
	c := lire(t, f)
	if !strings.Contains(c, `"Hack Nerd Font"`) || !strings.Contains(c, `"Cascadia Mono"`) {
		t.Errorf("seul le profil change :\n%s", c)
	}
	os.WriteFile(f, []byte(strings.Replace(c, "// à moi", "// à moi, modifié", 1)), 0o644)
	if err := a.Remettre(); err != nil || !strings.Contains(lire(t, f), "modifié") || !strings.Contains(lire(t, f), `"Ubuntu Mono"`) {
		t.Errorf("fichier modifié pendant l'aperçu : %v\n%s", err, lire(t, f))
	}
}

// TestApercuInterrompu : le repère resté (processus tué) permet de remettre la police, une fois.
func TestApercuInterrompu(t *testing.T) {
	s, f := windowsTerminal(t, "{2222}")
	a, _ := s.PreparerApercu()
	a.Essayer("Hack Nerd Font") // puis plus rien : interrompu
	msg := s.RemettreApercuInterrompu()
	if v, _ := ChaineJSONC(lire(t, f), "profiles", "defaults", "font", "face"); v != "Cascadia Mono" || !strings.Contains(msg, "Cascadia Mono est remise") {
		t.Errorf("remise après interruption : %q\n%s", msg, lire(t, f))
	}
	if s.RemettreApercuInterrompu() != "" {
		t.Error("une seule fois")
	}
}

// TestSansApercu : hors de Windows Terminal et de VS Code, pas d'aperçu.
func TestSansApercu(t *testing.T) {
	s := simuler(t, "windows")
	s.env["TERM_PROGRAM"] = "mintty"
	if _, err := s.PreparerApercu(); err != ErrSansApercu {
		t.Errorf("Git Bash : %v", err)
	}
}
