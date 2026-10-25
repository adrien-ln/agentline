package selecteur

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestScenesRealistes(t *testing.T) {
	sc := Scenes()
	if len(sc) < 3 {
		t.Fatalf("%d scènes, attendu au moins 3", len(sc))
	}
	fautes := 0
	for _, s := range sc {
		if s.duree < 10000 || s.duree > 40000 {
			t.Errorf("scène %s : %d ms, attendu entre 10 et 40 s", s.nom, s.duree)
		}
		// Frappe humaine : ni trop lente ni trop rapide, et irrégulière (pas de rythme mécanique)
		moy, n := 0.0, float64(len(s.frappe))
		for _, d := range s.frappe {
			if d < 45 || d > 650 {
				t.Errorf("scène %s : délai de frappe de %d ms hors de [45, 650]", s.nom, d)
			}
			moy += float64(d) / n
		}
		ecart := 0.0
		for _, d := range s.frappe {
			ecart += (float64(d) - moy) * (float64(d) - moy) / n
		}
		if math.Sqrt(ecart) < 15 {
			t.Errorf("scène %s : frappe trop régulière (écart type %.0f ms)", s.nom, math.Sqrt(ecart))
		}
		// Fautes corrigées : la saisie raccourcit pendant la frappe ; réflexion d'au moins 1,5 s
		avant, debutReflexion := etatScene{reflexion: -1}, -1
		for _, ev := range s.evts {
			e := s.etatA(ev.t)
			if len(e.saisie) < len(avant.saisie) && e.saisie != "" {
				fautes++
			}
			if e.reflexion >= 0 && debutReflexion < 0 {
				debutReflexion = ev.t
			}
			if e.reflexion < 0 && debutReflexion >= 0 {
				if ev.t-debutReflexion < 1500 {
					t.Errorf("scène %s : réflexion de %d ms, attendu au moins 1,5 s", s.nom, ev.t-debutReflexion)
				}
				debutReflexion = -1
			}
			avant = e
		}
		// La scène finit sur une réponse complète
		fin := s.etatA(s.duree)
		var reponse string
		for _, l := range fin.lignes {
			if strings.Contains(l, "⏺") && !strings.Contains(l, "(") {
				reponse = l
			}
		}
		if len(couleurs.ReplaceAllString(reponse, "")) < 20 || fin.saisie != "" || fin.reflexion >= 0 {
			t.Errorf("scène %s : fin incohérente (réponse %q, saisie %q)", s.nom, reponse, fin.saisie)
		}
	}
	if fautes == 0 {
		t.Error("aucune faute de frappe corrigée dans les scènes")
	}
	// Déterministe : même déroulé à chaque ouverture
	for k, s := range Scenes() {
		if s.duree != sc[k].duree || len(s.evts) != len(sc[k].evts) {
			t.Errorf("scène %s : déroulé différent d'une exécution à l'autre", s.nom)
		}
	}
}

func TestAnimationBorduresAlignees(t *testing.T) {
	for _, l := range []int{110, 70} {
		m, _ := nouveau(t)
		m.Update(tea.WindowSizeMsg{Width: l, Height: 40})
		total := 0
		for _, s := range m.scenes {
			total += s.duree
		}
		for ms := 0; ms < total; ms += 200 {
			m.t = ms
			if w := lignesFenetre(ecran(m)); len(w) != 1 {
				t.Fatalf("largeur %d, t = %d ms : lignes de largeurs %v\n%s", l, ms, w, ecran(m))
			}
		}
	}
}

// instantOu : le premier instant (pas de 50 ms) où la session fictive vérifie la condition, Clawd n'étant ni
// en train de faire un geste ni de cligner des yeux.
func instantOu(m *Modele, ok func(etatScene) bool) int {
	for t := 0; t < 120000; t += 50 {
		sc, ts := sceneA(m.scenes, t)
		if g, _ := gesteA(t); g == "" && !cligneA(t) && ok(sc.etatA(ts)) {
			return t
		}
	}
	return -1
}

func sceneCalme(e etatScene) bool { return e.saisie == "" && e.reflexion < 0 }

// debutGeste : le premier instant où Clawd commence le geste nom au repos, la session fictive restant calme
// jusqu'à la fin du geste (sinon elle passe avant) ; -1 si aucun dans les dix premières minutes.
func debutGeste(m *Modele, nom string) int {
	duree := 0
	for _, g := range gestes {
		if g.nom == nom {
			duree = g.duree
		}
	}
	for t := 0; t < 600000; t++ {
		if g, ec := gesteA(t); g != nom || ec != 0 {
			continue
		}
		ok := true
		for u := t; u < t+duree && ok; u++ {
			sc, ts := sceneA(m.scenes, u)
			ok = sceneCalme(sc.etatA(ts))
		}
		if ok {
			return t
		}
	}
	return -1
}

// eveille : Clawd à l'instant t, sans réaction en cours ni sommeil.
func eveille(m *Modele, t int) { m.t, m.derniereTouche, m.reaction = t, t, "" }

func TestMascotteModes(t *testing.T) {
	m, _ := nouveau(t)
	eveille(m, instantOu(m, sceneCalme))
	l, pattes, ok := m.Clawd(m.noms[m.i])
	v := visible(strings.Join(l[:], "\n"))
	if !ok || !strings.Contains(v, " ▐▛███▛█ \n▝▜██████▀") || visible(pattes) != " ▝▝   ▝▝ " {
		t.Fatalf("Clawd absent ou incomplet :\n%s\npattes %q", v, pattes)
	}
	if !strings.Contains(m.View(), strings.ReplaceAll(pattes, " ", "─")) {
		t.Error("les pattes de Clawd doivent être posées sur le cadre de la fenêtre")
	}
	c := m.clawdCouleurs(m.noms[m.i])
	if c[0] == c[len(c)-1] {
		t.Error("dégradé : la première et la dernière colonne devraient différer")
	}
	appuyer(m, "m")
	if c := m.clawdCouleurs(m.noms[m.i]); c[0] != c[len(c)-1] || !strings.Contains(m.View(), "m mascotte (unie)") {
		t.Error("mode uni : Clawd doit être d'une seule couleur")
	}
	appuyer(m, "m")
	if _, _, ok := m.Clawd(m.noms[m.i]); ok || strings.Contains(ecran(m), "▝▜██████▀") {
		t.Error("mode aucune : Clawd doit disparaître")
	}
	appuyer(m, "m")
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 18}) // trop bas : la fenêtre passe avant Clawd
	if strings.Contains(ecran(m), "▝▜██████▀") {
		t.Error("terminal trop bas : Clawd doit laisser la place à la fenêtre")
	}
}

func TestMascotteHumeurs(t *testing.T) {
	m, _ := nouveau(t)
	calme := instantOu(m, sceneCalme)
	pense := instantOu(m, func(e etatScene) bool { return e.reflexion >= 0 })
	tape := instantOu(m, func(e etatScene) bool { return e.saisie != "" })
	if calme < 0 || pense < 0 || tape < 0 {
		t.Fatal("la session fictive doit avoir des moments calmes, de réflexion et de frappe")
	}

	eveille(m, pense)
	if h := m.humeur(); h != "pensif" {
		t.Errorf("Claude réfléchit : %s, attendu pensif", h)
	}
	eveille(m, tape)
	if h := m.humeur(); h != "concentre" {
		t.Errorf("la frappe : %s, attendu concentre", h)
	}

	eveille(m, calme)
	appuyer(m, "↓")
	l, pattes, _ := m.Clawd(m.noms[m.i])
	if h := m.humeur(); h != "content" || pattes != "" || !strings.HasPrefix(visible(l[0]), "▗▟") || !strings.Contains(visible(l[2]), "▝▝") {
		t.Errorf("changement de thème : %s, il doit sauter les bras levés (pattes en l'air) :\n%s", h, visible(strings.Join(l[:], "\n")))
	}
	m.t += 1200
	m.derniereTouche = m.t
	if h := m.humeur(); h == "content" {
		t.Error("la joie ne doit pas durer")
	}

	eveille(m, calme)
	appuyer(m, "↓")
	m.t += dureeSaut + dureeAtterrissage/2
	if l, p, _ := m.Clawd(m.noms[m.i]); p != "" || strings.TrimSpace(visible(l[0])) == "" || strings.HasPrefix(visible(l[0]), "▗▟") {
		t.Errorf("atterrissage : il doit passer par la mi-hauteur :\n%s\n%q", visible(strings.Join(l[:], "\n")), visible(p))
	}
	m.t += dureeAtterrissage
	if _, p, _ := m.Clawd(m.noms[m.i]); p == "" {
		t.Error("après l'atterrissage, les pattes doivent reposer sur le cadre")
	}
	appuyer(m, "↓")
	if _, p, _ := m.Clawd(m.noms[m.i]); m.humeur() != "suit" || p == "" {
		t.Errorf("navigation rapide : %s, attendu qu'il suive la liste des yeux sans ressauter", m.humeur())
	}

	eveille(m, calme)
	appuyer(m, "→")
	if h := m.humeur(); h != "surpris" {
		t.Errorf("changement de filtre : %s, attendu surpris", h)
	}

	m, _ = nouveau(t)
	eveille(m, calme)
	appuyer(m, "entrée")
	pas := map[string]bool{}
	for k := 0; k < 4; k++ {
		_, p, _ := m.Clawd(m.noms[m.i])
		pas[visible(p)] = true
		m.t += pasDanse
	}
	if h := m.humeur(); h != "danse" || !pas[" ▝▝      "] || !pas["      ▝▝ "] {
		t.Errorf("validation : %s, attendu une danse d'une patte sur l'autre (%v)", h, pas)
	}
	appuyer(m, "échap")
	if h := m.humeur(); h != "clin" {
		t.Errorf("annulation : %s, attendu un clin d'œil", h)
	}

	m, _ = nouveau(t)
	eveille(m, calme)
	appuyer(m, "/", "z", "z", "z", "z")
	if l, _, _ := m.Clawd(m.noms[m.i]); m.humeur() != "triste" || visible(l[1]) != " ▐██████ " {
		t.Errorf("aucun résultat : %s, attendu triste, de dos : %q", m.humeur(), visible(l[1]))
	}

	m, _ = nouveau(t)
	eveille(m, calme)
	m.t = calme + sommeil - dureeSomnolence + 1
	if h := m.humeur(); h != "baille" {
		t.Errorf("juste avant de s'endormir : %s, attendu un bâillement", h)
	}
	m.t += dureeBaillement
	if h := m.humeur(); h != "somnole" {
		t.Errorf("après le bâillement : %s, attendu les yeux qui se ferment", h)
	}
	m.t = calme + sommeil + 1000
	if l, _, _ := m.Clawd(m.noms[m.i]); m.humeur() != "endormi" || !strings.ContainsAny(couleurs.ReplaceAllString(l[0], ""), "zZ") {
		t.Errorf("20 s sans touche : %s, attendu endormi avec des z", m.humeur())
	}
	appuyer(m, "x")
	if h := m.humeur(); h != "reveil" {
		t.Errorf("une touche pendant le sommeil : %s, attendu le sursaut du réveil", h)
	}
}

func TestMascotteTour(t *testing.T) {
	m, _ := nouveau(t)
	t0 := debutGeste(m, "tour")
	if t0 < 0 {
		t.Fatal("aucun tour sur lui-même en dix minutes")
	}
	vues := map[string]bool{}
	for k := 0; k < len(tour); k++ {
		eveille(m, t0+k*pasTour)
		l, p, _ := m.Clawd(m.noms[m.i])
		vues[visible(l[1]+"|"+l[2]+"|"+p)] = true
		if m.humeur() != "tour" {
			t.Fatalf("instant %d : %s, attendu tour", m.t, m.humeur())
		}
	}
	if len(vues) != len(tour) || !vues["   ▐██   |   ▐██   |   ▝ ▝   "] {
		t.Errorf("le tour doit passer par ses %d poses, dont le profil : %v", len(tour), vues)
	}
	eveille(m, t0+len(tour)*pasTour)
	if m.humeur() == "tour" {
		t.Error("le tour doit finir de face")
	}
}

// ecarts : moyenne, écart type relatif et nombre de valeurs distinctes des écarts entre instants successifs.
func ecarts(instants []int) (moy, cv float64, distincts int) {
	vus := map[int]bool{}
	n := float64(len(instants) - 1)
	for k := 1; k < len(instants); k++ {
		d := instants[k] - instants[k-1]
		vus[d] = true
		moy += float64(d) / n
	}
	for k := 1; k < len(instants); k++ {
		d := float64(instants[k]-instants[k-1]) - moy
		cv += d * d / n
	}
	return moy, math.Sqrt(cv) / moy, len(vus)
}

const demiHeure = 30 * 60 * 1000

func TestGestesAuRepos(t *testing.T) {
	var debuts []int
	var noms []string
	compte := map[string]int{}
	for u := 0; u < demiHeure; u++ {
		if g, ec := gesteA(u); g != "" && ec == 0 {
			debuts, noms = append(debuts, u), append(noms, g)
			compte[g]++
		}
	}
	if len(compte) != len(gestes) {
		t.Errorf("en une demi-heure, tous les gestes doivent apparaître : %v", compte)
	}
	for g, n := range compte {
		if 3*n > len(noms) {
			t.Errorf("« %s » revient trop souvent : %d gestes sur %d", g, n, len(noms))
		}
	}
	for k := 1; k < len(noms); k++ {
		if noms[k] == noms[k-1] {
			t.Errorf("deux « %s » d'affilée (%d puis %d ms)", noms[k], debuts[k-1], debuts[k])
		}
	}
	// ni figé ni agité, et surtout pas mécanique : des écarts variés autour de quelques secondes
	moy, cv, distincts := ecarts(debuts)
	if moy < 2000 || moy > 10000 || cv < 0.2 || 2*distincts < len(debuts) {
		t.Errorf("écarts entre gestes : moyenne %.0f ms, variation %.2f, %d distincts sur %d", moy, cv, distincts, len(debuts))
	}
	// même instant, même geste, dans n'importe quel ordre de calcul
	for k := len(debuts) - 1; k >= 0; k-- {
		if g, ec := gesteA(debuts[k]); g != noms[k] || ec != 0 {
			t.Fatalf("instant %d : %s (%d ms), attendu le début de %s", debuts[k], g, ec, noms[k])
		}
	}
}

func TestClignements(t *testing.T) {
	var debuts []int // début de chaque clignement, double ou non
	doubles, ferme, dernier := 0, false, -10000
	for u := 0; u < demiHeure; u += 10 {
		c := cligneA(u)
		if c && !ferme {
			if u-dernier < 500 {
				doubles++
			} else {
				debuts = append(debuts, u)
			}
			dernier = u
		}
		ferme = c
	}
	if doubles == 0 || 2*doubles > len(debuts) {
		t.Errorf("%d doubles clignements sur %d : il en faut, mais pas trop", doubles, len(debuts))
	}
	moy, cv, distincts := ecarts(debuts)
	if moy < 2000 || moy > 6000 || cv < 0.2 || 2*distincts < len(debuts) {
		t.Errorf("écarts entre clignements : moyenne %.0f ms, variation %.2f, %d distincts sur %d", moy, cv, distincts, len(debuts))
	}
	for k := 1; k < len(debuts); k++ {
		if debuts[k]-debuts[k-1] < 2*dureeCligne+500 {
			t.Errorf("clignements trop rapprochés : %d puis %d ms", debuts[k-1], debuts[k])
		}
	}

	m, _ := nouveau(t)
	for u := 0; u < demiHeure; u += 50 {
		sc, ts := sceneA(m.scenes, u)
		if g, _ := gesteA(u); g == "" && cligneA(u) && sceneCalme(sc.etatA(ts)) {
			eveille(m, u)
			if l, _, _ := m.Clawd(m.noms[m.i]); visible(l[1]) != " ▐▂███▂█ " {
				t.Errorf("clignement : %q, attendu les paupières baissées", visible(l[1]))
			}
			return
		}
	}
	t.Error("aucun clignement au calme")
}

// demi-ligne : chaque bloc passe pour moitié dans la ligne du dessus, sans rien perdre du dessin
func TestDemiLigne(t *testing.T) {
	somme := func(ls ...string) (n int) {
		for _, l := range ls {
			for _, r := range l {
				if r == paupiereHaute {
					r = paupiere
				}
				p := points(r)
				for ; p > 0; p >>= 1 {
					n += p & 1
				}
			}
		}
		return n
	}
	for _, y := range []string{"ouverts", "fermés"} {
		d := dessiner("bas", y, "deux")
		l, p := poser(d, 1)
		if p != "" || somme(l[:]...) != somme(d[:]...) {
			t.Errorf("yeux %s, à mi-hauteur : pattes %q, %d points au lieu de %d :\n%s", y, p, somme(l[:]...), somme(d[:]...), strings.Join(l[:], "\n"))
		}
		if y == "fermés" && !strings.ContainsRune(l[1], paupiereHaute) {
			t.Errorf("les paupières doivent suivre le corps à mi-hauteur :\n%s", strings.Join(l[:], "\n"))
		}
	}

	m, _ := nouveau(t)
	t0 := debutGeste(m, "saute")
	if t0 < 0 {
		t.Fatal("aucun petit saut en dix minutes")
	}
	pose, mi, enLair := false, false, false
	for u := t0; u < t0+1000; u += 50 {
		eveille(m, u)
		l, p, _ := m.Clawd(m.noms[m.i])
		switch haut := strings.TrimSpace(visible(l[0])); {
		case p != "":
			pose = true
		case strings.HasPrefix(haut, "▗▟"):
			enLair = true
		case haut != "":
			mi = true
		}
	}
	if !pose || !mi || !enLair {
		t.Errorf("le petit saut doit passer par mi-hauteur (%v), en l'air (%v) et retomber (%v)", mi, enLair, pose)
	}
}

func visible(s string) string { return couleurs.ReplaceAllString(s, "") }

func TestAnimationTic(t *testing.T) {
	m, _ := nouveau(t)
	if m.Init() == nil {
		t.Fatal("Init doit lancer l'animation")
	}
	if _, cmd := m.Update(tic(m.debut.Add(1234 * time.Millisecond))); cmd == nil || m.t != 1234 {
		t.Errorf("une image doit faire avancer le temps (t = %d) et demander la suivante", m.t)
	}
}

func BenchmarkImageAnimee(b *testing.B) {
	m, _ := nouveau(&testing.T{})
	for n := 0; n < b.N; n++ {
		m.t = (n * 50) % 60000
		m.View()
	}
}

// TestAnimationEndormie : quand la mascotte dort, l'animation ralentit (l'image suivante vient plus tard) ;
// une touche la réveille et le rythme normal reprend à l'image suivante.
func TestAnimationEndormie(t *testing.T) {
	m, _ := nouveau(t)
	m.Init()
	attente := func(cmd tea.Cmd) time.Duration {
		debut := time.Now()
		cmd()
		return time.Since(debut)
	}
	_, cmd := m.Update(tic(m.debut.Add(1000 * time.Millisecond)))
	if d := attente(cmd); d > 300*time.Millisecond {
		t.Errorf("éveillé : image suivante après %v", d)
	}
	_, cmd = m.Update(tic(m.debut.Add(time.Duration(sommeil+1000) * time.Millisecond)))
	if !m.dort() {
		t.Fatal("la mascotte doit dormir")
	}
	if d := attente(cmd); d < 300*time.Millisecond {
		t.Errorf("endormi : image suivante après %v seulement", d)
	}
}
