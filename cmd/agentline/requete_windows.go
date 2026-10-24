package main

import (
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// enregistrementConsole : un INPUT_RECORD de la console, lu comme un KEY_EVENT_RECORD (20 octets).
type enregistrementConsole struct {
	Type       uint16
	_          uint16
	Enfoncee   int32
	Repetition uint16
	Touche     uint16
	Scan       uint16
	Caractere  uint16
	Etat       uint32
}

var lireEntreeConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

// interrogerTerminal : envoie la requête à la console (CONOUT$) et lit sa réponse dans CONIN$ (entrée en
// séquences VT, sans écho ni attente de ligne), jusqu'à ce que fin la reconnaisse ou que le délai passe ;
// faux sans console. ReadConsoleInputW ne lit que ce qui est déjà là (après WaitForSingleObject) : rien
// ne reste bloqué si le terminal ne répond pas.
func interrogerTerminal(requete string, delai time.Duration, fin func(string) bool) (string, bool) {
	entree, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return "", false
	}
	defer entree.Close()
	sortie, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		return "", false
	}
	defer sortie.Close()
	hin, hout := windows.Handle(entree.Fd()), windows.Handle(sortie.Fd())
	var modeIn, modeOut uint32
	if windows.GetConsoleMode(hin, &modeIn) != nil || windows.GetConsoleMode(hout, &modeOut) != nil {
		return "", false
	}
	if windows.SetConsoleMode(hin, windows.ENABLE_VIRTUAL_TERMINAL_INPUT) != nil {
		return "", false // console trop ancienne
	}
	defer windows.SetConsoleMode(hin, modeIn)
	if windows.SetConsoleMode(hout, modeOut|windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
		return "", false
	}
	defer windows.SetConsoleMode(hout, modeOut)
	if _, err := sortie.WriteString(requete); err != nil {
		return "", false
	}
	var lu []rune
	var e [32]enregistrementConsole
	limite := time.Now().Add(delai)
	for !fin(string(lu)) {
		reste := time.Until(limite)
		if reste <= 0 {
			break
		}
		if r, err := windows.WaitForSingleObject(hin, uint32(reste.Milliseconds())+1); err != nil || r != windows.WAIT_OBJECT_0 {
			break
		}
		var n uint32
		if r, _, _ := lireEntreeConsole.Call(uintptr(hin), uintptr(unsafe.Pointer(&e[0])), uintptr(len(e)), uintptr(unsafe.Pointer(&n))); r == 0 {
			break
		}
		for _, x := range e[:n] {
			if x.Type == 1 && x.Enfoncee != 0 && x.Caractere != 0 { // KEY_EVENT, touche enfoncée
				lu = append(lu, rune(x.Caractere))
			}
		}
	}
	return string(lu), len(lu) > 0
}
