package ansi

import "testing"

func TestDetecterNiveau(t *testing.T) {
	for _, c := range []struct {
		env    map[string]string
		niveau Niveau
		raison string
	}{
		{nil, Truecolor, "défaut"},
		{map[string]string{"AGENTLINE_COLORS": "256", "COLORTERM": "truecolor"}, Couleurs256, "AGENTLINE_COLORS"},
		{map[string]string{"AGENTLINE_COLORS": "truecolor", "NO_COLOR": "1"}, Truecolor, "AGENTLINE_COLORS"},
		{map[string]string{"AGENTLINE_COLORS": "n'importe quoi", "TERM": "xterm-256color"}, Couleurs256, "TERM=xterm-256color"},
		{map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"}, SansCouleur, "NO_COLOR"},
		{map[string]string{"NO_COLOR": "", "COLORTERM": "truecolor"}, Truecolor, "COLORTERM=truecolor"}, // vide : ignoré (no-color.org)
		{map[string]string{"TERM": "dumb", "COLORTERM": "truecolor"}, SansCouleur, "TERM=dumb"},
		{map[string]string{"COLORTERM": "24bit", "TERM": "xterm-256color"}, Truecolor, "COLORTERM=24bit"},
		{map[string]string{"COLORTERM": "yes", "TERM": "xterm-256color"}, Couleurs256, "TERM=xterm-256color"},
		{map[string]string{"WT_SESSION": "1b2c", "TERM": "xterm-256color"}, Truecolor, "WT_SESSION"},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM": "xterm-256color"}, Truecolor, "TERM_PROGRAM=iTerm.app"},
		{map[string]string{"TERM_PROGRAM": "vscode"}, Truecolor, "TERM_PROGRAM=vscode"},
		{map[string]string{"TERM_PROGRAM": "WezTerm"}, Truecolor, "TERM_PROGRAM=WezTerm"},
		{map[string]string{"TERM_PROGRAM": "ghostty"}, Truecolor, "TERM_PROGRAM=ghostty"},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal", "TERM": "xterm-256color"}, Couleurs256, "TERM_PROGRAM=Apple_Terminal"},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal", "COLORTERM": "truecolor"}, Truecolor, "COLORTERM=truecolor"},
		{map[string]string{"TERM_PROGRAM": "tmux", "TERM": "tmux-256color"}, Couleurs256, "TERM=tmux-256color"},
		{map[string]string{"TERM": "xterm-kitty"}, Truecolor, "TERM=xterm-kitty"},
		{map[string]string{"TERM": "alacritty"}, Truecolor, "TERM=alacritty"},
		{map[string]string{"TERM": "xterm-direct"}, Truecolor, "TERM=xterm-direct"},
		{map[string]string{"TERM": "xterm-256color"}, Couleurs256, "TERM=xterm-256color"},
		{map[string]string{"TERM": "linux"}, Couleurs16, "TERM=linux"},
		{map[string]string{"TERM": "vt100"}, Couleurs16, "TERM=vt100"},
		{map[string]string{"TERM": "xterm"}, Couleurs256, "TERM=xterm"},
		{map[string]string{"TERM": "screen"}, Couleurs256, "TERM=screen"},
	} {
		n, r := DetecterNiveau(func(k string) string { return c.env[k] })
		if n != c.niveau || r != c.raison {
			t.Errorf("%v : %v (%s), attendu %v (%s)", c.env, n, r, c.niveau, c.raison)
		}
	}
	if !NiveauImpose("AGENTLINE_COLORS") || !NiveauImpose("NO_COLOR") || NiveauImpose("TERM=xterm") || NiveauImpose("défaut") {
		t.Error("NiveauImpose")
	}
}
