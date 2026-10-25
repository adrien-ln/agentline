package installation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// --- agentline doctor : l'installation vérifiée, chaque problème avec sa correction ---

// Niveau : ✓ en ordre, ⚠ à regarder (sans effet sur le code de sortie), ✗ à corriger.
type Niveau int

const (
	Bon Niveau = iota
	Avertissement
	Probleme
)

// Verification : un point vérifié, et la commande exacte qui le corrige.
type Verification struct {
	Niveau     Niveau
	Sujet      string
	Detail     string
	Correction string
}

// Diagnostiquer : tous les points de l'installation (sans rien modifier). o.Source : le binaire qui
// pose la question, comparé à celui qui est installé.
func Diagnostiquer(o Options) []Verification {
	v := []Verification{o.verifierBinaire(), o.verifierPath(), o.verifierStatusLine()}
	v = append(v, o.verifierOutils()...)
	v = append(v, o.verifierAnciens()...)
	v = append(v, o.verifierPolice()...)
	return append(v, o.verifierSSH())
}

// binaireInstalle : là où « agentline install » l'a mis, ou celui que lance statusLine (Scoop, Homebrew),
// sauf si c'est encore l'ancien sl.
func (o Options) binaireInstalle() string {
	if _, err := os.Stat(o.Binaire()); err == nil {
		return o.Binaire()
	}
	if c, ok := o.commandeStatusLine(); ok && !lanceAncien(c) {
		if b := strings.TrimSuffix(strings.TrimSpace(c), " render"); b != c {
			b = filepath.FromSlash(strings.Trim(b, `"`))
			if _, err := os.Stat(b); err == nil {
				return b
			}
		}
	}
	return o.Binaire()
}

func (o Options) verifierBinaire() Verification {
	b := o.binaireInstalle()
	installe, err := os.ReadFile(b)
	if err != nil {
		return Verification{Probleme, i18n.T("Binaire"), i18n.T("%s absent", b), "agentline install"}
	}
	if filepath.Clean(b) == filepath.Clean(o.Source) {
		return Verification{Niveau: Bon, Sujet: i18n.T("Binaire"), Detail: b}
	}
	if courant, err := os.ReadFile(o.Source); err == nil && !bytes.Equal(courant, installe) {
		return Verification{Avertissement, i18n.T("Binaire"), i18n.T("%s diffère de celui-ci (%s)", b, o.Source),
			i18n.T("si celui-ci est le plus récent : %s", `"`+o.Source+`" install`)}
	}
	return Verification{Niveau: Bon, Sujet: i18n.T("Binaire"), Detail: i18n.T("%s (à jour)", b)}
}

func (o Options) verifierPath() Verification {
	d := o.dossierCommandes()
	if o.dansPath(d) {
		return Verification{Niveau: Bon, Sujet: "PATH", Detail: d}
	}
	if o.windows() {
		if o.Registre != nil {
			if v, ok, _ := o.Registre.Lire(cleEnvironnement, "Path"); ok && o.dansListe(v.Texte, d) {
				return Verification{Avertissement, "PATH", i18n.T("%s est dans le PATH utilisateur, pas encore dans ce terminal", d), i18n.T("ouvre un nouveau terminal")}
			}
		}
		return Verification{Probleme, "PATH", i18n.T("%s n'est pas dans le PATH", d), "agentline install"}
	}
	if f, _, ok := o.fichierShell(); ok {
		if c, err := os.ReadFile(f); err == nil && ABloc(string(c)) {
			return Verification{Avertissement, "PATH", i18n.T("%s est ajouté par %s, pas encore dans ce terminal", d, f), i18n.T("source %s (ou ouvre un nouveau terminal)", f)}
		}
	}
	return Verification{Probleme, "PATH", i18n.T("%s n'est pas dans le PATH", d), "agentline install"}
}

// commandeStatusLine : la commande de statusLine dans settings.json ; faux si elle manque.
func (o Options) commandeStatusLine() (string, bool) {
	b, err := os.ReadFile(filepath.Join(o.Claude, "settings.json"))
	if err != nil {
		return "", false
	}
	var r struct {
		StatusLine *struct {
			Command string `json:"command"`
		} `json:"statusLine"`
	}
	if json.Unmarshal(b, &r) != nil || r.StatusLine == nil {
		return "", false
	}
	return r.StatusLine.Command, true
}

func (o Options) verifierStatusLine() Verification {
	f := filepath.Join(o.Claude, "settings.json")
	b, err := os.ReadFile(f)
	if err != nil {
		return Verification{Probleme, "Claude Code", i18n.T("%s absent", f), "agentline install"}
	}
	if !json.Valid(b) {
		return Verification{Probleme, "Claude Code", i18n.T("%s n'est pas un JSON valide", f), i18n.T("corrige %s, puis : agentline install", f)}
	}
	c, ok := o.commandeStatusLine()
	attendu := `"` + filepath.ToSlash(o.binaireInstalle()) + `" render`
	switch {
	case !ok:
		return Verification{Probleme, "Claude Code", i18n.T("pas de statusLine dans %s", f), "agentline install"}
	case lanceAncien(c):
		return Verification{Probleme, "Claude Code", i18n.T("statusLine lance encore l'ancien sl : « %s »", c), "agentline install"}
	case c != attendu:
		return Verification{Probleme, "Claude Code", i18n.T("statusLine lance « %s », pas %s", c, attendu), "agentline install"}
	}
	return Verification{Niveau: Bon, Sujet: "Claude Code", Detail: i18n.T("statusLine : %s", c)}
}

// verifierPolice : une Nerd Font est-elle installée ; puis, pour chaque terminal dont agentline lit les
// réglages (Windows Terminal, VS Code), la police qui y est vraiment choisie.
func (o Options) verifierPolice() []Verification {
	if o.env("SSH_CONNECTION") != "" {
		return []Verification{{Niveau: Bon, Sujet: i18n.T("Police"), Detail: i18n.T("en SSH, c'est celle du terminal de ta machine qui compte")}}
	}
	if !o.policeNerdFont() {
		return []Verification{{Avertissement, i18n.T("Police"), i18n.T("aucune Nerd Font : les icônes s'afficheront mal"), o.CommandePolice()}}
	}
	v := []Verification{{Niveau: Bon, Sujet: i18n.T("Police"), Detail: i18n.T("une Nerd Font est installée (choisie dans ton terminal ?)")}}
	if p := o.police(); o.PoliceInstallee(p) {
		v[0].Detail = i18n.T("%s est installée", p.Famille)
	}
	if f := o.ApercuInterrompu(); f != "" {
		v = append(v, Verification{Avertissement, i18n.T("Police"), i18n.T("un aperçu de police a été interrompu : %s garde la police d'essai", f), "agentline font preview"})
	}
	for _, r := range o.PolicesReglees() {
		switch {
		case strings.Contains(strings.ToLower(r.Police), "nerd font"):
			v = append(v, Verification{Niveau: Bon, Sujet: r.Terminal, Detail: i18n.T("police %s", r.Police)})
		case r.Police == "":
			v = append(v, Verification{Avertissement, r.Terminal, i18n.T("aucune police choisie : pas de Nerd Font, les icônes s'afficheront mal"), "agentline font set " + o.police().Nom})
		default:
			v = append(v, Verification{Avertissement, r.Terminal, i18n.T("police « %s », pas une Nerd Font : les icônes s'afficheront mal", r.Police), "agentline font set " + o.police().Nom})
		}
	}
	return v
}

func (o Options) verifierSSH() Verification {
	if o.env("SSH_CONNECTION") == "" {
		return Verification{Niveau: Bon, Sujet: "SSH", Detail: i18n.T("SSH_CONNECTION absent : statusline complète")}
	}
	if !o.Court {
		return Verification{Niveau: Bon, Sujet: "SSH", Detail: i18n.T("en SSH : statusline complète (ligne courte : agentline style set ssh.short yes)")}
	}
	return Verification{Avertissement, "SSH", i18n.T("en SSH : statusline sur une ligne courte, sans icônes (ssh.short=yes)"),
		i18n.T("AGENTLINE_SSH_RICH=1 : les couleurs du thème quand même ; la statusline complète : agentline style set ssh.short no")}
}

// Afficher : une ligne par vérification, la correction en dessous ; renvoie le code de sortie (1 s'il
// y a un ✗).
func Afficher(w io.Writer, v []Verification) int {
	code := 0
	largeur := 0
	for _, x := range v {
		largeur = max(largeur, len([]rune(x.Sujet)))
	}
	for _, x := range v {
		marque := map[Niveau]string{Bon: "\x1b[32m✓\x1b[0m", Avertissement: "\x1b[33m⚠\x1b[0m", Probleme: "\x1b[31m✗\x1b[0m"}[x.Niveau]
		fmt.Fprintf(w, "%s %s%s  %s\n", marque, x.Sujet, strings.Repeat(" ", largeur-len([]rune(x.Sujet))), x.Detail)
		if x.Niveau != Bon && x.Correction != "" {
			fmt.Fprintf(w, "  %s  → %s\n", strings.Repeat(" ", largeur), x.Correction)
		}
		if x.Niveau == Probleme {
			code = 1
		}
	}
	return code
}
