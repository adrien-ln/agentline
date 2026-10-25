package theme

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adrien-ln/agentline"
)

// TestFormatAllerRetour : chaque thème intégré, réécrit par Format (noms anglais) puis relu, redonne la même
// palette, le même titre et les mêmes tags.
func TestFormatAllerRetour(t *testing.T) {
	s := Sources{Integre: agentline.Themes}
	u := Sources{Config: t.TempDir()}
	for _, nom := range s.Noms() {
		p, _ := s.Charger(nom)
		titre, tags := s.Entete(nom)
		b := Format(p, titre, tags)
		if strings.Contains(string(b), "TEXTE") || strings.Contains(string(b), "INTENSITE") {
			t.Errorf("%s : Format doit écrire les noms anglais\n%s", nom, b)
		}
		if err := u.EcrireTheme(nom, b); err != nil {
			t.Fatal(err)
		}
		q, ok := u.Charger(nom)
		t2, g2 := u.Entete(nom)
		if !ok || !reflect.DeepEqual(p, q) || t2 != titre || !reflect.DeepEqual(g2, tags) {
			t.Errorf("%s : relu différent\n%s\navant %+v %q %v\naprès %+v %q %v", nom, b, p, titre, tags, q, t2, g2)
		}
		if !u.DeLUtilisateur(nom) {
			t.Errorf("%s : écrit, il doit être de l'utilisateur", nom)
		}
	}
	if err := u.EcrireTheme("Pas Bon", nil); err == nil || NomValide("Pas Bon") {
		t.Error("un nom invalide doit être refusé")
	}
	if s.DeLUtilisateur("tokyo") {
		t.Error("tokyo est intégré")
	}
}
