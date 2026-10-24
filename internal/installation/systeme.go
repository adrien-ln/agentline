package installation

// Registre : les valeurs de HKEY_CURRENT_USER (Windows). Injecté : les tests passent une table en mémoire,
// « agentline install » le vrai registre (RegistreReel).
type Registre interface {
	// Lire : la valeur nom de la clé (chemin sous HKCU, « Environment ») ; faux si elle est absente.
	Lire(cle, nom string) (ValeurRegistre, bool, error)
	// Ecrire : pose la valeur (la clé est créée au besoin).
	Ecrire(cle, nom string, v ValeurRegistre) error
}

// ValeurRegistre : une valeur texte du registre et son type.
type ValeurRegistre struct {
	Texte  string
	Etendu bool // REG_EXPAND_SZ (les %VARIABLES% y sont développées), sinon REG_SZ
}

// cleEnvironnement : les variables d'environnement de l'utilisateur (sans droits d'administrateur).
const cleEnvironnement = "Environment"
