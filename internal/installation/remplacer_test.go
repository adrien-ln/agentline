package installation

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestRemplacer : le fichier est remplacé d'un coup, garde ses droits, ne laisse aucun fichier temporaire,
// et un lien symbolique est suivi (sa cible change, le lien reste un lien).
func TestRemplacer(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "settings.json")
	os.WriteFile(f, []byte("{}\n"), 0o600)
	if err := remplacer(f, []byte("{\"a\": 1}\n")); err != nil {
		t.Fatal(err)
	}
	if lire(t, f) != "{\"a\": 1}\n" {
		t.Errorf("contenu : %q", lire(t, f))
	}
	if i, _ := os.Stat(f); runtime.GOOS != "windows" && i.Mode().Perm() != 0o600 {
		t.Errorf("droits : %v", i.Mode().Perm())
	}
	if e, _ := os.ReadDir(d); len(e) != 1 {
		t.Errorf("fichiers temporaires restés : %v", e)
	}
	lien := filepath.Join(d, "lien.json")
	if err := os.Symlink(f, lien); err != nil {
		t.Skipf("pas de lien symbolique ici : %v", err)
	}
	if err := remplacer(lien, []byte("{\"b\": 2}\n")); err != nil {
		t.Fatal(err)
	}
	if i, _ := os.Lstat(lien); i.Mode()&os.ModeSymlink == 0 || lire(t, f) != "{\"b\": 2}\n" {
		t.Errorf("le lien doit rester un lien et sa cible changer : %v %q", i.Mode(), lire(t, f))
	}
}
