package installation

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// L'aperçu des polices (agentline font preview) : la police du terminal en cours change pour de vrai,
// le temps de choisir, puis revient. Une copie .bak est faite au premier changement ; le contenu d'origine
// est gardé pour être réécrit tel quel ; un repère dans les réglages d'agentline permet de remettre la
// police si l'aperçu est interrompu (processus tué).

// ErrSansApercu : le terminal en cours n'est pas l'un de ceux dont agentline sait changer la police à chaud.
var ErrSansApercu = errors.New("terminal sans aperçu")

const fichierRepere = "police-apercu"

// ApercuPolice : l'aperçu en cours dans un terminal.
type ApercuPolice struct {
	o                 Options
	Terminal, Fichier string
	Avant             string // la police d'origine (vide : celle par défaut du terminal)
	profil            string // Windows Terminal : le profil qui a sa propre police (guid) ; vide : defaults
	chemin            []string
	origine, ecrit    []byte // le contenu d'origine, et le dernier écrit par l'aperçu
	commence          bool
	Copie             string // la copie .bak faite au premier changement
}

// repere : ce qu'il faut pour remettre la police si l'aperçu est interrompu.
type repere struct {
	Fichier string   `json:"file"`
	Profil  string   `json:"profile,omitempty"`
	Chemin  []string `json:"path"`
	Avant   string   `json:"before"`
}

// PreparerApercu : l'aperçu pour le terminal en cours (Windows Terminal ou VS Code, s'il a ses réglages ici) ;
// ErrSansApercu sinon. Rien n'est encore modifié.
func (o Options) PreparerApercu() (*ApercuPolice, error) {
	t, _ := o.terminalCourant()
	for _, r := range o.reglagesTerminaux() {
		if r.terminal != t || (t != "Windows Terminal" && t != "VS Code") {
			continue
		}
		b, err := os.ReadFile(r.fichier)
		if err != nil {
			return nil, err
		}
		a := &ApercuPolice{o: o, Terminal: t, Fichier: r.fichier, chemin: r.chemin, origine: b}
		if t == "Windows Terminal" {
			a.profil = profilAvecPolice(string(b), o.env("WT_PROFILE_ID"))
		}
		a.Avant, _ = a.lire(string(b))
		return a, nil
	}
	return nil, ErrSansApercu
}

// profilAvecPolice : le guid du profil de Windows Terminal, s'il a sa propre police (elle passe avant celle de
// defaults) ; vide sinon.
func profilAvecPolice(s, guid string) string {
	if p := noeudProfil(s, guid); p != nil {
		if f := p.cle("font"); f != nil && f.genre == '{' && f.cle("face") != nil {
			return guid
		}
	}
	return ""
}

// noeudProfil : l'objet du profil guid dans profiles.list ; nil s'il n'y est pas.
func noeudProfil(s, guid string) *noeud {
	if guid == "" {
		return nil
	}
	n, err := lireJSONC(s)
	if err != nil || n.genre != '{' {
		return nil
	}
	p := n.cle("profiles")
	if p == nil || p.genre != '{' {
		return nil
	}
	l := p.cle("list")
	if l == nil {
		return nil
	}
	for _, e := range l.elements {
		if g := e.cle("guid"); e.genre == '{' && g != nil && strings.EqualFold(strings.Trim(s[g.debut:g.fin], `"`), guid) {
			return e
		}
	}
	return nil
}

// lire : la police réglée là où l'aperçu la change.
func (a *ApercuPolice) lire(s string) (string, bool) {
	if a.profil != "" {
		if p := noeudProfil(s, a.profil); p != nil {
			return ChaineJSONC(s[p.debut:p.fin], "font", "face")
		}
	}
	return ChaineJSONC(s, a.chemin...)
}

// poser : le contenu avec la police famille, là où l'aperçu la change.
func (a *ApercuPolice) poser(s, famille string) (string, error) {
	if a.profil != "" {
		if p := noeudProfil(s, a.profil); p != nil {
			n, _, err := PoserChaineJSONC(s[p.debut:p.fin], famille, "font", "face")
			if err != nil {
				return s, err
			}
			return s[:p.debut] + n + s[p.fin:], nil
		}
	}
	n, _, err := PoserChaineJSONC(s, famille, a.chemin...)
	return n, err
}

// Essayer : la police famille dans les réglages du terminal. Au premier appel : la copie .bak et le repère.
func (a *ApercuPolice) Essayer(famille string) error {
	cour, err := os.ReadFile(a.Fichier)
	if err != nil {
		return err
	}
	n, err := a.poser(string(cour), famille)
	if err != nil {
		return err
	}
	if err := jsoncValide(n); err != nil {
		return err
	}
	if !a.commence {
		a.Copie = CopieNom(a.Fichier, a.o.Horodatage)
		if err := os.WriteFile(a.Copie, cour, 0o644); err != nil {
			return err
		}
		a.ecrireRepere()
		a.commence = true
	}
	a.ecrit = []byte(n)
	return remplacer(a.Fichier, a.ecrit)
}

// Remettre : la police d'origine. Si personne n'a touché au fichier pendant l'aperçu, il est réécrit tel
// qu'il était ; sinon seule la police y est remise. Le repère est effacé.
func (a *ApercuPolice) Remettre() error {
	if !a.commence {
		return nil
	}
	defer a.effacerRepere()
	a.commence = false
	cour, err := os.ReadFile(a.Fichier)
	if err != nil {
		return err
	}
	if bytes.Equal(cour, a.ecrit) {
		return remplacer(a.Fichier, a.origine)
	}
	if a.Avant == "" { // la police n'était pas réglée : on ne sait pas retirer la clé sans tout reformater
		return errors.New(i18n.T("%s a changé pendant l'aperçu : remets sa police à la main", a.Fichier))
	}
	n, err := a.poser(string(cour), a.Avant)
	if err != nil {
		return err
	}
	return remplacer(a.Fichier, []byte(n))
}

// Garder : l'aperçu s'arrête sur la police en cours (le repère est effacé, rien n'est remis).
func (a *ApercuPolice) Garder() {
	a.commence = false
	a.effacerRepere()
}

func (a *ApercuPolice) cheminRepere() string {
	if a.o.Reglages == "" {
		return ""
	}
	return filepath.Join(a.o.Reglages, fichierRepere)
}

func (a *ApercuPolice) ecrireRepere() {
	if f := a.cheminRepere(); f != "" {
		b, _ := json.Marshal(repere{a.Fichier, a.profil, a.chemin, a.Avant})
		os.MkdirAll(a.o.Reglages, 0o755)
		os.WriteFile(f, b, 0o644)
	}
}

func (a *ApercuPolice) effacerRepere() {
	if f := a.cheminRepere(); f != "" {
		os.Remove(f)
	}
}

// RemettreApercuInterrompu : si un aperçu a été interrompu (son repère est resté), la police d'origine est
// remise (copie .bak avant) ; le compte rendu, vide s'il n'y avait rien à faire.
func (o Options) RemettreApercuInterrompu() string {
	if o.Reglages == "" {
		return ""
	}
	f := filepath.Join(o.Reglages, fichierRepere)
	b, err := os.ReadFile(f)
	if err != nil {
		return ""
	}
	var r repere
	if json.Unmarshal(b, &r) != nil || r.Fichier == "" {
		os.Remove(f)
		return ""
	}
	a := &ApercuPolice{o: o, Fichier: r.Fichier, profil: r.Profil, chemin: r.Chemin, Avant: r.Avant}
	if r.Avant == "" {
		os.Remove(f)
		return i18n.T("Un aperçu de police a été interrompu : remets la police de %s à la main.", r.Fichier)
	}
	_, copie, err := o.modifier(r.Fichier, func(c string) (string, string) {
		n, err := a.poser(c, r.Avant)
		if err != nil {
			return c, identique
		}
		return n, "modifié"
	}, jsoncValide)
	os.Remove(f)
	if err != nil {
		return i18n.T("Un aperçu de police a été interrompu : %s", err.Error())
	}
	return i18n.T("Un aperçu de police avait été interrompu : %s est remise dans %s%s.", r.Avant, r.Fichier, avecCopie(copie))
}

// ApercuInterrompu : le fichier de réglages dont un aperçu de police a été interrompu (son repère est resté),
// vide sinon. Rien n'est modifié (doctor) : agentline font preview remettra la police.
func (o Options) ApercuInterrompu() string {
	if o.Reglages == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(o.Reglages, fichierRepere))
	if err != nil {
		return ""
	}
	var r repere
	json.Unmarshal(b, &r)
	return r.Fichier
}
