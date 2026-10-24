package installation

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// --- Les restes de « sl », le nom d'avant agentline ---
//
// Jusqu'au renommage, « sl install » posait le binaire sl (sl.exe), les commandes sl-theme (et
// sl-theme.cmd sous Windows) et une fonction sl-theme dans un bloc balisé du profil PowerShell. Plus rien
// de tout ça n'est utile : « agentline » seul ouvre le sélecteur. « agentline install » et « agentline
// uninstall » les retirent, seulement ce qui est reconnu comme venant de sl : le binaire et les commandes
// sont renommés en copie .bak datée (pas effacés), le bloc est retiré du profil (copie .bak avant).

const (
	marqueCommande = "# commande sl-theme de « sl install »" // dans le sl-theme de Git Bash, Linux et macOS
	marqueCmd      = "rem commande sl-theme de sl install"   // dans sl-theme.cmd
	moduleAncien   = "adrien-ln/statusline"                  // le module de sl, inscrit dans son binaire
)

// reste : un fichier laissé par sl, et ce que c'est.
type reste struct{ chemin, quoi string }

// dossiersAnciens : où sl a pu poser ses fichiers (le dossier des commandes, celui du binaire).
func (o Options) dossiersAnciens() []string {
	d := []string{o.dossierCommandes()}
	if b := filepath.Dir(o.Binaire()); filepath.Clean(b) != filepath.Clean(d[0]) {
		d = append(d, b)
	}
	return d
}

func contient(f string, marques ...string) bool {
	b, err := os.ReadFile(f)
	if err != nil {
		return false
	}
	for _, m := range marques {
		if bytes.Contains(b, []byte(m)) {
			return true
		}
	}
	return false
}

// restes : les fichiers de sl encore là (ni le binaire courant, ni celui qui s'installe).
func (o Options) restes() []reste {
	var r []reste
	for _, d := range o.dossiersAnciens() {
		for _, n := range []string{"sl", "sl.exe", "sl.exe.ancien"} {
			f := filepath.Join(d, n)
			if filepath.Clean(f) != filepath.Clean(o.Binaire()) && filepath.Clean(f) != filepath.Clean(o.Source) && contient(f, moduleAncien) {
				r = append(r, reste{f, i18n.T("ancien binaire")})
			}
		}
		for _, n := range []string{"sl-theme", "sl-theme.cmd"} {
			if f := filepath.Join(d, n); contient(f, marqueCommande, marqueCmd, "statusline") {
				r = append(r, reste{f, i18n.T("commande %s", n)})
			}
		}
	}
	return r
}

// profilsAvecBloc : les profils PowerShell qui contiennent encore le bloc de sl.
func (o Options) profilsAvecBloc() []string {
	var p []string
	if !o.windows() {
		return p
	}
	for _, f := range o.Profils {
		if b, err := os.ReadFile(f); err == nil && ABloc(string(b)) {
			p = append(p, f)
		}
	}
	return p
}

// retirerAnciens : les restes de sl renommés en .bak, le bloc retiré des profils PowerShell.
func (o Options) retirerAnciens() error {
	for _, x := range o.restes() {
		copie := CopieNom(x.chemin, o.Horodatage)
		if err := os.Rename(x.chemin, copie); err != nil {
			return err
		}
		o.etape(i18n.T("Ancien sl"), i18n.T("%s mis de côté : %s", x.quoi, copie))
	}
	for _, p := range o.profilsAvecBloc() {
		_, copie, err := o.modifier(p, func(c string) (string, string) {
			n, ok := RetirerBloc(c)
			if !ok {
				return c, "identique"
			}
			return n, "retiré"
		}, nil)
		if err != nil {
			return err
		}
		o.etape(i18n.T("Ancien sl"), i18n.T("fonction sl-theme retirée de %s%s", p, avecCopie(copie)))
	}
	return nil
}

// verifierAnciens : un avertissement par reste de sl, corrigé par « agentline install ».
func (o Options) verifierAnciens() []Verification {
	var v []Verification
	for _, x := range o.restes() {
		v = append(v, Verification{Avertissement, i18n.T("Ancien sl"), i18n.T("%s encore là : %s", x.quoi, x.chemin), "agentline install"})
	}
	for _, p := range o.profilsAvecBloc() {
		v = append(v, Verification{Avertissement, i18n.T("Ancien sl"), i18n.T("fonction sl-theme dans le profil PowerShell %s", p), "agentline install"})
	}
	return v
}

// lanceAncien : vrai si la commande de statusLine lance le binaire sl (« "…/sl.exe" render »).
func lanceAncien(commande string) bool {
	b := strings.TrimSuffix(strings.TrimSpace(commande), " render")
	b = strings.Trim(b, `"`)
	n := strings.ToLower(filepath.Base(filepath.FromSlash(b)))
	return n == "sl" || n == "sl.exe"
}
