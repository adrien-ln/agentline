//go:build !windows

package installation

// RegistreReel : pas de registre hors de Windows.
func RegistreReel() Registre { return nil }

// DiffuserReel : rien à prévenir hors de Windows.
func DiffuserReel(string) bool { return false }
