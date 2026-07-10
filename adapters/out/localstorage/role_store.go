// adapters/out/localstorage/role_store.go
// Implémente domain/role.Repo via un fichier JSON local (roles.json).
package localstorage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"myr/domain/role"
)

var _ role.Repo = (*JSONRoleStore)(nil)

type roleDB struct {
	Roles []*role.Role `json:"roles"`
}

// JSONRoleStore persiste les rôles RBAC dans un fichier JSON. Thread-safe.
type JSONRoleStore struct {
	path string
	mu   sync.RWMutex
}

func NewJSONRoleStore(path string) *JSONRoleStore {
	return &JSONRoleStore{path: path}
}

func (s *JSONRoleStore) load() *roleDB {
	var db roleDB
	data, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(data, &db) //nolint:errcheck — fichier corrompu traité comme vide
	}
	if db.Roles == nil {
		db.Roles = []*role.Role{}
	}
	return &db
}

func (s *JSONRoleStore) save(db *roleDB) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("role_store marshal : %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}

// Save persiste ou met à jour un rôle (upsert par Name).
func (s *JSONRoleStore) Save(r *role.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	for i, existing := range db.Roles {
		if existing.Name == r.Name {
			db.Roles[i] = r
			return s.save(db)
		}
	}
	db.Roles = append(db.Roles, r)
	return s.save(db)
}

func (s *JSONRoleStore) FindAll() ([]*role.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.load().Roles, nil
}

func (s *JSONRoleStore) FindByName(name string) (*role.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db := s.load()
	for _, r := range db.Roles {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (s *JSONRoleStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := s.load()
	filtered := db.Roles[:0]
	found := false
	for _, r := range db.Roles {
		if r.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("rôle %q introuvable", name)
	}
	db.Roles = filtered
	return s.save(db)
}
