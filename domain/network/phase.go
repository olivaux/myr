// domain/network/phase.go
package network

import "errors"

// ErrNoIdentity indique que le réseau est connu mais qu'aucune identité client
// n'est disponible. L'utilisateur doit s'enrôler via le CA.
var ErrNoIdentity = errors.New("aucune identité disponible")

// ErrNoChaincode indique que le nom du chaincode n'est pas renseigné dans le
// profil réseau. L'utilisateur doit l'ajouter via le formulaire d'édition.
var ErrNoChaincode = errors.New("chaincode non renseigné")
