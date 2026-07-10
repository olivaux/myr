// adapters/out/localstorage/session_store.go
package localstorage

import (
	"encoding/json"
	"errors"
	"os"

	"myr/domain/session"
)

var _ session.Store = (*JSONSessionStore)(nil)

type JSONSessionStore struct {
	path string
}

func NewJSONSessionStore(path string) *JSONSessionStore {
	return &JSONSessionStore{path: path}
}

func (s *JSONSessionStore) Save(sess *session.Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600) // 0600 : lisible uniquement par le user
}

func (s *JSONSessionStore) Load() (*session.Session, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil // pas de session active
	}
	if err != nil {
		return nil, err
	}
	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *JSONSessionStore) Clear() error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
