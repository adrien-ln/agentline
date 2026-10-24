//go:build darwin || freebsd || netbsd || openbsd

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

const lireTermios, ecrireTermios = unix.TIOCGETA, unix.TIOCSETA

// console : vrai si f est un terminal (/dev/null n'en est pas un).
func console(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)
	return err == nil
}
