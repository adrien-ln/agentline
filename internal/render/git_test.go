package render

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBrancheDeSecours : sans réponse de git, la branche se lit dans HEAD : nom entier (feat/x), worktree
// (.git est un fichier « gitdir: »), HEAD détachée (début du commit) ; une branche déjà connue est gardée.
func TestBrancheDeSecours(t *testing.T) {
	d := t.TempDir()
	depot := filepath.Join(d, "depot")
	os.MkdirAll(filepath.Join(depot, ".git"), 0o755)
	os.WriteFile(filepath.Join(depot, ".git", "HEAD"), []byte("ref: refs/heads/feat/x\n"), 0o644)
	if e := brancheDeSecours(etatGit{}, depot); e.branche != "feat/x" {
		t.Errorf("dépôt : %q", e.branche)
	}
	arbre, gitdir := filepath.Join(d, "arbre"), filepath.Join(depot, ".git", "worktrees", "arbre")
	os.MkdirAll(arbre, 0o755)
	os.MkdirAll(gitdir, 0o755)
	os.WriteFile(filepath.Join(arbre, ".git"), []byte("gitdir: "+filepath.ToSlash(gitdir)+"\n"), 0o644)
	os.WriteFile(filepath.Join(gitdir, "HEAD"), []byte("0123456789abcdef\n"), 0o644)
	if e := brancheDeSecours(etatGit{}, filepath.ToSlash(arbre)); e.branche != "0123456" {
		t.Errorf("worktree en HEAD détachée : %q", e.branche)
	}
	if e := brancheDeSecours(etatGit{branche: "main"}, depot); e.branche != "main" {
		t.Errorf("branche connue : %q", e.branche)
	}
}
