// domain/session/ports.go
package session

// Store est le contrat de persistance de la session active.
type Store interface {
	Save(s *Session) error
	Load() (*Session, error) // retourne nil, nil si aucune session
	Clear() error
}
