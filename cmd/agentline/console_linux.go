package main

import (
	"os"

	"golang.org/x/sys/unix"
)

const lireTermios, ecrireTermios = unix.TCGETS, unix.TCSETS

// console : vrai si f est un terminal (/dev/null n'en est pas un).
func console(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}
