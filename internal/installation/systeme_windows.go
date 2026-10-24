//go:build windows

package installation

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type registreWindows struct{}

// RegistreReel : HKEY_CURRENT_USER de cette machine.
func RegistreReel() Registre { return registreWindows{} }

func (registreWindows) Lire(cle, nom string) (ValeurRegistre, bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, cle, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return ValeurRegistre{}, false, nil
	}
	if err != nil {
		return ValeurRegistre{}, false, err
	}
	defer k.Close()
	s, t, err := k.GetStringValue(nom) // sans développer les %VARIABLES%
	if errors.Is(err, registry.ErrNotExist) {
		return ValeurRegistre{}, false, nil
	}
	if err != nil {
		return ValeurRegistre{}, false, err
	}
	return ValeurRegistre{Texte: s, Etendu: t == registry.EXPAND_SZ}, true, nil
}

func (registreWindows) Ecrire(cle, nom string, v ValeurRegistre) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, cle, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if v.Etendu {
		return k.SetExpandStringValue(nom, v.Texte)
	}
	return k.SetStringValue(nom, v.Texte)
}

// DiffuserReel : prévient les programmes ouverts (l'Explorateur, qui lance les nouveaux terminaux) que
// les variables d'environnement ou les polices ont changé ; quoi : "Environment" ou "polices".
func DiffuserReel(quoi string) bool {
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
		wmFontChange    = 0x001D
		smtoAbortIfHung = 0x0002
	)
	envoyer := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW")
	if envoyer.Find() != nil {
		return false
	}
	var resultat uintptr
	if quoi == "polices" {
		r, _, _ := envoyer.Call(hwndBroadcast, wmFontChange, 0, 0, smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&resultat)))
		return r != 0
	}
	p, err := syscall.UTF16PtrFromString(quoi)
	if err != nil {
		return false
	}
	r, _, _ := envoyer.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(p)), smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&resultat)))
	return r != 0
}
