//go:build linux || darwin || freebsd || netbsd || openbsd

package main

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// interrogerTerminal : envoie la requête au terminal (/dev/tty) et lit sa réponse, sans écho ni attente
// de ligne, jusqu'à ce que fin la reconnaisse ou que le délai passe ; faux sans terminal.
func interrogerTerminal(requete string, delai time.Duration, fin func(string) bool) (string, bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", false
	}
	defer tty.Close()
	fd := int(tty.Fd())
	avant, err := unix.IoctlGetTermios(fd, lireTermios)
	if err != nil {
		return "", false
	}
	brut := *avant
	brut.Lflag &^= unix.ICANON | unix.ECHO
	brut.Cc[unix.VMIN], brut.Cc[unix.VTIME] = 1, 0
	if err := unix.IoctlSetTermios(fd, ecrireTermios, &brut); err != nil {
		return "", false
	}
	defer unix.IoctlSetTermios(fd, ecrireTermios, avant)
	if _, err := tty.WriteString(requete); err != nil {
		return "", false
	}
	var lu []byte
	b := make([]byte, 256)
	limite := time.Now().Add(delai)
	for !fin(string(lu)) {
		reste := time.Until(limite)
		if reste <= 0 {
			break
		}
		p := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if n, err := unix.Poll(p, int(reste.Milliseconds())+1); err != nil || n == 0 {
			if err == unix.EINTR {
				continue
			}
			break
		}
		n, err := unix.Read(fd, b)
		if err != nil || n <= 0 {
			break
		}
		lu = append(lu, b[:n]...)
	}
	return string(lu), len(lu) > 0
}
