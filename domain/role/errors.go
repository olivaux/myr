package role

import "errors"

var (
	ErrNotFound          = errors.New("role: introuvable")
	ErrAlreadyExists     = errors.New("role: existe déjà")
	ErrBuiltIn           = errors.New("role: rôle intégré — non modifiable ni supprimable")
	ErrInvalidPermission = errors.New("role: permission inconnue")
	ErrInvalidName       = errors.New("role: nom invalide")
)
