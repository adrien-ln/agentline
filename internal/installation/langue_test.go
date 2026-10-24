package installation

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// TestMain : les textes attendus par les tests sont en français (TestAnglais vérifie l'anglais).
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	os.Exit(m.Run())
}

// TestAnglais : installation, doctor et désinstallation simulés, compte rendu sans un mot de français.
func TestAnglais(t *testing.T) {
	avant := i18n.Choisir(i18n.EN)
	defer i18n.Choisir(avant)
	for _, os_ := range []string{"windows", "darwin", "linux"} {
		s := simuler(t, os_)
		var doctor bytes.Buffer
		Afficher(&doctor, Diagnostiquer(s.Options))
		if err := Installer(s.Options); err != nil {
			t.Fatal(err)
		}
		Afficher(&doctor, Diagnostiquer(s.Options))
		if err := Desinstaller(s.Options); err != nil {
			t.Fatal(err)
		}
		texte := s.sortie.String() + doctor.String()
		for _, x := range []string{"Binary", "Settings", "Uninstalled", "Font"} {
			if !strings.Contains(texte, x) {
				t.Errorf("%s : %q absent\n%s", os_, x, texte)
			}
		}
		for _, x := range []string{"«", " : ", " ; ", "é", "è", "à "} {
			if strings.Contains(texte, x) {
				t.Errorf("%s : %q dans le compte rendu anglais\n%s", os_, x, texte)
			}
		}
	}
}
