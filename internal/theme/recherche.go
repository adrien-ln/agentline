package theme

import "strings"

// Trouver : le thème désigné par son nom (« tokyo ») ou son titre (« Tokyo Night »), casse indifférente.
// Un thème intégré retiré donne son remplaçant.
func (s Sources) Trouver(saisie string) (string, bool) {
	if r := s.Remplacant(saisie); r != saisie {
		return r, true
	}
	if _, ok := s.Fichier(saisie); ok {
		return saisie, true
	}
	x := strings.ToLower(strings.TrimSpace(saisie))
	for _, n := range s.Noms() {
		if t, _ := s.Entete(n); n == x || strings.ToLower(t) == x {
			return n, true
		}
	}
	return "", false
}

// Proche : le thème dont le nom ou le titre ressemble le plus à la saisie (faute de frappe, début du nom) ;
// vide si aucun n'est assez proche pour être suggéré.
func (s Sources) Proche(saisie string) string {
	x := strings.ToLower(strings.TrimSpace(saisie))
	if x == "" {
		return ""
	}
	meilleur, distance := "", len([]rune(x))/3+1 // au-delà, la suggestion serait du hasard
	for _, n := range s.Noms() {
		t, _ := s.Entete(n)
		for _, c := range []string{n, strings.ToLower(t)} {
			d := ecart(x, c)
			if len([]rune(x)) >= 3 && strings.HasPrefix(c, x) {
				d = 1 // le début d'un nom : « solar » pour solarized
			}
			if d < distance || d == distance && meilleur == "" {
				meilleur, distance = n, d
			}
		}
	}
	return meilleur
}

// ecart : la distance d'édition (insertions, suppressions, substitutions) entre a et b.
func ecart(a, b string) int {
	x, y := []rune(a), []rune(b)
	prec := make([]int, len(y)+1)
	for j := range prec {
		prec[j] = j
	}
	for i := 1; i <= len(x); i++ {
		cour := make([]int, len(y)+1)
		cour[0] = i
		for j := 1; j <= len(y); j++ {
			sub := prec[j-1]
			if x[i-1] != y[j-1] {
				sub++
			}
			cour[j] = min(prec[j]+1, cour[j-1]+1, sub)
		}
		prec = cour
	}
	return prec[len(y)]
}
