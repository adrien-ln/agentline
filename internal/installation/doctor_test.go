package installation

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// diagnostic : les vérifications par sujet (la première de chaque sujet), et le code de sortie.
func diagnostic(t *testing.T, s *simule) (map[string]Verification, int, string) {
	t.Helper()
	v := Diagnostiquer(s.Options)
	m := map[string]Verification{}
	for _, x := range v {
		if _, ok := m[x.Sujet]; !ok || x.Niveau > m[x.Sujet].Niveau {
			m[x.Sujet] = x
		}
	}
	var b bytes.Buffer
	code := Afficher(&b, v)
	return m, code, b.String()
}

func niveaux(t *testing.T, m map[string]Verification, attendu map[string]Niveau, texte string) {
	t.Helper()
	for sujet, n := range attendu {
		if m[sujet].Niveau != n {
			t.Errorf("%s : niveau %d, attendu %d (%s)\n%s", sujet, m[sujet].Niveau, n, m[sujet].Detail, texte)
		}
	}
}

func TestDoctorRienInstalle(t *testing.T) {
	for _, os_ := range []string{"windows", "darwin", "linux"} {
		s := simuler(t, os_)
		m, code, texte := diagnostic(t, s)
		if code != 1 {
			t.Errorf("%s : code %d", os_, code)
		}
		niveaux(t, m, map[string]Niveau{"Binaire": Probleme, "PATH": Probleme, "Claude Code": Probleme, "Police": Avertissement, "SSH": Bon}, texte)
		for _, sujet := range []string{"Binaire", "PATH", "Claude Code"} {
			if m[sujet].Correction != "agentline install" || !strings.Contains(texte, "→ agentline install") {
				t.Errorf("%s, %s : correction %q", os_, sujet, m[sujet].Correction)
			}
		}
		if m["Police"].Correction != s.CommandePolice() {
			t.Errorf("%s : correction de la police %q", os_, m["Police"].Correction)
		}
	}
}

// installe : la simulation après « agentline install », dans un nouveau terminal (PATH à jour), police présente.
func installe(t *testing.T, os_ string) *simule {
	t.Helper()
	s := simuler(t, os_)
	s.env["SHELL"] = "/bin/zsh"
	if err := Installer(s.Options); err != nil {
		t.Fatal(err)
	}
	s.Source = s.Binaire() // agentline doctor lancé par le binaire installé
	s.env["PATH"] = s.dossierCommandes()
	d := s.dossierPolicesUtilisateur()
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "FiraCodeNerdFont-Regular.ttf"), nil, 0o644)
	return s
}

func TestDoctorToutVaBien(t *testing.T) {
	for _, os_ := range []string{"windows", "darwin", "linux"} {
		s := installe(t, os_)
		m, code, texte := diagnostic(t, s)
		if code != 0 || strings.Contains(texte, "→") {
			t.Errorf("%s : code %d\n%s", os_, code, texte)
		}
		for sujet, v := range m {
			if v.Niveau != Bon {
				t.Errorf("%s, %s : %s", os_, sujet, v.Detail)
			}
		}
		if n := strings.Count(texte, "✓"); n != 5 {
			t.Errorf("%s : %d lignes ✓\n%s", os_, n, texte)
		}
	}
}

func TestDoctorNouveauTerminalNecessaire(t *testing.T) {
	for _, os_ := range []string{"windows", "linux"} {
		s := installe(t, os_)
		s.env["PATH"] = "" // le terminal d'avant l'installation
		m, code, texte := diagnostic(t, s)
		if code != 0 || m["PATH"].Niveau != Avertissement || !strings.Contains(m["PATH"].Correction, "nouveau terminal") {
			t.Errorf("%s : %d %+v\n%s", os_, code, m["PATH"], texte)
		}
	}
}

func TestDoctorProblemes(t *testing.T) {
	s := installe(t, "linux")
	reglages := filepath.Join(s.Claude, "settings.json")

	// statusLine qui lance autre chose
	os.WriteFile(reglages, []byte(`{"statusLine": {"type": "command", "command": "bash ~/.claude/statusline.sh"}}`), 0o644)
	m, code, _ := diagnostic(t, s)
	if code != 1 || m["Claude Code"].Niveau != Probleme || !strings.Contains(m["Claude Code"].Detail, "statusline.sh") {
		t.Errorf("autre statusLine : %d %+v", code, m["Claude Code"])
	}
	os.WriteFile(reglages, []byte(`{"statusLine": `), 0o644)
	if m, _, _ := diagnostic(t, s); !strings.Contains(m["Claude Code"].Detail, "JSON valide") {
		t.Errorf("JSON invalide : %+v", m["Claude Code"])
	}

	// sl-theme qui n'est pas à sl : rien à signaler
	os.WriteFile(filepath.Join(s.dossierCommandes(), "sl-theme"), []byte("#!/bin/sh\necho autre\n"), 0o755)
	if _, _, texte := diagnostic(t, s); strings.Contains(texte, "sl-theme") {
		t.Errorf("sl-theme étranger signalé :\n%s", texte)
	}

	// binaire plus ancien que celui qui pose la question
	src := filepath.Join(t.TempDir(), "agentline-nouveau")
	os.WriteFile(src, []byte("binaire v2"), 0o755)
	s.Source = src
	if m, _, _ := diagnostic(t, s); m["Binaire"].Niveau != Avertissement || !strings.Contains(m["Binaire"].Correction, src+`" install`) {
		t.Errorf("binaire différent : %+v", m["Binaire"])
	}

	// SSH
	s.env["SSH_CONNECTION"] = "1.2.3.4 5 6.7.8.9 22"
	if m, _, _ := diagnostic(t, s); m["SSH"].Niveau != Bon || m["Police"].Niveau != Bon {
		t.Errorf("SSH, statusline complète par défaut : %+v %+v", m["SSH"], m["Police"])
	}
	s.Court = true // ssh.short=yes
	if m, _, _ := diagnostic(t, s); m["SSH"].Niveau != Avertissement || !strings.Contains(m["SSH"].Correction, "ssh.short no") {
		t.Errorf("SSH, ligne courte : %+v", m["SSH"])
	}
}
