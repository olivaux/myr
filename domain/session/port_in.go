// domain/session/port_in.go — port d'entrée : interface exposée aux adapters entrants
package session

// SessionService est le port d'entrée du domaine session.
type SessionService interface {
	Create(name, orgID string) (*Session, error)
	Current() (*Session, error)
	Logout() error
}
