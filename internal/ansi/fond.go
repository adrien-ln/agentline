package ansi

import (
	"strconv"
	"strings"
)

// LireFond : le fond d'après son nom (dark ou sombre, light ou clair) ; auto (ou vide) : à détecter.
// Faux si le nom est inconnu.
func LireFond(s string) (f Fond, auto, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dark", "sombre":
		return Sombre, false, true
	case "light", "clair":
		return Clair, false, true
	case "auto", "":
		return Sombre, true, true
	}
	return Sombre, false, false
}

// FondDeCOLORFGBG : le fond d'après COLORFGBG (« 15;0 », « 0;default;15 », posée par rxvt, Konsole…) : sa
// dernière valeur est la couleur ANSI du fond ; comme Vim, 0 à 6 et 8 sont sombres, les autres claires.
// Faux si la variable est absente ou illisible.
func FondDeCOLORFGBG(s string) (Fond, bool) {
	if s == "" {
		return Sombre, false
	}
	x := strings.Split(s, ";")
	n, err := strconv.Atoi(x[len(x)-1])
	if err != nil || n < 0 || n > 15 {
		return Sombre, false
	}
	if n <= 6 || n == 8 {
		return Sombre, true
	}
	return Clair, true
}

// RequeteFond : la question au terminal sur sa couleur de fond (OSC 11), suivie d'une demande d'attributs
// (DA1) à laquelle tous les terminaux répondent : sa réponse marque la fin de l'attente, même pour ceux
// qui ignorent OSC 11.
const RequeteFond = "\x1b]11;?\x1b\\" + "\x1b[c"

// FondDeReponse : le fond d'après la réponse du terminal à RequeteFond (« ESC ]11;rgb:RRRR/GGGG/BBBB »,
// terminée par BEL ou ESC \, composantes de 1 à 4 chiffres hexadécimaux) ; faux sans réponse lisible.
func FondDeReponse(r string) (Fond, bool) {
	i := strings.Index(r, "\x1b]11;rgb:")
	if i < 0 {
		return Sombre, false
	}
	r = r[i+len("\x1b]11;rgb:"):]
	if j := strings.IndexAny(r, "\x07\x1b"); j >= 0 {
		r = r[:j]
	} else {
		return Sombre, false
	}
	c := strings.Split(r, "/")
	if len(c) != 3 {
		return Sombre, false
	}
	var v [3]int
	for k, x := range c {
		if len(x) < 1 || len(x) > 4 {
			return Sombre, false
		}
		n, err := strconv.ParseUint(x, 16, 16)
		if err != nil {
			return Sombre, false
		}
		v[k] = int(n * 255 / (1<<(4*len(x)) - 1)) // ramené à 0-255
	}
	if EstClaire(hex(v[0], v[1], v[2])) {
		return Clair, true
	}
	return Sombre, true
}

// FinReponse : vrai quand la réponse lue contient celle de DA1 (« ESC [? … c ») : plus rien à attendre.
func FinReponse(r string) bool {
	i := strings.Index(r, "\x1b[?")
	return i >= 0 && strings.IndexByte(r[i:], 'c') >= 0
}
