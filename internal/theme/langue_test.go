package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// TestMain : les textes attendus par les tests sont en français.
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	os.Exit(m.Run())
}

// TestThemeEnAnglais : un thème écrit avec les noms anglais (variables et tags) se lit comme le même
// thème écrit en français.
func TestThemeEnAnglais(t *testing.T) {
	fr := "# Essai\n# tags: bleu, violet, populaire\nTEXTE=c0caf5 DISCRET=737aa2 SEP=3b4261 ETEINT=545c7e\n" +
		"MODELE=bb9af7 MODELE_FIN=112233 DOSSIER=7aa2f7 LIEU=7dcfff\nD0=9ece6a D50=e0af68 D75=ff9e64 D100=f7768e\n" +
		"INTENSITE=(7dcfff 7aa2f7 bb9af7 ff007c ff9e64)\n"
	en := "# Essai\n# tags: blue, purple, popular\nTEXT=c0caf5 MUTED=737aa2 SEP=3b4261 DIM=545c7e\n" +
		"MODEL=bb9af7 MODEL_END=112233 FOLDER=7aa2f7 BRANCH=7dcfff\nD0=9ece6a D50=e0af68 D75=ff9e64 D100=f7768e\n" +
		"INTENSITY=(7dcfff 7aa2f7 bb9af7 ff007c ff9e64)\n"
	if a, b := Lire([]byte(fr)), Lire([]byte(en)); a != b || a.Intensite[4] != "ff9e64" || a.ModeleFin != "112233" {
		t.Errorf("palettes différentes :\n%+v\n%+v", a, b)
	}
	// fond clair et couleurs pour l'autre fond : « # background: light », _LIGHT/_DARK
	frc := Lire([]byte("# fond: clair\nTEXTE=111111 TEXTE_SOMBRE=eeeeee\nINTENSITE_SOMBRE=(1 2 3 4 5)\n"))
	enc := Lire([]byte("# background: light\nTEXT=111111 TEXT_DARK=eeeeee\nINTENSITY_DARK=(1 2 3 4 5)\n"))
	if !reflect.DeepEqual(frc, enc) || !enc.Clair || enc.Autre.Texte != "eeeeee" || enc.Autre.Intensite[4] != "5" {
		t.Errorf("fond clair :\n%+v\n%+v", frc, enc)
	}
	fro := Lire([]byte("TEXTE=111111 TEXTE_CLAIR=222222\n"))
	eno := Lire([]byte("TEXT=111111 TEXT_LIGHT=222222\n"))
	if !reflect.DeepEqual(fro, eno) || eno.Clair || eno.Autre.Texte != "222222" {
		t.Errorf("couleurs pour fond clair :\n%+v\n%+v", fro, eno)
	}
	d := t.TempDir()
	os.MkdirAll(filepath.Join(d, "themes"), 0o755)
	os.WriteFile(filepath.Join(d, "themes", "essai-fr.theme"), []byte(fr), 0o644)
	os.WriteFile(filepath.Join(d, "themes", "essai-en.theme"), []byte(en), 0o644)
	s := Sources{Config: d}
	_, tfr := s.Entete("essai-fr")
	_, ten := s.Entete("essai-en")
	if !reflect.DeepEqual(tfr, ten) || !reflect.DeepEqual(tfr, []string{"bleu", "violet", "populaire"}) {
		t.Errorf("tags : %v / %v", tfr, ten)
	}
}
