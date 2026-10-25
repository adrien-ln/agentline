package selecteur

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ecranPolices : l'écran de l'aperçu des polices, avec un essai qui note les polices posées.
func ecranPolices(t *testing.T) (*Polices, *[]string) {
	t.Helper()
	m, _ := nouveau(t)
	var posees []string
	liste := []ChoixPolice{{Nom: "firacode", Titre: "Fira Code", Famille: "FiraCode Nerd Font", Installee: true},
		{Nom: "hack", Titre: "Hack", Famille: "Hack Nerd Font"},
		{Nom: "meslo", Titre: "Meslo", Famille: "MesloLGS Nerd Font", Installee: true}}
	p := NouvellesPolices(m.apercu, liste, "Windows Terminal", "Cascadia Mono", func(f string) error { posees = append(posees, f); return nil })
	return p, &posees
}

func touchePolices(p *Polices, k tea.KeyType) tea.Cmd {
	_, cmd := p.Update(tea.KeyMsg{Type: k})
	return cmd
}

// TestPolicesAvertissement : rien ne change avant Entrée ; l'avertissement dit ce qui va se passer.
func TestPolicesAvertissement(t *testing.T) {
	p, posees := ecranPolices(t)
	touchePolices(p, tea.KeyDown)
	if p.i != 0 || len(*posees) != 0 || !strings.Contains(visible(p.View()), "copie .bak") {
		t.Errorf("avant le départ : ligne %d, posées %v\n%s", p.i, *posees, visible(p.View()))
	}
	if touchePolices(p, tea.KeyEsc) == nil || p.Garder {
		t.Error("Échap avant le départ : quitter sans rien garder")
	}
}

// TestPolicesEssai : un déplacement n'essaie la police qu'une fois le délai passé sans autre déplacement ; une
// police non installée n'est pas essayée ; Entrée garde la police de la ligne.
func TestPolicesEssai(t *testing.T) {
	p, posees := ecranPolices(t)
	touchePolices(p, tea.KeyEnter)
	touchePolices(p, tea.KeyDown) // Hack, non installée
	vieux := p.jeton
	touchePolices(p, tea.KeyDown) // Meslo
	p.Update(essaiPolice(vieux))
	if len(*posees) != 0 {
		t.Errorf("un essai dépassé ne pose rien : %v", *posees)
	}
	p.Update(essaiPolice(p.jeton))
	if len(*posees) != 1 || (*posees)[0] != "MesloLGS Nerd Font" || !strings.Contains(visible(p.View()), "Avec cette police") {
		t.Errorf("après le délai : %v", *posees)
	}
	touchePolices(p, tea.KeyUp)
	p.Update(essaiPolice(p.jeton))
	if len(*posees) != 1 {
		t.Errorf("Hack n'est pas installée : rien à poser (%v)", *posees)
	}
	if touchePolices(p, tea.KeyEnter) != nil || p.Garder || !strings.Contains(visible(p.View()), "agentline font set hack") {
		t.Error("Entrée sur une police non installée : elle le dit, sans quitter")
	}
	touchePolices(p, tea.KeyUp) // Fira Code, gardée tout de suite, sans attendre le délai
	if touchePolices(p, tea.KeyEnter) == nil || !p.Garder || p.Choix.Nom != "firacode" || (*posees)[len(*posees)-1] != "FiraCode Nerd Font" {
		t.Errorf("Entrée : garder Fira Code (%+v, %v)", p.Choix, *posees)
	}
}
