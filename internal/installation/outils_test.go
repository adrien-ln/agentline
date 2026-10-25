package installation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutilsAbsents : un outil qui n'est pas installé n'est ni configuré ni signalé.
func TestOutilsAbsents(t *testing.T) {
	o, sortie := options(t)
	o.Interactif, o.SansPolice = true, true
	o.Demander = func(q string) bool { t.Errorf("question sans outil installé : %s", q); return false }
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	for _, x := range outils {
		if _, err := os.Stat(x.reglages(o)); !os.IsNotExist(err) {
			t.Errorf("%s : réglages créés", x.nom)
		}
	}
	if strings.Contains(sortie.String(), "Antigravity") || len(o.verifierOutils()) != 0 {
		t.Errorf("rien à dire des outils absents :\n%s", sortie)
	}
}

// TestOutilsConfigures : installé, l'outil est configuré après un oui (copie .bak, reste du fichier
// gardé), pas après un non, et sans terminal on dit seulement comment faire ; doctor suit ; uninstall
// retire notre statusLine et seulement elle.
func TestOutilsConfigures(t *testing.T) {
	o, sortie := options(t)
	antigravity, copilot := outils[0].reglages(o), outils[1].reglages(o)
	os.MkdirAll(filepath.Dir(antigravity), 0o755)
	os.WriteFile(antigravity, []byte("{\n  \"theme\": \"dark\"\n}\n"), 0o644)
	os.MkdirAll(filepath.Dir(copilot), 0o755)

	// sans terminal : un conseil, rien d'écrit
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	if _, ok := commandeDe(antigravity); ok || !strings.Contains(sortie.String(), "dans un terminal") {
		t.Errorf("sans terminal, rien ne doit être écrit :\n%s", sortie)
	}
	if m, _, _ := diagnostic(t, &simule{Options: o}); m["Antigravity CLI"].Niveau != Avertissement {
		t.Errorf("doctor, outil sans statusline : %+v", m["Antigravity CLI"])
	}

	// oui pour Antigravity, non pour Copilot
	o.Interactif, o.SansPolice = true, true // la police : sans question
	o.Demander = func(q string) bool { return strings.Contains(q, "Antigravity") }
	sortie.Reset()
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	c, ok := commandeDe(antigravity)
	if !ok || c != o.Commande()+" --from antigravity" || !strings.Contains(lire(t, antigravity), `"theme": "dark"`) || copies(antigravity) != 1 {
		t.Errorf("Antigravity : %q\n%s", c, lire(t, antigravity))
	}
	if _, err := os.Stat(copilot); !os.IsNotExist(err) {
		t.Error("Copilot : non, donc rien d'écrit")
	}
	if m, _, _ := diagnostic(t, &simule{Options: o}); m["Antigravity CLI"].Niveau != Bon {
		t.Errorf("doctor, après install : %+v", m["Antigravity CLI"])
	}

	// oui pour Copilot : la note sur la fonction expérimentale ; Antigravity déjà à jour
	o.Demander = func(string) bool { return true }
	sortie.Reset()
	if err := Installer(o); err != nil {
		t.Fatal(err)
	}
	if c, _ := commandeDe(copilot); c != o.Commande()+" --from copilot" || !strings.Contains(sortie.String(), "/experimental") {
		t.Errorf("Copilot : %q\n%s", c, sortie)
	}
	if copies(antigravity) != 1 {
		t.Error("Antigravity déjà à jour : pas de nouvelle copie")
	}

	// une statusLine qui n'est pas la nôtre reste à sa place
	os.WriteFile(copilot, []byte(`{"statusLine": {"type": "command", "command": "~/mon-script.sh"}}`), 0o644)
	if err := Desinstaller(o); err != nil {
		t.Fatal(err)
	}
	if _, ok := commandeDe(antigravity); ok || !strings.Contains(lire(t, antigravity), `"theme": "dark"`) {
		t.Errorf("uninstall : notre statusLine retirée, le reste gardé :\n%s", lire(t, antigravity))
	}
	if c, _ := commandeDe(copilot); c != "~/mon-script.sh" {
		t.Errorf("uninstall ne touche pas une statusLine étrangère : %q", c)
	}
}
