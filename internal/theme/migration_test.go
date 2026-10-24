package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func ecrire(t *testing.T, f, contenu string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(f), 0o755)
	if err := os.WriteFile(f, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contenu(t *testing.T, f string) string {
	t.Helper()
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMigrer(t *testing.T) {
	base := t.TempDir()
	ancien, nouveau := filepath.Join(base, "statusline"), filepath.Join(base, "agentline")
	ecrire(t, filepath.Join(ancien, "defaut"), "nord\n")
	ecrire(t, filepath.Join(ancien, "projets"), "gitlab.com/x/y\tdracula\n")
	ecrire(t, filepath.Join(ancien, "themes", "perso.theme"), "# Perso\nTEXTE=ffffff\n")

	if ok, err := Migrer(ancien, nouveau); !ok || err != nil {
		t.Fatalf("première migration : %v %v", ok, err)
	}
	for f, attendu := range map[string]string{"defaut": "nord\n", "projets": "gitlab.com/x/y\tdracula\n", "themes/perso.theme": "# Perso\nTEXTE=ffffff\n"} {
		if c := contenu(t, filepath.Join(nouveau, filepath.FromSlash(f))); c != attendu {
			t.Errorf("%s : %q", f, c)
		}
		if c := contenu(t, filepath.Join(ancien, filepath.FromSlash(f))); c != attendu {
			t.Errorf("l'ancien %s doit rester intact : %q", f, c)
		}
	}
	if s := (Sources{Config: nouveau}); s.ThemeDuProjet("gitlab.com/x/y") != "dracula" {
		t.Error("les réglages copiés doivent être lus")
	}
	if m, _ := filepath.Glob(filepath.Join(base, ".agentline.migration-*")); len(m) != 0 {
		t.Errorf("dossier temporaire resté : %v", m)
	}

	// Une seule fois : le nouveau dossier existe, rien n'est recopié (ni écrasé)
	ecrire(t, filepath.Join(nouveau, "defaut"), "tokyo\n")
	ecrire(t, filepath.Join(ancien, "branches"), "main\tnord\n")
	if ok, err := Migrer(ancien, nouveau); ok || err != nil {
		t.Errorf("deuxième migration : %v %v", ok, err)
	}
	if contenu(t, filepath.Join(nouveau, "defaut")) != "tokyo\n" {
		t.Error("le nouveau dossier ne doit jamais être écrasé")
	}
	if _, err := os.Stat(filepath.Join(nouveau, "branches")); err == nil {
		t.Error("rien n'est recopié après la première fois")
	}
}

func TestMigrerSansAncien(t *testing.T) {
	base := t.TempDir()
	nouveau := filepath.Join(base, "sous", "agentline")
	if ok, err := Migrer(filepath.Join(base, "statusline"), nouveau); ok || err != nil {
		t.Errorf("sans ancien dossier : %v %v", ok, err)
	}
	if _, err := os.Stat(nouveau); err == nil {
		t.Error("sans ancien dossier, le nouveau n'est pas créé")
	}
	// un fichier à la place de l'ancien dossier : ignoré
	ecrire(t, filepath.Join(base, "fichier"), "x")
	if ok, err := Migrer(filepath.Join(base, "fichier"), nouveau); ok || err != nil {
		t.Errorf("ancien qui n'est pas un dossier : %v %v", ok, err)
	}
}
