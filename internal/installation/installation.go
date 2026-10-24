package installation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// identique : le résultat d'une modification qui ne change rien. Les autres résultats (créé, ajouté,
// remplacé, retiré) sont affichés, traduits par i18n.T.
const identique = "identique"

var _ = []string{i18n.M("créé"), i18n.M("ajouté"), i18n.M("remplacé"), i18n.M("retiré")}

// Options : où installer, et quoi toucher.
type Options struct {
	Home       string    // dossier personnel
	Claude     string    // ~/.claude (ou CLAUDE_CONFIG_DIR)
	Source     string    // le binaire qui s'installe (os.Executable)
	SurPlace   bool      // ne pas copier le binaire (installé par Scoop ou Homebrew, qui gèrent sa mise à jour)
	Profils    []string  // profils PowerShell (Windows), où chercher le bloc qu'y posait « sl install »
	OS         string    // "windows", "darwin", "linux"… (runtime.GOOS ; les tests simulent les autres)
	Horodatage string    // pour les copies .bak (AAAAMMJJ-HHMMSS)
	Sortie     io.Writer // le compte rendu
	Court      bool      // la statusline passe en ligne courte ici (SSH_CONNECTION défini et ssh.short=yes)

	// Ce qui touche au système, injecté (les tests n'ont aucun effet réel) ; nil : rien n'est lu ni lancé.
	Getenv         func(string) string                              // l'environnement
	Lancer         func(nom string, args ...string) (string, error) // lance une commande, renvoie sa sortie
	Registre       Registre                                         // HKCU (Windows)
	Diffuser       func(quoi string) bool                           // WM_SETTINGCHANGE / WM_FONTCHANGE (Windows)
	Chercher       func(programme string) (string, error)           // exec.LookPath
	Telecharger    func(url string) ([]byte, error)
	PolicesSysteme []string // dossiers de polices de tous les utilisateurs (PolicesSystemeDe)

	// La police : proposée seulement si Interactif, installée seulement si Demander répond oui.
	Interactif bool
	Demander   func(question string) bool
	SansPolice bool // --no-font : ne rien proposer, afficher la commande
}

func (o Options) windows() bool { return o.OS == "windows" }

func (o Options) env(cle string) string {
	if o.Getenv == nil {
		return ""
	}
	return o.Getenv(cle)
}

func (o Options) lancer(nom string, args ...string) (string, error) {
	if o.Lancer == nil {
		return "", fmt.Errorf(i18n.T("%s : aucune commande ne peut être lancée"), nom)
	}
	return o.Lancer(nom, args...)
}

// Etape : une ligne du compte rendu, titre en gras (pour ce que la CLI ajoute à l'installation).
func (o Options) Etape(titre, texte string) { o.etape(titre, texte) }

func (o Options) etape(titre, texte string) {
	fmt.Fprintf(o.Sortie, "\x1b[1m%s\x1b[0m %s\n", titre, texte)
}

// Binaire : où le binaire est installé (dans le PATH de Git Bash, ou ~/.local/bin).
func (o Options) Binaire() string {
	switch {
	case o.SurPlace:
		return o.Source
	case o.windows():
		return filepath.Join(o.Home, "bin", "agentline.exe")
	default:
		return filepath.Join(o.Home, ".local", "bin", "agentline")
	}
}

func (o Options) dossierCommandes() string {
	if o.windows() {
		return filepath.Join(o.Home, "bin")
	}
	return filepath.Join(o.Home, ".local", "bin")
}

// Commande : ce que Claude Code lance (barres obliques et guillemets : compris par bash comme par cmd).
func (o Options) Commande() string { return `"` + filepath.ToSlash(o.Binaire()) + `" render` }

// modifier : réécrit un fichier de configuration par une fonction ; copie .bak datée avant, seulement
// s'il change ; valide est appelé sur le nouveau contenu avant de l'écrire.
func (o Options) modifier(f string, transformer func(string) (string, string), valide func(string) error) (string, string, error) {
	ancien, err := os.ReadFile(f)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	nouveau, resultat := transformer(string(ancien))
	if resultat == identique || nouveau == string(ancien) {
		return identique, "", nil
	}
	if valide != nil {
		if err := valide(nouveau); err != nil {
			return "", "", fmt.Errorf(i18n.T("%s deviendrait invalide (%v) : rien n'a été changé"), f, err)
		}
	}
	copie := ""
	if err == nil {
		copie = CopieNom(f, o.Horodatage)
		if err := os.WriteFile(copie, ancien, 0o644); err != nil {
			return "", "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return "", "", err
	}
	return resultat, copie, os.WriteFile(f, []byte(nouveau), 0o644) // en place : le fichier garde ses permissions
}

func jsonValide(s string) error {
	if !json.Valid([]byte(s)) {
		return errors.New(i18n.T("JSON invalide"))
	}
	return nil
}

func avecCopie(copie string) string {
	if copie == "" {
		return ""
	}
	return i18n.T(" (copie : %s)", copie)
}

// copierBinaire : le binaire à sa place ; sous Windows, un .exe en cours d'exécution ne peut pas être
// remplacé, mais peut être renommé : l'ancien est mis de côté.
func (o Options) copierBinaire() (bool, error) {
	dest := o.Binaire()
	src, err := os.ReadFile(o.Source)
	if err != nil {
		return false, err
	}
	if actuel, err := os.ReadFile(dest); err == nil && bytes.Equal(actuel, src) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	tmp := dest + ".nouveau"
	if err := os.WriteFile(tmp, src, 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		ancien := dest + ".ancien"
		os.Remove(ancien)
		if err2 := os.Rename(dest, ancien); err2 != nil {
			os.Remove(tmp)
			return false, err
		}
		if err := os.Rename(tmp, dest); err != nil {
			return false, err
		}
	}
	return true, nil
}

// Installer : installe ou met à jour ; ne change que ce qui diffère. Les restes de « sl » (le nom
// d'avant) sont retirés à la fin, quand la statusline lance déjà le nouveau binaire.
func Installer(o Options) error {
	if !o.SurPlace && filepath.Clean(o.Source) != filepath.Clean(o.Binaire()) {
		change, err := o.copierBinaire()
		if err != nil {
			return fmt.Errorf(i18n.T("copie du binaire : %w"), err)
		}
		if change {
			o.etape(i18n.T("Installé"), o.Binaire())
		} else {
			o.etape(i18n.T("À jour"), o.Binaire())
		}
	}

	reglages := filepath.Join(o.Claude, "settings.json")
	res, copie, err := o.modifier(reglages, func(c string) (string, string) { return PoserStatusLine(c, BlocStatusLine(o.Commande())) }, jsonValide)
	if err != nil {
		return err
	}
	if res == identique {
		o.etape(i18n.T("À jour"), reglages)
	} else {
		o.etape(i18n.T("Réglages"), i18n.T("statusLine %s dans %s%s : %s", i18n.T(res), reglages, avecCopie(copie), o.Commande()))
	}

	poserPath := o.poserPathShell
	if o.windows() {
		poserPath = o.poserPathWindows
	}
	if err := poserPath(); err != nil {
		return err
	}
	if err := o.retirerAnciens(); err != nil {
		return err
	}

	o.proposerPolice()
	return nil
}

// Desinstaller : retire statusLine, ce que l'installation a ajouté au PATH et les restes de « sl »
// (copies .bak avant) ; le binaire est renommé, pas effacé ; les réglages de l'utilisateur sont gardés.
func Desinstaller(o Options) error {
	reglages := filepath.Join(o.Claude, "settings.json")
	if res, copie, err := o.modifier(reglages, func(c string) (string, string) {
		n, ok := RetirerStatusLine(c)
		if !ok {
			return c, identique
		}
		return n, "retiré"
	}, jsonValide); err != nil {
		return err
	} else if res != identique {
		o.etape(i18n.T("Réglages"), i18n.T("statusLine retirée de %s%s", reglages, avecCopie(copie)))
	}
	if err := o.retirerAnciens(); err != nil {
		return err
	}
	retirerPath := o.retirerPathShell
	if o.windows() {
		retirerPath = o.retirerPathWindows
	}
	if err := retirerPath(); err != nil {
		return err
	}
	if !o.SurPlace {
		if _, err := os.Stat(o.Binaire()); err == nil {
			copie := CopieNom(o.Binaire(), o.Horodatage)
			if err := os.Rename(o.Binaire(), copie); err != nil {
				return err
			}
			o.etape(i18n.T("Binaire"), i18n.T("renommé : %s", copie))
		}
	}
	o.etape(i18n.T("Désinstallé"), i18n.T("relance Claude Code pour qu'il oublie la statusline"))
	return nil
}
