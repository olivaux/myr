// domain/identity/errors.go
package identity

import "errors"

// ErrNoAdminCredentials est retourné quand l'opération CA nécessite les
// identifiants administrateur (cert + clé) qui ne sont pas configurés dans
// le profil réseau.
//
// Pour corriger : éditez le réseau (touche e dans le menu) et renseignez
// les champs "Cert path" et "Key path" avec le certificat et la clé de
// l'administrateur CA.
var ErrNoAdminCredentials = errors.New(
	"l'enregistrement CA requiert les identifiants admin — " +
		"éditez le réseau (e) et renseignez Cert path + Key path",
)

// IsNoAdminCredentials rapporte si err est ou enveloppe ErrNoAdminCredentials.
func IsNoAdminCredentials(err error) bool {
	return errors.Is(err, ErrNoAdminCredentials)
}
