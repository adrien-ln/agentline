package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// console : vrai si f est une console Windows (NUL n'en est pas une).
func console(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}
