package theme

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Les réglages de l'utilisateur ($AGENTLINE_CONFIG), les mêmes fichiers que la version bash :
//   defaut   : le nom du thème par défaut
//   projets  : une ligne par projet, « identifiant<TAB>thème »
// et un de plus, propre à la version Go :
//   branches : une ligne par règle, « motif<TAB>thème »

// Noms : tous les thèmes, triés, sans doublon (un thème de l'utilisateur masque celui de l'installation).
func (s Sources) Noms() []string {
	vus := map[string]bool{}
	var noms []string
	ajouter := func(f string) {
		n := strings.TrimSuffix(f, ".theme")
		if f != n && nomValide.MatchString(n) && !vus[n] {
			vus[n] = true
			noms = append(noms, n)
		}
	}
	if e, err := os.ReadDir(filepath.Join(s.Config, "themes")); err == nil {
		for _, f := range e {
			ajouter(f.Name())
		}
	}
	if s.Integre != nil {
		if e, err := fs.ReadDir(s.Integre, "themes"); err == nil {
			for _, f := range e {
				ajouter(f.Name())
			}
		}
	}
	sort.Strings(noms)
	return noms
}

// Entete : le nom affiché (1re ligne « # … ») et les tags (2e ligne « # tags: … ») d'un thème. Les tags
// anglais (popular, red, green…) sont rendus sous leur nom français, celui des filtres du sélecteur.
func (s Sources) Entete(nom string) (titre string, tags []string) {
	b, ok := s.Fichier(nom)
	if !ok {
		return nom, nil
	}
	lignes := strings.SplitN(string(b), "\n", 3)
	titre = strings.TrimPrefix(strings.TrimRight(lignes[0], "\r"), "# ")
	if len(lignes) > 1 {
		if l := strings.TrimRight(lignes[1], "\r"); strings.HasPrefix(l, "# tags: ") {
			tags = strings.Fields(strings.ReplaceAll(strings.TrimPrefix(l, "# tags: "), ",", " "))
			for k, t := range tags {
				if f, ok := tagsAnglais[t]; ok {
					tags[k] = f
				}
			}
		}
	}
	return titre, tags
}

// tagsAnglais : tag anglais -> tag français (les thèmes intégrés écrivent les tags en français).
var tagsAnglais = map[string]string{"popular": "populaire", "red": "rouge", "yellow": "jaune", "green": "vert",
	"blue": "bleu", "purple": "violet", "pink": "rose", "gray": "gris", "grey": "gris"}

// EcrireDefaut : le thème par défaut. Écriture atomique : un rendu lancé au même instant lit l'ancien
// thème ou le nouveau, jamais un fichier vide.
func (s Sources) EcrireDefaut(nom string) error {
	return EcrireAtomique(s.Config, "defaut", nom+"\n")
}

// Associations : les lignes du fichier projets, dans l'ordre (identifiant, thème).
func (s Sources) Associations() [][2]string { return s.lireTable("projets") }

// Associer : le thème du projet (nom vide : retire l'association).
func (s Sources) Associer(id, nom string) error { return s.ecrireTable("projets", id, nom) }

// ReglesBranches : les lignes du fichier branches, dans l'ordre (motif, thème).
func (s Sources) ReglesBranches() [][2]string { return s.lireTable("branches") }

// RegleBranche : le thème des branches qui correspondent au motif (nom vide : retire la règle).
func (s Sources) RegleBranche(motif, nom string) error { return s.ecrireTable("branches", motif, nom) }

// MotifValide : un nom de branche ou un motif (« * », « ? ») utilisable comme règle.
func MotifValide(m string) bool {
	return strings.TrimSpace(m) != "" && !strings.ContainsAny(m, "\t\r\n")
}

// lireTable : les lignes « clé<TAB>thème » d'un fichier de réglages, dans l'ordre.
func (s Sources) lireTable(fichier string) [][2]string {
	b, err := os.ReadFile(filepath.Join(s.Config, fichier))
	if err != nil {
		return nil
	}
	var a [][2]string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.Trim(strings.TrimRight(l, "\r"), "\t")
		if l == "" {
			continue
		}
		cle, nom, _ := strings.Cut(l, "\t")
		a = append(a, [2]string{cle, strings.Trim(nom, "\t")})
	}
	return a
}

// ecrireTable : remplace ou ajoute la ligne de la clé (nom vide : la retire). Écriture atomique.
func (s Sources) ecrireTable(fichier, cle, nom string) error {
	var b strings.Builder
	for _, a := range s.lireTable(fichier) {
		if a[0] != cle {
			fmt.Fprintf(&b, "%s\t%s\n", a[0], a[1])
		}
	}
	if nom != "" {
		fmt.Fprintf(&b, "%s\t%s\n", cle, nom)
	}
	return EcrireAtomique(s.Config, fichier, b.String())
}

// EcrireAtomique : le fichier du dossier (créé au besoin) remplacé d'un coup par le contenu : fichier
// temporaire à côté, puis renommage. Un lecteur voit l'ancien contenu ou le nouveau, jamais un mélange.
func EcrireAtomique(dossier, fichier, contenu string) error {
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dossier, "."+fichier+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(contenu); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), filepath.Join(dossier, fichier))
}
