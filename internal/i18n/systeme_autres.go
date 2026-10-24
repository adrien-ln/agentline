//go:build !windows

package i18n

// langueSysteme : hors Windows, la langue vient des variables POSIX (LC_ALL, LC_MESSAGES, LANG).
func langueSysteme() string { return "" }
