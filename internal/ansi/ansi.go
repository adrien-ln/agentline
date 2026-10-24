// Package ansi : les séquences de couleur (24 bits, 256, 16 couleurs ou aucune, voir Profil), les calculs de
// couleur (conversion, contraste) et le texte visible (séquences retirées), communs à la statusline et au sélecteur.
package ansi

import (
	"fmt"
	"strconv"
	"strings"
)

// Z : retour aux attributs par défaut.
const Z = "\x1b[0m"

// RGB : les trois composantes d'une couleur « rrggbb » (0 si elle est illisible).
func RGB(h string) (int, int, int) {
	v, _ := strconv.ParseUint(h, 16, 32)
	return int(v >> 16 & 255), int(v >> 8 & 255), int(v & 255)
}

// Fg : la séquence qui donne la couleur « rrggbb » au texte.
func Fg(h string) string {
	r, g, b := RGB(h)
	return FgRGB(r, g, b)
}

// FgRGB : la séquence qui donne au texte la couleur de composantes r, g, b (0 à 255).
func FgRGB(r, g, b int) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// Bg : la séquence qui donne la couleur « rrggbb » au fond.
func Bg(h string) string {
	r, g, b := RGB(h)
	return BgRGB(r, g, b)
}

// BgRGB : la séquence qui donne au fond la couleur de composantes r, g, b (0 à 255).
func BgRGB(r, g, b int) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// Visible : le texte sans ses séquences d'échappement (couleurs, liens OSC 8).
func Visible(s string) string {
	var t strings.Builder
	for {
		i := strings.IndexByte(s, 0x1b)
		if i < 0 {
			break
		}
		t.WriteString(s[:i])
		s = s[i+1:]
		if strings.HasPrefix(s, "]") {
			if j := strings.IndexByte(s, 0x07); j >= 0 {
				s = s[j+1:]
			}
		} else if j := strings.IndexByte(s, 'm'); j >= 0 {
			s = s[j+1:]
		}
	}
	t.WriteString(s)
	return t.String()
}
