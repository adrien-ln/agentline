package render

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adrien-ln/agentline/internal/theme"
)

// État git du dépôt, mis en cache 5 s (même fichier que l'ancienne version bash).
type etatGit struct {
	branche                        string
	indexe, modifie, neuf, conflit int
	avance, retard                 int
	fiable                         bool // git a répondu : les compteurs sont justes
}

func cleCache(racine string) string { // comme ${racine//[^A-Za-z0-9]/_} : un « _ » par caractère
	var b strings.Builder
	for _, r := range racine {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// nettoyer : au plus une fois par jour, supprime les caches git inutilisés depuis plus d'un jour, ceux
// d'avant le renommage (claude-statusline-git-*) compris.
func nettoyer(tmp string, now int64) {
	repere := filepath.Join(tmp, "agentline-nettoyage")
	dernier := int64(0)
	if b, err := os.ReadFile(repere); err == nil {
		l, _, _ := strings.Cut(string(b), "\n")
		dernier, _ = strconv.ParseInt(strings.TrimSpace(l), 10, 64)
	}
	if now-dernier <= 86400 {
		return
	}
	os.WriteFile(repere, []byte(fmt.Sprintf("%d\n", now)), 0o644)
	fichiers, _ := filepath.Glob(filepath.Join(tmp, "agentline-git-*"))
	anciens, _ := filepath.Glob(filepath.Join(tmp, "claude-statusline-git-*"))
	for _, f := range append(fichiers, anciens...) {
		if fi, err := os.Stat(f); err == nil && time.Since(fi.ModTime()) >= 48*time.Hour { // find -mtime +1
			os.Remove(f)
		}
	}
}

func lireGit(racine, tmp string, now int64, delai time.Duration) etatGit {
	cache := filepath.Join(tmp, "agentline-git-"+cleCache(racine))
	var e etatGit
	if b, err := os.ReadFile(cache); err == nil {
		c := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if t, err := strconv.ParseInt(c[0], 10, 64); err == nil && len(c) >= 8 && now-t < 5 {
			e.branche = c[1]
			e.indexe, _ = strconv.Atoi(c[2])
			e.modifie, _ = strconv.Atoi(c[3])
			e.neuf, _ = strconv.Atoi(c[4])
			e.conflit, _ = strconv.Atoi(c[5])
			e.avance, _ = strconv.Atoi(c[6])
			e.retard, _ = strconv.Atoi(c[7])
			e.fiable = len(c) > 8 && c[8] != "" && c[8] != "0"
			return brancheDeSecours(e, racine)
		}
	}
	ctx, fin := context.WithTimeout(context.Background(), delai)
	defer fin()
	cmd := exec.CommandContext(ctx, "git", "-C", racine, "--no-optional-locks", "status", "--porcelain=v2", "--branch")
	sortie, err := cmd.Output()
	if err != nil {
		sortie = nil // sortie partielle = données fausses
	} else {
		e.fiable = true
	}
	oid := ""
	for _, l := range strings.Split(string(sortie), "\n") {
		switch {
		case strings.HasPrefix(l, "# branch.oid "):
			oid = strings.TrimPrefix(l, "# branch.oid ")
		case strings.HasPrefix(l, "# branch.head "):
			e.branche = strings.TrimPrefix(l, "# branch.head ")
		case strings.HasPrefix(l, "# branch.ab "):
			l = strings.TrimPrefix(l, "# branch.ab +")
			a, _, _ := strings.Cut(l, " ")
			e.avance, _ = strconv.Atoi(a)
			if i := strings.LastIndex(l, "-"); i >= 0 {
				e.retard, _ = strconv.Atoi(l[i+1:])
			}
		case len(l) >= 2 && (l[0] == '1' || l[0] == '2') && l[1] == ' ':
			if len(l) > 2 && l[2] != '.' {
				e.indexe++
			}
			if len(l) > 3 && l[3] != '.' {
				e.modifie++
			}
		case strings.HasPrefix(l, "u "):
			e.conflit++
		case strings.HasPrefix(l, "? "):
			e.neuf++
		}
	}
	if e.branche == "(detached)" {
		e.branche = oid
		if len(oid) > 7 {
			e.branche = oid[:7]
		}
	}
	fiable := 0
	if e.fiable {
		fiable = 1
	}
	// atomique : une autre session dans le même dépôt lit l'ancien cache ou le nouveau, jamais une moitié
	theme.EcrireAtomique(filepath.Dir(cache), filepath.Base(cache), fmt.Sprintf("%d\n%s\n%d\n%d\n%d\n%d\n%d\n%d\n%d\n",
		now, e.branche, e.indexe, e.modifie, e.neuf, e.conflit, e.avance, e.retard, fiable))
	return brancheDeSecours(e, racine)
}

// brancheDeSecours : si git n'a rien donné, la branche se lit dans HEAD, comme pour les règles de thème
// (theme.TeteGit : worktrees compris, nom entier, « feat/x » et non « x ») ; en HEAD détachée, le début
// du commit, comme git.
func brancheDeSecours(e etatGit, racine string) etatGit {
	if e.branche != "" {
		return e
	}
	branche, commit := theme.TeteGit(racine)
	switch {
	case branche != "":
		e.branche = branche
	case len(commit) > 7:
		e.branche = commit[:7]
	default:
		e.branche = commit
	}
	return e
}
