package theme

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// Migrer : copie l'ancien dossier des réglages (~/.config/statusline) dans le nouveau
// (~/.config/agentline), une seule fois : seulement si le nouveau n'existe pas encore et que l'ancien
// est un dossier. L'ancien est gardé tel quel. La copie se fait dans un dossier temporaire à côté du
// nouveau, renommé à la fin : un autre processus (la statusline, lancée à chaque événement) voit le
// nouveau dossier complet ou pas du tout. Vrai si la copie a eu lieu ici.
func Migrer(ancien, nouveau string) (bool, error) {
	if _, err := os.Lstat(nouveau); !errors.Is(err, os.ErrNotExist) {
		return false, nil // déjà là (ou illisible : on n'y touche pas)
	}
	fi, err := os.Stat(ancien)
	if err != nil || !fi.IsDir() {
		return false, nil
	}
	parent := filepath.Dir(nouveau)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return false, err
	}
	tmp, err := os.MkdirTemp(parent, "."+filepath.Base(nouveau)+".migration-*")
	if err != nil {
		return false, err
	}
	if err := copierDossier(ancien, tmp, 0); err != nil {
		os.RemoveAll(tmp)
		return false, fmt.Errorf(i18n.T("copie de %s : %w"), ancien, err)
	}
	os.Chmod(tmp, fi.Mode().Perm()|0o700) // MkdirTemp crée en 0700
	if err := os.Rename(tmp, nouveau); err != nil {
		os.RemoveAll(tmp)
		if _, e := os.Stat(nouveau); e == nil {
			return false, nil // un autre processus l'a fait au même moment
		}
		return false, err
	}
	return true, nil
}

// copierDossier : le contenu de src dans dst (qui existe), liens suivis (un fichier lié par un
// gestionnaire de dotfiles est copié avec son contenu) ; ni fichier spécial, ni plus de 8 niveaux.
func copierDossier(src, dst string, profondeur int) error {
	if profondeur > 8 {
		return nil
	}
	entrees, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entrees {
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		fi, err := os.Stat(s)
		if err != nil {
			continue // lien cassé
		}
		switch {
		case fi.IsDir():
			if err := os.Mkdir(d, fi.Mode().Perm()|0o700); err != nil {
				return err
			}
			if err := copierDossier(s, d, profondeur+1); err != nil {
				return err
			}
		case fi.Mode().IsRegular():
			if err := copierFichier(s, d, fi.Mode().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

func copierFichier(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
