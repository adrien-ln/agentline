package style

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStyleDeProjet : le projet ne règle que ses différences, même un retour au défaut ; son fichier rappelle
// son identifiant, et un autre projet au nom ressemblant ne s'en sert pas.
func TestStyleDeProjet(t *testing.T) {
	d := t.TempDir()
	general := Defaut()
	general.Definir("cost.when", "always")
	general.Definir("ctx.width", "12")
	id := "gitlab.com/adrien-ln/agentline"
	s := &Surcharge{}
	if err := s.Definir("cost.when", "api"); err != nil { // le défaut : annule le réglage général
		t.Fatal(err)
	}
	s.Definir("hide", "pr")
	if err := EcrireProjet(d, id, s); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(d, DossierProjets, "gitlab-com-adrien-ln-agentline"))
	if !strings.HasPrefix(string(b), "# "+id+"\n") || !strings.Contains(string(b), "cost.when=api") {
		t.Errorf("fichier :\n%s", b)
	}
	e := Superposer(general, ChargerProjet(d, id))
	if e.Valeur("cost.when") != "api" || e.Valeur("ctx.width") != "12" || !e.Masques()["pr"] {
		t.Errorf("superposé : cost.when=%s ctx.width=%s", e.Valeur("cost.when"), e.Valeur("ctx.width"))
	}
	if ChargerProjet(d, "gitlab.com/adrien/ln-agentline") != nil { // même nom de fichier, autre projet
		t.Error("un autre projet ne doit pas prendre ce style")
	}
	if ec := Ecart(e, general); ec.Nombre() != 2 || !ec.Regle("cost.when") || !ec.Regle("hide") {
		t.Errorf("écart : %d options", ec.Nombre())
	}
	if err := EcrireProjet(d, id, &Surcharge{}); err != nil || ChargerProjet(d, id) != nil {
		t.Errorf("sans option, le fichier part : %v", err)
	}
	if s := ChargerPour(d, id, func(string) string { return "default" }); !s.EstDefaut() {
		t.Error("AGENTLINE_STYLE=default ignore aussi le style du projet")
	}
}
