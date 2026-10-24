package installation

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// --- Le dossier des commandes dans le PATH ---

// separateurPath : « ; » sous Windows, « : » ailleurs (selon l'OS installé, pas celui des tests).
func (o Options) separateurPath() string {
	if o.windows() {
		return ";"
	}
	return ":"
}

// memeDossier : deux entrées du PATH désignent-elles le même dossier ? Sous Windows, sans tenir compte
// de la casse ni des barres finales, %VARIABLES% développées.
func (o Options) memeDossier(a, b string) bool {
	n := func(s string) string {
		s = strings.TrimSpace(s)
		if o.windows() {
			s = developper(s, o.env)
			s = strings.ToLower(strings.ReplaceAll(s, "/", `\`))
			return strings.TrimRight(s, `\`)
		}
		return strings.TrimRight(s, "/")
	}
	return n(a) != "" && n(a) == n(b)
}

func (o Options) dansPath(d string) bool { return o.dansListe(o.env("PATH"), d) }

// decouper : les entrées d'une liste PATH. Hors Windows, une lettre seule suivie d'une entrée qui
// commence par « \ » est recollée (« C:\… ») : jamais le cas d'un vrai PATH Unix, mais celui des tests
// qui simulent Linux ou macOS sur une machine Windows.
func (o Options) decouper(liste string) []string {
	p := strings.Split(liste, o.separateurPath())
	if o.windows() {
		return p
	}
	var r []string
	for i := 0; i < len(p); i++ {
		if len(p[i]) == 1 && i+1 < len(p) && strings.HasPrefix(p[i+1], `\`) {
			r = append(r, p[i]+":"+p[i+1])
			i++
			continue
		}
		r = append(r, p[i])
	}
	return r
}

func (o Options) dansListe(liste, d string) bool {
	for _, p := range o.decouper(liste) {
		if o.memeDossier(p, d) {
			return true
		}
	}
	return false
}

var variableWindows = regexp.MustCompile(`%([^%;]+)%`)

// developper : les %VARIABLES% d'une valeur REG_EXPAND_SZ (celles qui sont inconnues restent telles quelles).
func developper(s string, getenv func(string) string) string {
	return variableWindows.ReplaceAllStringFunc(s, func(v string) string {
		if x := getenv(strings.Trim(v, "%")); x != "" {
			return x
		}
		return v
	})
}

// AjouterAuPath : la liste (« ; » ou « : ») avec le dossier en dernier ; faux s'il y est déjà.
func (o Options) AjouterAuPath(liste, d string) (string, bool) {
	if o.dansListe(liste, d) {
		return liste, false
	}
	sep := o.separateurPath()
	if l := strings.TrimRight(liste, sep); l != "" {
		return l + sep + d, true
	}
	return d, true
}

// RetirerDuPath : la liste sans les entrées du dossier, les autres gardées telles quelles ; faux s'il n'y était pas.
func (o Options) RetirerDuPath(liste, d string) (string, bool) {
	sep := o.separateurPath()
	var garde []string
	retire := false
	for _, p := range o.decouper(liste) {
		if o.memeDossier(p, d) {
			retire = true
			continue
		}
		garde = append(garde, p)
	}
	if !retire {
		return liste, false
	}
	return strings.Join(garde, sep), true
}

// --- Linux et macOS : un bloc balisé dans le fichier que lit le shell ---

// shell : le nom du shell de l'utilisateur ($SHELL) ; à défaut, celui du système (zsh sous macOS).
func (o Options) shell() string {
	if s := o.env("SHELL"); s != "" {
		return path.Base(s)
	}
	if o.OS == "darwin" {
		return "zsh"
	}
	return "bash"
}

// fichierShell : le fichier lu par le shell à l'ouverture d'un terminal, et le bloc qui y ajoute le
// dossier des commandes au PATH ; faux pour un shell inconnu.
func (o Options) fichierShell() (string, string, bool) {
	d := o.dossierCommandes()
	dossier := `"` + d + `"`
	if rel, err := filepath.Rel(o.Home, d); err == nil && !strings.HasPrefix(rel, "..") {
		dossier = `"$HOME/` + filepath.ToSlash(rel) + `"`
	}
	posix := "case \":$PATH:\" in\n  *:" + dossier + ":*) ;;\n  *) export PATH=" + dossier + ":\"$PATH\" ;;\nesac"
	switch o.shell() {
	case "zsh":
		z := o.env("ZDOTDIR")
		if z == "" {
			z = o.Home
		}
		return filepath.Join(z, ".zshrc"), Bloc(posix), true
	case "bash":
		if o.OS != "darwin" {
			return filepath.Join(o.Home, ".bashrc"), Bloc(posix), true
		}
		// macOS : Terminal ouvre des shells de connexion, qui lisent le premier de ces fichiers (en créer
		// un masquerait les suivants)
		for _, f := range []string{".bash_profile", ".bash_login", ".profile"} {
			if _, err := os.Stat(filepath.Join(o.Home, f)); err == nil {
				return filepath.Join(o.Home, f), Bloc(posix), true
			}
		}
		return filepath.Join(o.Home, ".bash_profile"), Bloc(posix), true
	case "fish":
		c := o.env("XDG_CONFIG_HOME")
		if c == "" {
			c = filepath.Join(o.Home, ".config")
		}
		return filepath.Join(c, "fish", "config.fish"), Bloc("contains -- " + dossier + " $PATH\nor set -gx PATH " + dossier + " $PATH"), true
	case "sh", "dash", "ksh", "mksh":
		return filepath.Join(o.Home, ".profile"), Bloc(posix), true
	}
	return "", "", false
}

// fichiersShell : tous les fichiers où « agentline install » a pu poser son bloc (le shell a pu changer depuis).
func (o Options) fichiersShell() []string {
	c := o.env("XDG_CONFIG_HOME")
	if c == "" {
		c = filepath.Join(o.Home, ".config")
	}
	f := []string{".bashrc", ".bash_profile", ".bash_login", ".profile", ".zshrc"}
	for i := range f {
		f[i] = filepath.Join(o.Home, f[i])
	}
	if z := o.env("ZDOTDIR"); z != "" {
		f = append(f, filepath.Join(z, ".zshrc"))
	}
	return append(f, filepath.Join(c, "fish", "config.fish"))
}

// poserPathShell : le bloc PATH dans le fichier du shell, sauf si le dossier est déjà dans le PATH par
// ailleurs ; copie .bak avant toute modification.
func (o Options) poserPathShell() error {
	d := o.dossierCommandes()
	f, bloc, ok := o.fichierShell()
	if !ok {
		if !o.dansPath(d) {
			o.etape(i18n.T("À faire"), i18n.T("ajoute %s à ton PATH (shell %s : pas de fichier connu)", d, o.shell()))
		}
		return nil
	}
	if actuel, _ := os.ReadFile(f); !ABloc(string(actuel)) && o.dansPath(d) {
		o.etape(i18n.T("À jour"), i18n.T("PATH : %s", d))
		return nil
	}
	res, copie, err := o.modifier(f, func(c string) (string, string) { return PoserBloc(c, bloc) }, nil)
	if err != nil {
		return err
	}
	if res == identique {
		o.etape(i18n.T("À jour"), f)
		return nil
	}
	o.etape("PATH", i18n.T("%s ajouté au PATH dans %s%s : ouvre un nouveau terminal (ou : source %s)", d, f, avecCopie(copie), f))
	return nil
}

// retirerPathShell : retire le bloc de tous les fichiers du shell où il est.
func (o Options) retirerPathShell() error {
	for _, f := range o.fichiersShell() {
		res, copie, err := o.modifier(f, func(c string) (string, string) {
			n, ok := RetirerBloc(c)
			if !ok {
				return c, "identique"
			}
			return n, "retiré"
		}, nil)
		if err != nil {
			return err
		}
		if res != "identique" {
			o.etape("PATH", i18n.T("bloc retiré de %s%s", f, avecCopie(copie)))
		}
	}
	return nil
}

// marquePath : le fichier qui retient que « agentline install » a ajouté le dossier au PATH utilisateur
// (Windows) ; sans lui, « agentline uninstall » n'y touche pas (le dossier y était peut-être avant).
func (o Options) marquePath() string { return filepath.Join(o.dossierCommandes(), ".agentline-path") }

// reprendreMarquePath : la marque de « sl install » (.sl-path), renommée : c'est sl qui avait ajouté le
// dossier, agentline peut l'en retirer.
func (o Options) reprendreMarquePath() {
	ancienne := filepath.Join(o.dossierCommandes(), ".sl-path")
	if _, err := os.Stat(o.marquePath()); err != nil {
		os.Rename(ancienne, o.marquePath())
	}
}

// poserPathWindows : le dossier des commandes dans le PATH de l'utilisateur (HKCU\Environment\Path),
// sans droits d'administrateur ; le type de la valeur (REG_EXPAND_SZ le plus souvent) est gardé.
func (o Options) poserPathWindows() error {
	o.reprendreMarquePath()
	d := o.dossierCommandes()
	if o.Registre == nil {
		if !o.dansPath(d) {
			o.etape(i18n.T("À faire"), i18n.T("ajoute %s à ton PATH (Paramètres > Système > Informations système > Paramètres avancés > Variables d'environnement)", d))
		}
		return nil
	}
	v, existe, err := o.Registre.Lire(cleEnvironnement, "Path")
	if err != nil {
		return fmt.Errorf(i18n.T("lecture du PATH utilisateur : %w"), err)
	}
	if !existe {
		v.Etendu = true // le type que Windows lui donne
	}
	nouveau, change := o.AjouterAuPath(v.Texte, d)
	if !change {
		o.etape(i18n.T("À jour"), i18n.T("PATH utilisateur : %s", d))
		return nil
	}
	copie := ""
	if existe {
		copie = CopieNom(filepath.Join(d, "path-utilisateur"), o.Horodatage)
		os.MkdirAll(d, 0o755)
		if err := os.WriteFile(copie, []byte(v.Texte+"\n"), 0o644); err != nil {
			return err
		}
	}
	if err := o.Registre.Ecrire(cleEnvironnement, "Path", ValeurRegistre{Texte: nouveau, Etendu: v.Etendu}); err != nil {
		return fmt.Errorf(i18n.T("écriture du PATH utilisateur : %w"), err)
	}
	os.MkdirAll(d, 0o755)
	if err := os.WriteFile(o.marquePath(), []byte(d+"\n"), 0o644); err != nil {
		return err
	}
	suite := i18n.T("ouvre un nouveau terminal")
	if o.Diffuser == nil || !o.Diffuser(cleEnvironnement) {
		suite = i18n.T("ferme et rouvre ton terminal")
	}
	o.etape("PATH", i18n.T("%s ajouté au PATH utilisateur (HKCU\\Environment)%s : %s", d, avecCopie(copie), suite))
	return nil
}

// retirerPathWindows : défait poserPathWindows, seulement si c'est lui qui a ajouté le dossier.
func (o Options) retirerPathWindows() error {
	o.reprendreMarquePath()
	b, err := os.ReadFile(o.marquePath())
	if err != nil || o.Registre == nil {
		return nil
	}
	d := strings.TrimSpace(string(b))
	v, existe, err := o.Registre.Lire(cleEnvironnement, "Path")
	if err != nil {
		return fmt.Errorf(i18n.T("lecture du PATH utilisateur : %w"), err)
	}
	if existe {
		if nouveau, change := o.RetirerDuPath(v.Texte, d); change {
			copie := CopieNom(filepath.Join(o.dossierCommandes(), "path-utilisateur"), o.Horodatage)
			if err := os.WriteFile(copie, []byte(v.Texte+"\n"), 0o644); err != nil {
				return err
			}
			if err := o.Registre.Ecrire(cleEnvironnement, "Path", ValeurRegistre{Texte: nouveau, Etendu: v.Etendu}); err != nil {
				return fmt.Errorf(i18n.T("écriture du PATH utilisateur : %w"), err)
			}
			if o.Diffuser != nil {
				o.Diffuser(cleEnvironnement)
			}
			o.etape("PATH", i18n.T("%s retiré du PATH utilisateur%s", d, avecCopie(copie)))
		}
	}
	return os.Remove(o.marquePath())
}
