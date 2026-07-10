// domain/session/service.go
package session

import (
	"fmt"
	"time"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create crée et persiste un compte local (premier lancement).
func (s *Service) Create(name, orgID string) (*Session, error) {
	if name == "" {
		return nil, fmt.Errorf("le nom est requis")
	}
	sess := &Session{
		Name:      name,
		OrgID:     orgID,
		CreatedAt: time.Now(),
	}
	if err := s.store.Save(sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// Current retourne la session active, ou nil si aucune.
func (s *Service) Current() (*Session, error) {
	return s.store.Load()
}

// Logout efface la session courante.
func (s *Service) Logout() error {
	return s.store.Clear()
}
