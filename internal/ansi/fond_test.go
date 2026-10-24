package ansi

import "testing"

func TestLireFond(t *testing.T) {
	for _, c := range []struct {
		s        string
		f        Fond
		auto, ok bool
	}{{"dark", Sombre, false, true}, {"Light", Clair, false, true}, {"clair", Clair, false, true}, {"sombre", Sombre, false, true},
		{"auto", Sombre, true, true}, {"", Sombre, true, true}, {"gris", Sombre, false, false}} {
		if f, a, ok := LireFond(c.s); f != c.f || a != c.auto || ok != c.ok {
			t.Errorf("LireFond(%q) = %v %v %v", c.s, f, a, ok)
		}
	}
}

func TestFondDeCOLORFGBG(t *testing.T) {
	for s, attendu := range map[string]struct {
		f  Fond
		ok bool
	}{"15;0": {Sombre, true}, "0;15": {Clair, true}, "0;default;15": {Clair, true}, "7;8": {Sombre, true}, "0;7": {Clair, true},
		"": {Sombre, false}, "15;default": {Sombre, false}, "x": {Sombre, false}, "0;99": {Sombre, false}} {
		if f, ok := FondDeCOLORFGBG(s); f != attendu.f || ok != attendu.ok {
			t.Errorf("COLORFGBG=%q : %v %v", s, f, ok)
		}
	}
}

func TestFondDeReponse(t *testing.T) {
	for r, attendu := range map[string]struct {
		f  Fond
		ok bool
	}{
		"\x1b]11;rgb:ffff/ffff/ffff\x1b\\\x1b[?62;22c": {Clair, true},
		"\x1b]11;rgb:1a1a/1b1b/2626\x07":               {Sombre, true},
		"\x1b]11;rgb:fd/f6/e3\x07":                     {Clair, true}, // Solarized Light, 2 chiffres
		"\x1b]11;rgb:e1e2/e2e2/e7e7\x1b\\":             {Clair, true}, // Tokyo Night Day
		"\x1b]11;rgb:0000/2b2b/3636\x1b\\":             {Sombre, true},
		"\x1b[?1;2c":                                   {Sombre, false}, // OSC 11 ignoré : seulement DA1
		"\x1b]11;rgb:ffff/ffff":                        {Sombre, false},
		"":                                             {Sombre, false},
	} {
		if f, ok := FondDeReponse(r); f != attendu.f || ok != attendu.ok {
			t.Errorf("%q : %v %v", r, f, ok)
		}
	}
	if !FinReponse("\x1b]11;rgb:0/0/0\x07\x1b[?62;c") || FinReponse("\x1b]11;rgb:0/0/0\x07") || FinReponse("\x1b[?62;") {
		t.Error("FinReponse")
	}
}
