package i18n

import "golang.org/x/sys/windows"

// langueSysteme : la première langue d'interface de l'utilisateur (« fr-FR »), vide si elle est inconnue.
func langueSysteme() string {
	l, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil || len(l) == 0 {
		return ""
	}
	return l[0]
}
