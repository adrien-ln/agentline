package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/adrien-ln/agentline/internal/i18n"
	"github.com/adrien-ln/agentline/internal/installation"
)

// terminal : vrai si f est une console (ni tube ni fichier).
func terminal(f *os.File) bool {
	i, err := f.Stat()
	return err == nil && i.Mode()&os.ModeCharDevice != 0 && console(f) // NUL et /dev/null sont aussi des périphériques caractère
}

// telecharger : le contenu d'une adresse (200 Mo au plus, deux minutes au plus).
func telecharger(url string) ([]byte, error) {
	c := &http.Client{Timeout: 2 * time.Minute}
	r, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s : %s", url, r.Status)
	}
	return io.ReadAll(io.LimitReader(r.Body, 200<<20))
}

// demander : pose la question, lit la réponse ; vide, « o », « oui », « y » ou « yes » : oui.
func demander(sortie io.Writer) func(string) bool {
	return func(q string) bool {
		fmt.Fprint(sortie, q)
		l, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(l)) {
		case "", "o", "oui", "y", "yes":
			return true
		}
		return false
	}
}

// version : injectée à la compilation (-ldflags "-X main.version=…") ; sinon lue dans les informations
// de construction (version du module, ou commit Git que Go y inscrit).
var version = "dev"

func versionAffichee() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	rev, modifie := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modifie = s.Value == "true"
		}
	}
	if len(rev) >= 7 {
		v := "dev-" + rev[:7]
		if modifie {
			v += "-modifié"
		}
		return v
	}
	return version
}

// profilsPowerShell : les profils de PowerShell 7 et de Windows PowerShell, où « sl install » posait sa
// fonction sl-theme (pour l'en retirer) ; AGENTLINE_PS_PROFILE pour imposer le profil.
func profilsPowerShell() []string {
	if p := getenv("AGENTLINE_PS_PROFILE"); p != "" {
		return []string{p}
	}
	var profils []string
	for _, ps := range []string{"pwsh.exe", "powershell.exe"} {
		if sortie, err := exec.Command(ps, "-NoProfile", "-Command", "$PROFILE.CurrentUserCurrentHost").Output(); err == nil {
			if p := strings.TrimSpace(string(sortie)); p != "" && !slices.Contains(profils, p) {
				profils = append(profils, p)
			}
		}
	}
	return profils
}

// lancerCommande : la commande et sa sortie (erreurs comprises), pour l'installation.
func lancerCommande(nom string, args ...string) (string, error) {
	sortie, err := exec.Command(nom, args...).CombinedOutput()
	return string(sortie), err
}

// optionsReelles : l'installation de cette machine, avec le vrai système (environnement, commandes).
func optionsReelles(a *appel) installation.Options {
	o := installation.Options{OS: runtime.GOOS, Horodatage: time.Now().Format("20060102-150405"), Sortie: a.sortie,
		Getenv: getenv, Lancer: lancerCommande, Registre: installation.RegistreReel(), Diffuser: installation.DiffuserReel,
		Chercher: exec.LookPath, Telecharger: telecharger, PolicesSysteme: installation.PolicesSystemeDe(runtime.GOOS, getenv),
		Interactif: terminal(os.Stdin) && terminal(os.Stdout), Demander: demander(a.sortie), SansPolice: a.opts["--no-font"]}
	o.SurPlace = a.opts["--in-place"]
	o.Court = courtIci()
	o.Home, _ = os.UserHomeDir()
	o.Claude = getenv("CLAUDE_CONFIG_DIR")
	if o.Claude == "" {
		o.Claude = filepath.Join(o.Home, ".claude")
	}
	src, err := os.Executable()
	if err == nil {
		if r, err := filepath.EvalSymlinks(src); err == nil {
			src = r
		}
	}
	o.Source = src
	if o.OS == "windows" {
		o.Profils = profilsPowerShell()
	}
	return o
}

// commandeInstallation : « agentline install » et « agentline uninstall » ; renvoie le code de sortie.
func commandeInstallation(retirer bool, a *appel) int {
	o := optionsReelles(a)
	f := installation.Installer
	if retirer {
		f = installation.Desinstaller
	} else if m, ok := verifierMigration(); ok {
		o.Etape(i18n.T("Réglages"), m.Detail)
	}
	if err := f(o); err != nil {
		return a.echec("%v", err)
	}
	if !retirer {
		if o.Interactif {
			signalerFond(sources(), a.sortie)
		}
		fmt.Fprintln(a.sortie, i18n.T("\nTerminé. Choisis un thème avec : agentline (dans un nouveau terminal). Vérifie avec : agentline doctor"))
	}
	return 0
}
