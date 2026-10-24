//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd

package main

import (
	"os"
	"time"
)

// console : sans moyen de le vérifier ici, tout périphérique caractère passe pour un terminal.
func console(*os.File) bool { return true }

// interrogerTerminal : rien à interroger sur ces systèmes.
func interrogerTerminal(string, time.Duration, func(string) bool) (string, bool) { return "", false }
