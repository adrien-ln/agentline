package render

import (
	"os"
	"path/filepath"
	"strings"
)

// signe : l'icône Nerd Font d'un langage ou d'une forge, et sa couleur propre (project.color=brand).
type signe struct{ icone, couleur string }

// langages : le fichier qui signe le langage d'un projet, à sa racine, dans l'ordre où on les cherche
// (tsconfig.json avant package.json, build.gradle.kts avant build.gradle).
var langages = []struct {
	fichier string
	signe
}{
	{"go.mod", signe{"", "00add8"}},
	{"Cargo.toml", signe{"", "dea584"}},
	{"tsconfig.json", signe{"", "3178c6"}},
	{"package.json", signe{"", "8cc84b"}},
	{"pyproject.toml", signe{"", "4b8bbe"}},
	{"requirements.txt", signe{"", "4b8bbe"}},
	{"setup.py", signe{"", "4b8bbe"}},
	{"Gemfile", signe{"", "cc342d"}},
	{"build.gradle.kts", signe{"", "7f52ff"}},
	{"pom.xml", signe{"", "e76f00"}},
	{"build.gradle", signe{"", "e76f00"}},
	{"composer.json", signe{"", "777bb4"}},
	{"Package.swift", signe{"", "f05138"}},
	{"pubspec.yaml", signe{"", "0175c2"}},
	{"mix.exs", signe{"", "6e4a7e"}},
	{"build.zig", signe{"", "f7a41d"}},
	{"CMakeLists.txt", signe{"", "659ad2"}},
}

// langageDe : le langage du projet dont la racine est dir (faux : aucun fichier connu).
func langageDe(dir string) (signe, bool) {
	if dir == "" {
		return signe{}, false
	}
	for _, l := range langages {
		if _, err := os.Stat(filepath.Join(filepath.FromSlash(dir), l.fichier)); err == nil {
			return l.signe, true
		}
	}
	return signe{}, false
}

// forges : la forge d'une pull request, d'après l'hôte de son URL.
var forges = []struct {
	hote string
	signe
}{
	{"github", signe{"", ""}}, // couleur propre : le texte (le logo de GitHub est noir ou blanc)
	{"gitlab", signe{"", "fc6d26"}},
	{"bitbucket", signe{"", "2684ff"}},
}

// forgeDe : la forge de l'URL (faux : inconnue ou absente).
func forgeDe(url string) (signe, bool) {
	hote := url
	if _, apres, ok := strings.Cut(url, "://"); ok {
		hote = apres
	}
	hote, _, _ = strings.Cut(hote, "/")
	for _, f := range forges {
		if strings.Contains(strings.ToLower(hote), f.hote) {
			return f.signe, true
		}
	}
	return signe{}, false
}
