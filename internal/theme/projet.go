package theme

import (
	"os"
	"regexp"
	"strings"
)

// Un projet est un dépôt Git, reconnu par l'URL de son remote (la même sur le PC et sur le VPS) ;
// sans remote, ou hors Git, par son chemin. Même logique que l'ancienne version bash, pour des identifiants identiques.

var (
	disqueGitBash  = regexp.MustCompile(`^/([a-zA-Z])(/|$)`)
	racineDisque   = regexp.MustCompile(`^.:/$`)
	scpUtilisateur = regexp.MustCompile(`^[^/]+@[^/]+:`)
	scpHote        = regexp.MustCompile(`^[^/:]{2,}:`)
)

// CheminNormaliser : « /c/x » et « C:\x » donnent « c:/x » sous Windows ; sans la barre finale.
func (s Sources) CheminNormaliser(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	if s.Windows {
		if m := disqueGitBash.FindStringSubmatch(p); m != nil {
			reste := ""
			if len(p) > 3 {
				reste = p[3:]
			}
			p = m[1] + ":/" + reste
		}
		p = strings.ToLower(p)
	}
	if strings.HasSuffix(p, "/") && len(p) > 1 && !racineDisque.MatchString(p) {
		p = p[:len(p)-1]
	}
	return p
}

// RemoteNormaliser : URL de remote -> « hôte/chemin » en minuscules (ssh, https et scp donnent le même).
func (s Sources) RemoteNormaliser(u string) string {
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")
	switch {
	case strings.Contains(u, "://"): // ssh://git@hôte:port/chemin, https://user@hôte/chemin
		_, u, _ = strings.Cut(u, "://")
		if _, apres, ok := strings.Cut(u, "@"); ok {
			u = apres
		}
		hote, reste, ok := strings.Cut(u, "/")
		if !ok {
			reste = u
		}
		hote, _, _ = strings.Cut(hote, ":")
		u = hote + "/" + reste
	case scpUtilisateur.MatchString(u) || scpHote.MatchString(u): // git@hôte:chemin, hôte:chemin
		if _, apres, ok := strings.Cut(u, "@"); ok {
			u = apres
		}
		u = strings.Replace(u, ":", "/", 1)
	default: // remote local (chemin)
		return s.CheminNormaliser(u)
	}
	return strings.ToLower(u)
}

// DepotRacine : la racine du dépôt Git qui contient le dossier, vide hors Git.
func DepotRacine(d string) string {
	r := strings.ReplaceAll(d, `\`, "/")
	for r != "" {
		if _, err := os.Stat(r + "/.git"); err == nil {
			return r
		}
		if i := strings.LastIndex(r, "/"); i >= 0 {
			r = r[:i]
		} else {
			r = ""
		}
	}
	return ""
}

func absolu(p string) bool { return strings.HasPrefix(p, "/") || (len(p) > 1 && p[1] == ':') }

func lignePropre(f string) (string, bool) { // comme « read -r » : espaces de bord retirés, puis \r
	b, err := os.ReadFile(f)
	if err != nil {
		return "", false
	}
	l, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimRight(strings.Trim(l, " \t"), "\r"), true
}

// dossiersGit : le dossier Git de la racine r (celui du worktree, s'il en est un) et le dossier commun à
// tous ses worktrees, où se trouve la configuration.
func dossiersGit(r string) (git, commun string) {
	fi, err := os.Stat(r + "/.git")
	if err != nil || !fi.Mode().IsRegular() {
		return r + "/.git", r + "/.git"
	}
	g, _ := lignePropre(r + "/.git") // worktree : « gitdir: … »
	g = strings.TrimPrefix(g, "gitdir: ")
	if !absolu(g) {
		g = r + "/" + g
	}
	c := "."
	if l, ok := lignePropre(g + "/commondir"); ok {
		c = l
	}
	if !absolu(c) {
		c = g + "/" + c
	}
	return g, c
}

// BrancheDe : la branche courante du dépôt qui contient le dossier ; vide hors Git ou en HEAD détachée.
func BrancheDe(d string) string {
	r := DepotRacine(d)
	if r == "" {
		return ""
	}
	b, _ := TeteGit(r)
	return b
}

// TeteGit : ce que pointe HEAD dans le dépôt de racine r (worktrees et sous-modules compris, où .git est
// un fichier) : la branche entière (« feat/x »), ou en HEAD détachée le commit ; vide si illisible.
func TeteGit(r string) (branche, commit string) {
	g, _ := dossiersGit(r)
	l, _ := lignePropre(g + "/HEAD")
	if b, ok := strings.CutPrefix(l, "ref: refs/heads/"); ok {
		return b, ""
	}
	if strings.HasPrefix(l, "ref:") {
		return "", ""
	}
	return "", l
}

// ProjetID : l'identifiant du projet qui contient le dossier.
func (s Sources) ProjetID(d string) string {
	r := DepotRacine(d)
	if r == "" {
		return s.CheminNormaliser(d)
	}
	_, commun := dossiersGit(r)
	cfg := commun + "/config"
	url, premier := "", ""
	if b, err := os.ReadFile(cfg); err == nil {
		section := ""
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimLeft(strings.TrimRight(l, "\r"), " \t\v\f\r\n")
			switch {
			case strings.HasPrefix(l, "["):
				section = l
			case strings.HasPrefix(l, "url") && strings.Contains(l, "="):
				if !strings.HasPrefix(section, `[remote "`) || !strings.HasSuffix(section, `"]`) {
					continue
				}
				_, v, _ := strings.Cut(l, "=")
				v = strings.Trim(v, " \t\v\f\r\n")
				if premier == "" {
					premier = v
				}
				if section == `[remote "origin"]` {
					url = v
				}
			}
		}
	}
	if url == "" {
		url = premier
	}
	if url != "" {
		return s.RemoteNormaliser(url)
	}
	return s.CheminNormaliser(r)
}
