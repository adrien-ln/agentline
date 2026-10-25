package installation

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// --- La police : une Nerd Font, pour les icônes de la statusline ---

const (
	clePolices   = `Software\Microsoft\Windows NT\CurrentVersion\Fonts` // polices de l'utilisateur (HKCU)
	tailleMaxZip = 200 << 20
	// archivesPolices : les archives officielles de la dernière version de Nerd Fonts.
	archivesPolices = "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/"
)

// Police : une Nerd Font proposée, avec ce qu'il faut pour l'installer et la choisir dans un terminal.
type Police struct {
	Nom      string // le nom court, pour la ligne de commande
	Famille  string // le nom de famille, celui que les terminaux attendent
	Archive  string // l'archive officielle de Nerd Fonts
	Fichiers string // le début du nom des .ttf de la famille principale (ni Mono ni Propo)
	Scoop    string // Scoop, dépôt « nerd-fonts »
	Cask     string // Homebrew (dépôt principal des casks depuis 2024)
}

// Polices : les Nerd Fonts proposées ; la première est celle par défaut.
var Polices = []Police{
	{Nom: "firacode", Famille: "FiraCode Nerd Font", Archive: archivesPolices + "FiraCode.zip", Fichiers: "FiraCodeNerdFont-",
		Scoop: "nerd-fonts/FiraCode-NF", Cask: "font-fira-code-nerd-font"},
}

// police : la Nerd Font à installer et à choisir dans les terminaux.
func (o Options) police() Police { return Polices[0] }

// PolicesSystemeDe : les dossiers de polices communs à tous les utilisateurs, selon l'OS.
func PolicesSystemeDe(goos string, getenv func(string) string) []string {
	switch goos {
	case "windows":
		w := getenv("WINDIR")
		if w == "" {
			w = `C:\Windows`
		}
		return []string{filepath.Join(w, "Fonts")}
	case "darwin":
		return []string{"/Library/Fonts"}
	}
	return []string{"/usr/share/fonts", "/usr/local/share/fonts"}
}

// dossiersPolices : où chercher une Nerd Font (ceux de l'utilisateur d'abord).
func (o Options) dossiersPolices() []string {
	var d []string
	switch o.OS {
	case "windows", "darwin":
		d = []string{o.dossierPolicesUtilisateur(o.police())}
	default:
		donnees := o.env("XDG_DATA_HOME")
		if donnees == "" {
			donnees = filepath.Join(o.Home, ".local", "share")
		}
		d = []string{filepath.Join(donnees, "fonts"), filepath.Join(o.Home, ".fonts")}
	}
	return append(d, o.PolicesSysteme...)
}

// dossierPolicesUtilisateur : où installer la police p sans droits d'administrateur (sous Linux, un
// dossier par police).
func (o Options) dossierPolicesUtilisateur(p Police) string {
	switch o.OS {
	case "windows":
		return filepath.Join(o.localAppData(), "Microsoft", "Windows", "Fonts")
	case "darwin":
		return filepath.Join(o.Home, "Library", "Fonts")
	}
	donnees := o.env("XDG_DATA_HOME")
	if donnees == "" {
		donnees = filepath.Join(o.Home, ".local", "share")
	}
	return filepath.Join(donnees, "fonts", strings.TrimSuffix(p.Fichiers, "-"))
}

func (o Options) localAppData() string {
	if l := o.env("LOCALAPPDATA"); l != "" {
		return l
	}
	return filepath.Join(o.Home, "AppData", "Local")
}

func estNerdFont(nom string) bool {
	n := strings.ToLower(nom)
	return strings.Contains(n, "nerd") && (strings.HasSuffix(n, ".ttf") || strings.HasSuffix(n, ".otf") || strings.HasSuffix(n, ".ttc"))
}

// policeNerdFont : une Nerd Font est-elle installée ? En SSH, la police est celle du client : on ne
// cherche pas.
func (o Options) policeNerdFont() bool {
	if o.env("SSH_CONNECTION") != "" {
		return true
	}
	for _, d := range o.dossiersPolices() {
		trouve := false
		filepath.WalkDir(d, func(p string, e fs.DirEntry, err error) error {
			if err != nil || trouve {
				return filepath.SkipDir
			}
			if e.IsDir() && strings.Count(strings.TrimPrefix(p, d), string(filepath.Separator)) > 3 {
				return filepath.SkipDir // assez profond
			}
			if !e.IsDir() && estNerdFont(e.Name()) {
				trouve = true
				return filepath.SkipAll
			}
			return nil
		})
		if trouve {
			return true
		}
	}
	if !o.windows() {
		if s, err := o.lancer("fc-list", ":", "family"); err == nil && strings.Contains(strings.ToLower(s), "nerd font") {
			return true
		}
	}
	return false
}

// CommandePolice : ce que l'utilisateur peut lancer lui-même pour installer la police.
func (o Options) CommandePolice() string {
	p := o.police()
	switch o.OS {
	case "windows":
		if o.trouve("scoop") {
			return "scoop bucket add nerd-fonts ; scoop install " + p.Scoop
		}
		return i18n.T("agentline install (réponds O), ou télécharge %s et installe les .ttf (clic droit > Installer)", p.Archive)
	case "darwin":
		return "brew install --cask " + p.Cask
	}
	return i18n.T("agentline install (réponds O), ou télécharge %s dans ~/.local/share/fonts, puis fc-cache -f", p.Archive)
}

func (o Options) trouve(programme string) bool {
	if o.Chercher == nil {
		return false
	}
	_, err := o.Chercher(programme)
	return err == nil
}

// proposerPolice : sans Nerd Font, propose de l'installer et de régler le terminal (jamais sans réponse
// de l'utilisateur : en mode non interactif ou avec --no-font, on affiche seulement la commande).
func (o Options) proposerPolice() {
	if o.policeNerdFont() {
		return
	}
	manque := i18n.T("aucune Nerd Font trouvée : les icônes s'afficheraient mal. Pour l'installer : %s", o.CommandePolice())
	if o.SansPolice || !o.Interactif || o.Demander == nil {
		o.etape(i18n.T("Police"), manque)
		return
	}
	if !o.Demander(i18n.T("Installer %s et régler ton terminal ? [O/n] ", o.police().Famille)) {
		o.etape(i18n.T("Police"), i18n.T("pas installée. Plus tard : %s", o.CommandePolice()))
		return
	}
	if err := o.installerPolice(); err != nil {
		o.etape(i18n.T("Attention"), i18n.T("police non installée (%s). À la main : %s", err.Error(), o.CommandePolice()))
		return
	}
	o.reglerTerminaux()
}

// installerPolice : par le gestionnaire de paquets s'il y en a un (Scoop, Homebrew), sinon par
// l'archive officielle, dans les polices de l'utilisateur.
func (o Options) installerPolice() error {
	p := o.police()
	switch {
	case o.windows() && o.trouve("scoop"):
		o.lancer("scoop", "bucket", "add", "nerd-fonts") // déjà ajouté : une erreur sans conséquence
		if s, err := o.lancer("scoop", "install", p.Scoop); err != nil {
			return fmt.Errorf("scoop install %s : %v %s", p.Scoop, err, dernieresLignes(s))
		}
		o.etape(i18n.T("Police"), i18n.T("%s installée par %s", p.Famille, "Scoop"))
		return nil
	case o.OS == "darwin" && o.trouve("brew"):
		if s, err := o.lancer("brew", "install", "--cask", p.Cask); err != nil {
			return fmt.Errorf("brew install --cask %s : %v %s", p.Cask, err, dernieresLignes(s))
		}
		o.etape(i18n.T("Police"), i18n.T("%s installée par %s", p.Famille, "Homebrew"))
		return nil
	}
	return o.installerArchivePolice()
}

func dernieresLignes(s string) string {
	l := strings.Split(strings.TrimSpace(s), "\n")
	if len(l) > 3 {
		l = l[len(l)-3:]
	}
	return strings.TrimSpace(strings.Join(l, " / "))
}

// installerArchivePolice : télécharge l'archive, en copie les .ttf de la famille principale (ni Mono ni
// Propo) dans les polices de l'utilisateur ; sous Windows, les déclare dans HKCU ; sous Linux, fc-cache.
func (o Options) installerArchivePolice() error {
	p := o.police()
	if o.Telecharger == nil {
		return errors.New(i18n.T("téléchargement impossible"))
	}
	o.etape(i18n.T("Police"), i18n.T("téléchargement de %s", p.Archive))
	b, err := o.Telecharger(p.Archive)
	if err != nil {
		return err
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return fmt.Errorf(i18n.T("archive illisible : %v"), err)
	}
	var choisis []*zip.File
	for _, f := range z.File {
		if n := filepath.Base(f.Name); strings.HasPrefix(n, p.Fichiers) && strings.HasSuffix(n, ".ttf") {
			choisis = append(choisis, f)
		}
	}
	if len(choisis) == 0 {
		return errors.New(i18n.T("aucune police %s*.ttf dans l'archive", p.Fichiers))
	}
	dossier := o.dossierPolicesUtilisateur(p)
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return err
	}
	for _, f := range choisis {
		r, err := f.Open()
		if err != nil {
			return err
		}
		var tampon bytes.Buffer
		_, err = tampon.ReadFrom(&limite{r: r, reste: tailleMaxZip})
		r.Close()
		if err != nil {
			return err
		}
		dest := filepath.Join(dossier, filepath.Base(f.Name)) // jamais de chemin venu de l'archive
		if err := os.WriteFile(dest, tampon.Bytes(), 0o644); err != nil {
			return err
		}
		if o.windows() && o.Registre != nil {
			nom := strings.Replace(strings.TrimSuffix(filepath.Base(f.Name), ".ttf"), "NerdFont-", " Nerd Font ", 1) + " (TrueType)"
			if err := o.Registre.Ecrire(clePolices, nom, ValeurRegistre{Texte: dest}); err != nil {
				return fmt.Errorf(i18n.T("déclaration de la police : %v"), err)
			}
		}
	}
	switch {
	case o.windows():
		if o.Diffuser != nil {
			o.Diffuser("polices")
		}
	case o.OS != "darwin":
		if _, err := o.lancer("fc-cache", "-f"); err != nil {
			o.etape(i18n.T("Attention"), i18n.T("fc-cache -f a échoué : lance-le, ou ferme ta session"))
		}
	}
	o.etape(i18n.T("Police"), i18n.T("%s installée (%d fichiers) dans %s", p.Famille, len(choisis), dossier))
	return nil
}

// limite : un lecteur qui refuse de lire plus que reste (une archive piégée ne remplit pas le disque).
type limite struct {
	r     io.Reader
	reste int64
}

func (l *limite) Read(p []byte) (int, error) {
	if l.reste <= 0 {
		return 0, errors.New(i18n.T("fichier trop gros dans l'archive"))
	}
	if int64(len(p)) > l.reste {
		p = p[:l.reste]
	}
	n, err := l.r.Read(p)
	l.reste -= int64(n)
	return n, err
}

// --- Régler le terminal ---

// reglage : un fichier de réglages de terminal (JSON avec commentaires) et la clé de la police.
type reglage struct {
	terminal, fichier string
	chemin            []string
}

// reglagesTerminaux : ceux qui existent sur la machine (on n'en crée aucun).
func (o Options) reglagesTerminaux() []reglage {
	var r []reglage
	ajouter := func(terminal, f string, chemin ...string) {
		if _, err := os.Stat(f); err == nil {
			r = append(r, reglage{terminal, f, chemin})
		}
	}
	face := []string{"profiles", "defaults", "font", "face"}
	vscode := "terminal.integrated.fontFamily"
	switch o.OS {
	case "windows":
		l := o.localAppData()
		for _, p := range []string{"Microsoft.WindowsTerminal_8wekyb3d8bbwe", "Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe"} {
			ajouter("Windows Terminal", filepath.Join(l, "Packages", p, "LocalState", "settings.json"), face...)
		}
		ajouter("Windows Terminal", filepath.Join(l, "Microsoft", "Windows Terminal", "settings.json"), face...) // hors Microsoft Store
		appdata := o.env("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(o.Home, "AppData", "Roaming")
		}
		ajouter("VS Code", filepath.Join(appdata, "Code", "User", "settings.json"), vscode)
	case "darwin":
		ajouter("VS Code", filepath.Join(o.Home, "Library", "Application Support", "Code", "User", "settings.json"), vscode)
	default:
		c := o.env("XDG_CONFIG_HOME")
		if c == "" {
			c = filepath.Join(o.Home, ".config")
		}
		ajouter("VS Code", filepath.Join(c, "Code", "User", "settings.json"), vscode)
	}
	return r
}

// reglerTerminaux : la police dans Windows Terminal et VS Code (copie .bak avant) ; pour le terminal en
// cours, s'il n'est ni l'un ni l'autre, ce qu'il faut faire à la main.
func (o Options) reglerTerminaux() {
	famille := o.police().Famille
	regle := map[string]bool{}
	for _, r := range o.reglagesTerminaux() {
		res, copie, err := o.modifier(r.fichier, func(c string) (string, string) {
			if v, ok := ChaineJSONC(c, r.chemin...); ok && strings.Contains(strings.ToLower(v), "nerd font") {
				return c, "identique" // déjà une Nerd Font
			}
			n, res, err := PoserChaineJSONC(c, famille, r.chemin...)
			if err != nil {
				return c, "identique"
			}
			return n, res
		}, jsoncValide)
		switch {
		case err != nil:
			o.etape(i18n.T("Attention"), i18n.T("%s : %s", r.terminal, err.Error()))
		case res == identique:
			if c, _ := os.ReadFile(r.fichier); jsoncValide(string(c)) != nil {
				o.etape(i18n.T("Attention"), i18n.T("%s : %s illisible, choisis la police %s à la main", r.terminal, r.fichier, famille))
			} else if v, _ := ChaineJSONC(string(c), r.chemin...); !strings.Contains(strings.ToLower(v), "nerd font") {
				o.etape(i18n.T("Attention"), i18n.T("%s : %s introuvable dans %s, choisis la police %s à la main", r.terminal, strings.Join(r.chemin, "."), r.fichier, famille))
			} else {
				o.etape(i18n.T("À jour"), i18n.T("%s : %s", r.terminal, r.fichier))
			}
			regle[r.terminal] = true
		default:
			o.etape("Terminal", i18n.T("%s : police %s (%s %s) dans %s%s ; relance-le si la police n'apparaît pas",
				r.terminal, famille, strings.Join(r.chemin, "."), i18n.T(res), r.fichier, avecCopie(copie)))
			regle[r.terminal] = true
		}
		if r.terminal == "Windows Terminal" {
			o.profilsAvecPolice(r.fichier)
		}
	}
	if t, aide := o.terminalCourant(); t != "" && !regle[t] {
		o.etape("Terminal", aide)
	}
}

// profilsAvecPolice : les profils de Windows Terminal qui ont leur propre police (elle passe avant celle
// de « defaults »).
func (o Options) profilsAvecPolice(f string) {
	b, err := os.ReadFile(f)
	if err != nil {
		return
	}
	s := string(b)
	n, err := lireJSONC(s)
	if err != nil || n.genre != '{' {
		return
	}
	p := n.cle("profiles")
	if p == nil || p.genre != '{' {
		return
	}
	liste := p.cle("list")
	if liste == nil {
		return
	}
	for _, e := range liste.elements {
		if e.genre != '{' {
			continue
		}
		police, nom := e.cle("font"), e.cle("name")
		if police == nil || police.genre != '{' || police.cle("face") == nil {
			continue
		}
		face, _ := ChaineJSONC(s[police.debut:police.fin], "face")
		if !strings.Contains(strings.ToLower(face), "nerd font") {
			n := "?"
			if nom != nil {
				n, _ = ChaineJSONC(`{"n":`+s[nom.debut:nom.fin]+`}`, "n")
			}
			o.etape(i18n.T("Attention"), i18n.T("Windows Terminal : le profil « %s » garde sa police (%s) ; change-la dans ses réglages (Apparence > Police)", n, face))
		}
	}
}

// terminalCourant : le terminal où « agentline install » est lancé, et comment y choisir la police.
func (o Options) terminalCourant() (string, string) {
	famille := o.police().Famille
	p := o.env("TERM_PROGRAM")
	switch {
	case o.env("WT_SESSION") != "":
		return "Windows Terminal", i18n.T("Windows Terminal : Paramètres > Profils > Par défaut > Apparence > Police : %s", famille)
	case p == "vscode":
		return "VS Code", i18n.T("VS Code : réglage terminal.integrated.fontFamily : %s", famille)
	case p == "mintty":
		return "Git Bash", i18n.T("Git Bash (mintty) : clic droit sur la fenêtre > Options > Text > Font : %s", famille)
	case p == "iTerm.app":
		return "iTerm2", i18n.T("iTerm2 : Settings > Profiles > Text > Font : %s", famille)
	case p == "Apple_Terminal":
		return "Terminal", i18n.T("Terminal : Réglages > Profils > Texte > Police > Modifier… : %s (pour chaque profil utilisé)", famille)
	case p == "WezTerm":
		return "WezTerm", i18n.T("WezTerm : dans ~/.wezterm.lua, config.font = wezterm.font '%s'", famille)
	case p == "ghostty":
		return "Ghostty", i18n.T("Ghostty : dans ~/.config/ghostty/config, font-family = %s", famille)
	case o.env("KITTY_WINDOW_ID") != "":
		return "kitty", i18n.T("kitty : dans ~/.config/kitty/kitty.conf, font_family %s", famille)
	case o.env("ALACRITTY_WINDOW_ID") != "" || o.env("ALACRITTY_SOCKET") != "":
		return "Alacritty", i18n.T("Alacritty : dans ~/.config/alacritty/alacritty.toml, [font.normal] family = \"%s\"", famille)
	case o.env("KONSOLE_VERSION") != "":
		return "Konsole", i18n.T("Konsole : Configuration > Modifier le profil actuel > Apparence > Police : %s", famille)
	case o.env("GNOME_TERMINAL_SCREEN") != "" || o.env("VTE_VERSION") != "":
		return "GNOME Terminal", i18n.T("GNOME Terminal (ou autre terminal VTE) : Préférences > ton profil > Texte > Police personnalisée : %s", famille)
	}
	return "inconnu", i18n.T("choisis la police %s dans les réglages de ton terminal", famille)
}
